// Package apns sends push notifications directly to Apple's APNs service
// using token-based (ES256 JWT) authentication over HTTP/2.
package apns

import (
	"bytes"
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chrisgregori/boop/server/internal/config"
)

// Hosts for the two APNs environments.
const (
	HostProduction = "https://api.push.apple.com"
	HostSandbox    = "https://api.sandbox.push.apple.com"
)

// tokenLifetime is how long a provider JWT is reused. Apple accepts tokens
// for up to an hour and asks that they be refreshed no more than every 20 min.
const tokenLifetime = 50 * time.Minute

// Notification is the alert to deliver.
type Notification struct {
	Title     string
	Body      string
	EventID   string
	ProjectID string
	// Prominent asks for a time-sensitive, high-priority alert (critical events).
	Prominent bool
}

// Error is a non-2xx response from APNs.
type Error struct {
	Status int
	Reason string
	APNSID string
}

func (e *Error) Error() string {
	return fmt.Sprintf("apns: %d %s", e.Status, e.Reason)
}

// Unregistered reports whether the device token is no longer valid and
// should be dropped.
func (e *Error) Unregistered() bool {
	return e.Status == 410 || e.Reason == "BadDeviceToken" || e.Reason == "Unregistered" || e.Reason == "DeviceTokenNotForTopic"
}

// Retryable reports whether a retry might succeed.
func (e *Error) Retryable() bool {
	return e.Status >= 500 || e.Status == 429 || e.Reason == "ExpiredProviderToken"
}

// Client sends notifications.
type Client struct {
	teamID, keyID, bundleID string
	key                     *ecdsa.PrivateKey
	host                    string
	http                    *http.Client

	mu        sync.Mutex
	token     string
	tokenTime time.Time
	lastPush  Push
}

// Push records the outcome of the most recent send attempt (for the status page).
type Push struct {
	At      time.Time `json:"at"`
	OK      bool      `json:"ok"`
	Error   string    `json:"error,omitempty"`
	EventID string    `json:"event_id,omitempty"`
}

// New builds a Client from configuration. It returns an error describing the
// first problem found so the status page can surface it.
func New(cfg config.APNS) (*Client, error) {
	if missing := cfg.Missing(); len(missing) > 0 {
		return nil, fmt.Errorf("missing %s", strings.Join(missing, ", "))
	}
	pemBytes := []byte(strings.TrimSpace(cfg.PrivateKey))
	// APNS_PRIVATE_KEY may hold the PEM text or its base64 (handy for single-line env editors).
	if len(pemBytes) > 0 && !strings.Contains(string(pemBytes), "-----BEGIN") {
		if decoded, err := base64.StdEncoding.DecodeString(string(pemBytes)); err == nil {
			pemBytes = decoded
		}
	}
	if cfg.PrivateKeyPath != "" {
		b, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("read APNS_PRIVATE_KEY_PATH: %w", err)
		}
		pemBytes = b
	}
	key, err := ParseKey(pemBytes)
	if err != nil {
		return nil, err
	}
	host := HostProduction
	if cfg.Environment == "sandbox" {
		host = HostSandbox
	}
	return NewWithKey(cfg.TeamID, cfg.KeyID, cfg.BundleID, key, host), nil
}

// NewWithKey builds a Client from an already-parsed key (tests).
func NewWithKey(teamID, keyID, bundleID string, key *ecdsa.PrivateKey, host string) *Client {
	return &Client{
		teamID: teamID, keyID: keyID, bundleID: bundleID, key: key, host: host,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

// ParseKey parses a PKCS#8 (.p8) ECDSA P-256 private key.
func ParseKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("APNs private key is not PEM encoded")
	}
	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse APNs private key: %w", err)
	}
	ec, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("APNs private key must be an ECDSA (ES256) key")
	}
	return ec, nil
}

// Host returns the APNs host in use.
func (c *Client) Host() string { return c.host }

// LastPush returns the outcome of the most recent send.
func (c *Client) LastPush() Push {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.lastPush
}

// Send delivers n to deviceToken, retrying once on transient failure.
// It returns the apns-id header on success.
func (c *Client) Send(ctx context.Context, deviceToken string, n Notification) (string, error) {
	id, err := c.send(ctx, deviceToken, n)
	var ae *Error
	if err != nil && errors.As(err, &ae) && ae.Retryable() {
		if ae.Reason == "ExpiredProviderToken" {
			c.mu.Lock()
			c.token = ""
			c.mu.Unlock()
		}
		select {
		case <-time.After(500 * time.Millisecond):
		case <-ctx.Done():
			return "", ctx.Err()
		}
		id, err = c.send(ctx, deviceToken, n)
	}
	c.mu.Lock()
	c.lastPush = Push{At: time.Now(), OK: err == nil, EventID: n.EventID}
	if err != nil {
		c.lastPush.Error = err.Error()
	}
	c.mu.Unlock()
	return id, err
}

func (c *Client) send(ctx context.Context, deviceToken string, n Notification) (string, error) {
	payload := map[string]any{
		"aps": map[string]any{
			"alert": map[string]string{"title": n.Title, "body": n.Body},
			"sound": "default",
		},
		"event_id":   n.EventID,
		"project_id": n.ProjectID,
	}
	if n.Prominent {
		payload["aps"].(map[string]any)["interruption-level"] = "time-sensitive"
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	jwt, err := c.providerToken()
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.host+"/3/device/"+deviceToken, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("authorization", "bearer "+jwt)
	req.Header.Set("apns-topic", c.bundleID)
	req.Header.Set("apns-push-type", "alert")
	req.Header.Set("apns-priority", "10")
	req.Header.Set("content-type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return "", &Error{Status: 503, Reason: err.Error()}
	}
	defer resp.Body.Close()
	apnsID := resp.Header.Get("apns-id")
	if resp.StatusCode == http.StatusOK {
		return apnsID, nil
	}
	var parsed struct {
		Reason string `json:"reason"`
	}
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	_ = json.Unmarshal(b, &parsed)
	if parsed.Reason == "" {
		parsed.Reason = http.StatusText(resp.StatusCode)
	}
	return "", &Error{Status: resp.StatusCode, Reason: parsed.Reason, APNSID: apnsID}
}

// providerToken returns a cached or freshly signed ES256 JWT.
func (c *Client) providerToken() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Since(c.tokenTime) < tokenLifetime {
		return c.token, nil
	}
	now := time.Now()
	header, _ := json.Marshal(map[string]string{"alg": "ES256", "kid": c.keyID})
	claims, _ := json.Marshal(map[string]any{"iss": c.teamID, "iat": now.Unix()})
	signing := b64(header) + "." + b64(claims)
	digest := sha256.Sum256([]byte(signing))
	r, s, err := ecdsa.Sign(rand.Reader, c.key, digest[:])
	if err != nil {
		return "", err
	}
	// JWS wants the raw fixed-width R||S concatenation, not ASN.1.
	sig := make([]byte, 64)
	r.FillBytes(sig[:32])
	s.FillBytes(sig[32:])
	c.token = signing + "." + b64(sig)
	c.tokenTime = now
	return c.token, nil
}

func b64(b []byte) string { return base64.RawURLEncoding.EncodeToString(b) }

var _ crypto.Signer = (*ecdsa.PrivateKey)(nil)
