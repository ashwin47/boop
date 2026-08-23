// Package events validates, redacts, stores and lists events.
package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/chrisgreg/boop/server/internal/events/levels"
	"github.com/chrisgreg/boop/server/internal/events/redact"
	"github.com/chrisgreg/boop/server/internal/ids"
)

// ErrNotFound is returned for unknown event ids.
var ErrNotFound = errors.New("event not found")

// ErrInvalid wraps validation failures.
var ErrInvalid = errors.New("invalid event")

// Limits.
const (
	MaxTitle    = 200
	MaxBody     = 4000
	MaxDataSize = 256 * 1024
	MaxLimit    = 200
)

// Input is the inbound event envelope.
type Input struct {
	ExternalID  string          `json:"external_id"`
	Source      string          `json:"source"`
	Type        string          `json:"type"`
	Level       string          `json:"level"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	Fingerprint string          `json:"fingerprint"`
	OccurredAt  string          `json:"occurred_at"`
	Data        json.RawMessage `json:"data"`
}

// Event is a stored event.
type Event struct {
	ID          string          `json:"id"`
	ExternalID  string          `json:"external_id,omitempty"`
	ProjectID   string          `json:"project_id"`
	ProjectName string          `json:"project_name"`
	ProjectSlug string          `json:"project_slug"`
	ProjectIcon string          `json:"project_icon"`
	Source      string          `json:"source"`
	Type        string          `json:"type"`
	Level       string          `json:"level"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	Fingerprint string          `json:"fingerprint"`
	Data        json.RawMessage `json:"data"`
	OccurredAt  string          `json:"occurred_at"`
	CreatedAt   string          `json:"created_at"`
}

// Filter narrows List.
type Filter struct {
	ProjectID string
	Level     string
	Source    string
	Before    string // cursor: id of the last event seen
	Limit     int
}

// Page is a page of events plus the cursor for the next page ("" when exhausted).
type Page struct {
	Events     []Event `json:"events"`
	NextCursor string  `json:"next_cursor,omitempty"`
}

// Store persists events.
type Store struct {
	db *sql.DB
}

// New returns a Store.
func New(db *sql.DB) *Store { return &Store{db: db} }

// Validate normalises in, returning the redacted data ready for storage.
func Validate(in Input, r *redact.Redactor) (Input, []byte, error) {
	in.Title = strings.TrimSpace(in.Title)
	if in.Title == "" {
		return in, nil, fmt.Errorf("%w: title is required", ErrInvalid)
	}
	if len(in.Title) > MaxTitle {
		return in, nil, fmt.Errorf("%w: title must be %d characters or fewer", ErrInvalid, MaxTitle)
	}
	if len(in.Body) > MaxBody {
		return in, nil, fmt.Errorf("%w: body must be %d characters or fewer", ErrInvalid, MaxBody)
	}
	if in.Level == "" {
		in.Level = levels.Info
	}
	in.Level = strings.ToLower(in.Level)
	if !levels.Valid(in.Level) {
		return in, nil, fmt.Errorf("%w: level must be one of %s", ErrInvalid, strings.Join(levels.All, ", "))
	}
	for _, f := range []*string{&in.ExternalID, &in.Source, &in.Type, &in.Fingerprint} {
		*f = strings.TrimSpace(*f)
		if len(*f) > 200 {
			return in, nil, fmt.Errorf("%w: external_id, source, type and fingerprint must be 200 characters or fewer", ErrInvalid)
		}
	}
	if in.OccurredAt == "" {
		in.OccurredAt = ids.Now()
	} else {
		t, err := ids.Parse(in.OccurredAt)
		if err != nil {
			return in, nil, fmt.Errorf("%w: occurred_at must be an RFC 3339 timestamp", ErrInvalid)
		}
		in.OccurredAt = ids.Format(t)
	}

	data := []byte("{}")
	if len(in.Data) > 0 && string(in.Data) != "null" {
		if len(in.Data) > MaxDataSize {
			return in, nil, fmt.Errorf("%w: data must be %d bytes or fewer", ErrInvalid, MaxDataSize)
		}
		var v map[string]any
		if err := json.Unmarshal(in.Data, &v); err != nil {
			return in, nil, fmt.Errorf("%w: data must be a JSON object", ErrInvalid)
		}
		var err error
		data, err = json.Marshal(r.Apply(v))
		if err != nil {
			return in, nil, err
		}
	}
	return in, data, nil
}

// Create validates, redacts and stores an event for projectID.
func (s *Store) Create(ctx context.Context, projectID string, in Input, r *redact.Redactor) (Event, error) {
	in, data, err := Validate(in, r)
	if err != nil {
		return Event{}, err
	}
	e := Event{
		ID:          ids.New("evt"),
		ExternalID:  in.ExternalID,
		ProjectID:   projectID,
		Source:      in.Source,
		Type:        in.Type,
		Level:       in.Level,
		Title:       in.Title,
		Body:        in.Body,
		Fingerprint: in.Fingerprint,
		Data:        data,
		OccurredAt:  in.OccurredAt,
		CreatedAt:   ids.Now(),
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO events
		(id, external_id, project_id, source, type, level, title, body, fingerprint, payload_json, occurred_at, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.ID, nullable(e.ExternalID), e.ProjectID, e.Source, e.Type, e.Level, e.Title, e.Body, e.Fingerprint, string(e.Data), e.OccurredAt, e.CreatedAt)
	if err != nil {
		return Event{}, err
	}
	return s.Get(ctx, e.ID)
}

const selectCols = `e.id, COALESCE(e.external_id, ''), e.project_id, p.name, p.slug, p.icon, e.source, e.type, e.level, e.title, e.body, e.fingerprint, e.payload_json, e.occurred_at, e.created_at
	FROM events e JOIN projects p ON p.id = e.project_id`

func scan(row interface{ Scan(...any) error }) (Event, error) {
	var e Event
	var data string
	err := row.Scan(&e.ID, &e.ExternalID, &e.ProjectID, &e.ProjectName, &e.ProjectSlug, &e.ProjectIcon, &e.Source, &e.Type, &e.Level, &e.Title, &e.Body, &e.Fingerprint, &data, &e.OccurredAt, &e.CreatedAt)
	e.Data = json.RawMessage(data)
	return e, err
}

// Get returns one event.
func (s *Store) Get(ctx context.Context, id string) (Event, error) {
	e, err := scan(s.db.QueryRowContext(ctx, `SELECT `+selectCols+` WHERE e.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Event{}, ErrNotFound
	}
	return e, err
}

// List returns events newest-first using keyset pagination on (created_at, id).
func (s *Store) List(ctx context.Context, f Filter) (Page, error) {
	if f.Limit <= 0 {
		f.Limit = 50
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	where := []string{"1=1"}
	args := []any{}
	if f.ProjectID != "" {
		where = append(where, "(e.project_id = ? OR p.slug = ?)")
		args = append(args, f.ProjectID, f.ProjectID)
	}
	if f.Level != "" {
		where = append(where, "e.level = ?")
		args = append(args, f.Level)
	}
	if f.Source != "" {
		where = append(where, "e.source = ?")
		args = append(args, f.Source)
	}
	if f.Before != "" {
		var createdAt string
		err := s.db.QueryRowContext(ctx, `SELECT created_at FROM events WHERE id = ?`, f.Before).Scan(&createdAt)
		if errors.Is(err, sql.ErrNoRows) {
			return Page{}, fmt.Errorf("%w: unknown cursor", ErrInvalid)
		}
		if err != nil {
			return Page{}, err
		}
		where = append(where, "(e.created_at < ? OR (e.created_at = ? AND e.id < ?))")
		args = append(args, createdAt, createdAt, f.Before)
	}
	args = append(args, f.Limit+1)
	rows, err := s.db.QueryContext(ctx, `SELECT `+selectCols+` WHERE `+strings.Join(where, " AND ")+` ORDER BY e.created_at DESC, e.id DESC LIMIT ?`, args...)
	if err != nil {
		return Page{}, err
	}
	defer rows.Close()
	page := Page{Events: []Event{}}
	for rows.Next() {
		e, err := scan(rows)
		if err != nil {
			return Page{}, err
		}
		page.Events = append(page.Events, e)
	}
	if err := rows.Err(); err != nil {
		return Page{}, err
	}
	if len(page.Events) > f.Limit {
		page.Events = page.Events[:f.Limit]
		page.NextCursor = page.Events[len(page.Events)-1].ID
	}
	return page, nil
}

// Count returns the total number of stored events.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM events`).Scan(&n)
	return n, err
}

// Prune deletes events created more than days ago. days <= 0 disables pruning.
func (s *Store) Prune(ctx context.Context, days int, now time.Time) (int64, error) {
	if days <= 0 {
		return 0, nil
	}
	cutoff := ids.Format(now.Add(-time.Duration(days) * 24 * time.Hour))
	res, err := s.db.ExecContext(ctx, `DELETE FROM events WHERE created_at < ?`, cutoff)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func nullable(s string) any {
	if s == "" {
		return nil
	}
	return s
}
