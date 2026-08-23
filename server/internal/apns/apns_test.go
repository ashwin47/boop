package apns

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func testKey(t *testing.T) (*ecdsa.PrivateKey, []byte) {
	t.Helper()
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, _ := x509.MarshalPKCS8PrivateKey(k)
	return k, pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
}

func TestParseKey(t *testing.T) {
	_, pemBytes := testKey(t)
	if _, err := ParseKey(pemBytes); err != nil {
		t.Fatalf("ParseKey: %v", err)
	}
	if _, err := ParseKey([]byte("not a key")); err == nil {
		t.Fatal("expected error for garbage")
	}
}

func TestProviderTokenIsValidES256JWT(t *testing.T) {
	key, _ := testKey(t)
	c := NewWithKey("TEAM123", "KEY456", "com.example.Boop", key, "http://unused")
	tok, err := c.providerToken()
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(tok, ".")
	if len(parts) != 3 {
		t.Fatalf("jwt has %d parts", len(parts))
	}
	hdr, _ := base64.RawURLEncoding.DecodeString(parts[0])
	var h map[string]string
	_ = json.Unmarshal(hdr, &h)
	if h["alg"] != "ES256" || h["kid"] != "KEY456" {
		t.Errorf("header = %v", h)
	}
	cl, _ := base64.RawURLEncoding.DecodeString(parts[1])
	var claims map[string]any
	_ = json.Unmarshal(cl, &claims)
	if claims["iss"] != "TEAM123" || claims["iat"] == nil {
		t.Errorf("claims = %v", claims)
	}
	sig, _ := base64.RawURLEncoding.DecodeString(parts[2])
	if len(sig) != 64 {
		t.Fatalf("signature length %d, want 64 (raw R||S)", len(sig))
	}
	digest := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	r := new(bigInt).SetBytes(sig[:32])
	s := new(bigInt).SetBytes(sig[32:])
	if !ecdsa.Verify(&key.PublicKey, digest[:], r, s) {
		t.Fatal("signature does not verify")
	}
	// Cached on second call.
	tok2, _ := c.providerToken()
	if tok2 != tok {
		t.Error("token should be cached")
	}
}

func TestSendSuccessAndHeaders(t *testing.T) {
	key, _ := testKey(t)
	var gotPath, gotTopic, gotAuth string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotTopic = r.Header.Get("apns-topic")
		gotAuth = r.Header.Get("authorization")
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Header().Set("apns-id", "ABC-123")
		w.WriteHeader(200)
	}))
	defer srv.Close()
	c := NewWithKey("T", "K", "com.example.Boop", key, srv.URL)
	id, err := c.Send(context.Background(), "devtoken", Notification{Title: "Uini · KeyError", Body: "boom", EventID: "evt_1", ProjectID: "prj_1", Prominent: true})
	if err != nil {
		t.Fatal(err)
	}
	if id != "ABC-123" {
		t.Errorf("apns id = %q", id)
	}
	if gotPath != "/3/device/devtoken" || gotTopic != "com.example.Boop" || !strings.HasPrefix(gotAuth, "bearer ") {
		t.Errorf("request: path=%s topic=%s auth=%s", gotPath, gotTopic, gotAuth)
	}
	aps := gotBody["aps"].(map[string]any)
	if aps["alert"].(map[string]any)["title"] != "Uini · KeyError" || aps["interruption-level"] != "time-sensitive" {
		t.Errorf("aps = %v", aps)
	}
	if gotBody["event_id"] != "evt_1" || gotBody["project_id"] != "prj_1" {
		t.Errorf("custom keys = %v", gotBody)
	}
	if lp := c.LastPush(); !lp.OK || lp.EventID != "evt_1" {
		t.Errorf("last push = %+v", lp)
	}
}

func TestSendRetriesOn5xxAndReportsUnregistered(t *testing.T) {
	key, _ := testKey(t)
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`{"reason":"ServiceUnavailable"}`))
			return
		}
		w.WriteHeader(410)
		_, _ = w.Write([]byte(`{"reason":"Unregistered"}`))
	}))
	defer srv.Close()
	c := NewWithKey("T", "K", "b", key, srv.URL)
	_, err := c.Send(context.Background(), "tok", Notification{Title: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
	ae, ok := err.(*Error)
	if !ok || !ae.Unregistered() || ae.Status != 410 {
		t.Errorf("err = %v", err)
	}
	if atomic.LoadInt32(&calls) != 2 {
		t.Errorf("calls = %d, want 2 (one retry)", calls)
	}
}
