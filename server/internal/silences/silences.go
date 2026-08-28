// Package silences stores rules that stop matching events from triggering a push.
// Silenced events are still stored and shown; they just never reach a phone.
package silences

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/chrisgreg/boop/server/internal/ids"
)

// ErrNotFound is returned for unknown silence ids.
var ErrNotFound = errors.New("silence not found")

// ErrInvalid wraps validation failures.
var ErrInvalid = errors.New("invalid silence")

// Fields a silence can match on.
const (
	FieldFingerprint = "fingerprint"
	FieldTitle       = "title"
	FieldSource      = "source"
)

// Fields lists the supported match fields.
var Fields = []string{FieldFingerprint, FieldTitle, FieldSource}

// Silence is one rule. ProjectID empty means every project.
type Silence struct {
	ID          string `json:"id"`
	ProjectID   string `json:"project_id,omitempty"`
	ProjectName string `json:"project_name,omitempty"`
	Field       string `json:"field"`
	Value       string `json:"value"`
	Note        string `json:"note"`
	CreatedAt   string `json:"created_at"`
}

// Input creates a silence.
type Input struct {
	ProjectID string `json:"project_id"`
	Field     string `json:"field"`
	Value     string `json:"value"`
	Note      string `json:"note"`
}

// Store persists silences.
type Store struct {
	db *sql.DB
}

// New returns a Store.
func New(db *sql.DB) *Store { return &Store{db: db} }

const cols = `s.id, COALESCE(s.project_id, ''), COALESCE(p.name, ''), s.field, s.value, s.note, s.created_at
	FROM silences s LEFT JOIN projects p ON p.id = s.project_id`

func scan(row interface{ Scan(...any) error }) (Silence, error) {
	var s Silence
	err := row.Scan(&s.ID, &s.ProjectID, &s.ProjectName, &s.Field, &s.Value, &s.Note, &s.CreatedAt)
	return s, err
}

// Create validates and inserts a rule.
func (st *Store) Create(ctx context.Context, in Input) (Silence, error) {
	field := strings.ToLower(strings.TrimSpace(in.Field))
	if !contains(Fields, field) {
		return Silence{}, fmt.Errorf("%w: field must be one of %s", ErrInvalid, strings.Join(Fields, ", "))
	}
	value := strings.TrimSpace(in.Value)
	if value == "" {
		return Silence{}, fmt.Errorf("%w: value is required", ErrInvalid)
	}
	if len(value) > 200 {
		return Silence{}, fmt.Errorf("%w: value must be 200 characters or fewer", ErrInvalid)
	}
	if len(in.Note) > 500 {
		return Silence{}, fmt.Errorf("%w: note must be 500 characters or fewer", ErrInvalid)
	}
	s := Silence{ID: ids.New("sil"), ProjectID: strings.TrimSpace(in.ProjectID), Field: field, Value: value, Note: strings.TrimSpace(in.Note), CreatedAt: ids.Now()}
	var project any
	if s.ProjectID != "" {
		project = s.ProjectID
	}
	_, err := st.db.ExecContext(ctx, `INSERT INTO silences (id, project_id, field, value, note, created_at) VALUES (?,?,?,?,?,?)`,
		s.ID, project, s.Field, s.Value, s.Note, s.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			return Silence{}, fmt.Errorf("%w: unknown project", ErrInvalid)
		}
		return Silence{}, err
	}
	return st.Get(ctx, s.ID)
}

// Get returns one silence.
func (st *Store) Get(ctx context.Context, id string) (Silence, error) {
	s, err := scan(st.db.QueryRowContext(ctx, `SELECT `+cols+` WHERE s.id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Silence{}, ErrNotFound
	}
	return s, err
}

// List returns every silence, newest first.
func (st *Store) List(ctx context.Context) ([]Silence, error) {
	rows, err := st.db.QueryContext(ctx, `SELECT `+cols+` ORDER BY s.created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Silence{}
	for rows.Next() {
		s, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Delete removes a rule. Events it silenced keep their history.
func (st *Store) Delete(ctx context.Context, id string) error {
	res, err := st.db.ExecContext(ctx, `DELETE FROM silences WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Match returns the first silence that applies to an event with these values
// (fingerprint and source exact, title case-insensitive), scoped to projectID
// or global. Returns nil when nothing matches.
func (st *Store) Match(ctx context.Context, projectID, fingerprint, title, source string) (*Silence, error) {
	rows, err := st.db.QueryContext(ctx, `SELECT `+cols+` WHERE (s.project_id IS NULL OR s.project_id = ?) AND (
		(s.field = 'fingerprint' AND ? <> '' AND s.value = ?) OR
		(s.field = 'source' AND ? <> '' AND s.value = ?) OR
		(s.field = 'title' AND lower(s.value) = lower(?)))
		ORDER BY s.project_id IS NULL, s.created_at LIMIT 1`,
		projectID, fingerprint, fingerprint, source, source, title)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	s, err := scan(rows)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
