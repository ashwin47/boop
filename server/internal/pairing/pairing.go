// Package pairing issues short-lived, single-use tokens that an iOS client
// exchanges for a long-lived device credential.
package pairing

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/chrisgregori/boop/server/internal/auth"
	"github.com/chrisgregori/boop/server/internal/devices"
	"github.com/chrisgregori/boop/server/internal/ids"
)

// TTL is how long a pairing token stays valid.
const TTL = 10 * time.Minute

// ErrInvalidToken covers unknown, expired, used and revoked tokens.
var ErrInvalidToken = errors.New("pairing token is invalid or has expired")

// ErrNotFound is returned when revoking an unknown token.
var ErrNotFound = errors.New("pairing token not found")

// Token is a pending pairing. Raw is only set on creation.
type Token struct {
	ID        string `json:"id"`
	Raw       string `json:"token,omitempty"`
	ExpiresAt string `json:"expires_at"`
	CreatedAt string `json:"created_at"`
}

// QRPayload is the JSON the web UI encodes into the QR code.
type QRPayload struct {
	Version int    `json:"version"`
	Server  string `json:"server"`
	Token   string `json:"token"`
}

// Result is what the client receives after a successful exchange.
type Result struct {
	Device     devices.Device `json:"device"`
	Credential string         `json:"credential"`
}

// Store persists pairing tokens.
type Store struct {
	db      *sql.DB
	devices *devices.Store
	now     func() time.Time
}

// New returns a Store.
func New(db *sql.DB, d *devices.Store) *Store {
	return &Store{db: db, devices: d, now: time.Now}
}

// SetClock overrides the clock (tests).
func (s *Store) SetClock(now func() time.Time) { s.now = now }

// Create issues a new token.
func (s *Store) Create(ctx context.Context) (Token, error) {
	raw := auth.NewSecret(auth.PrefixPairing)
	now := s.now()
	t := Token{ID: ids.New("pair"), Raw: raw, ExpiresAt: ids.Format(now.Add(TTL)), CreatedAt: ids.Format(now)}
	_, err := s.db.ExecContext(ctx, `INSERT INTO pairing_tokens (id, token_hash, expires_at, created_at) VALUES (?,?,?,?)`,
		t.ID, auth.Hash(raw), t.ExpiresAt, t.CreatedAt)
	return t, err
}

// Exchange consumes raw and creates a device with a fresh credential.
func (s *Store) Exchange(ctx context.Context, raw, name, platform string) (Result, error) {
	if !auth.HasPrefix(raw, auth.PrefixPairing) {
		return Result{}, ErrInvalidToken
	}
	now := ids.Format(s.now())
	res, err := s.db.ExecContext(ctx, `UPDATE pairing_tokens SET used_at = ?
		WHERE token_hash = ? AND used_at IS NULL AND revoked_at IS NULL AND expires_at > ?`,
		now, auth.Hash(raw), now)
	if err != nil {
		return Result{}, err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return Result{}, ErrInvalidToken
	}
	d, cred, err := s.devices.Create(ctx, name, platform)
	if err != nil {
		return Result{}, err
	}
	return Result{Device: d, Credential: cred}, nil
}

// Revoke invalidates a pending token.
func (s *Store) Revoke(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE pairing_tokens SET revoked_at = ? WHERE id = ? AND revoked_at IS NULL`, ids.Format(s.now()), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Pending lists tokens that can still be exchanged.
func (s *Store) Pending(ctx context.Context) ([]Token, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, expires_at, created_at FROM pairing_tokens
		WHERE used_at IS NULL AND revoked_at IS NULL AND expires_at > ? ORDER BY created_at DESC`, ids.Format(s.now()))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Token{}
	for rows.Next() {
		var t Token
		if err := rows.Scan(&t.ID, &t.ExpiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Cleanup removes tokens that expired more than a day ago.
func (s *Store) Cleanup(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pairing_tokens WHERE expires_at < ?`, ids.Format(s.now().Add(-24*time.Hour)))
	return err
}
