// Package projects manages projects and their API keys.
package projects

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/chrisgregori/boop/server/internal/auth"
	"github.com/chrisgregori/boop/server/internal/database"
	"github.com/chrisgregori/boop/server/internal/events/levels"
	"github.com/chrisgregori/boop/server/internal/ids"
)

// ErrNotFound is returned when a project does not exist.
var ErrNotFound = errors.New("project not found")

// ErrInvalid wraps validation failures.
var ErrInvalid = errors.New("invalid project")

// Project is a group of events with its own API key.
type Project struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Icon      string `json:"icon"`
	Notify    bool   `json:"notify"`
	MinLevel  string `json:"min_level"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

// Input is the writable subset of a project.
type Input struct {
	Name     *string `json:"name"`
	Icon     *string `json:"icon"`
	Notify   *bool   `json:"notify"`
	MinLevel *string `json:"min_level"`
}

// Store persists projects.
type Store struct {
	db *sql.DB
}

// New returns a Store.
func New(db *sql.DB) *Store { return &Store{db: db} }

const cols = `id, name, slug, icon, notify, min_level, created_at, updated_at`

func scan(row interface{ Scan(...any) error }) (Project, error) {
	var p Project
	var notify int
	err := row.Scan(&p.ID, &p.Name, &p.Slug, &p.Icon, &notify, &p.MinLevel, &p.CreatedAt, &p.UpdatedAt)
	p.Notify = notify == 1
	return p, err
}

// Create inserts a project and returns it along with its raw API key. The raw
// key is never stored and cannot be recovered later.
func (s *Store) Create(ctx context.Context, in Input) (Project, string, error) {
	name := ""
	if in.Name != nil {
		name = strings.TrimSpace(*in.Name)
	}
	if name == "" {
		return Project{}, "", fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if len(name) > 80 {
		return Project{}, "", fmt.Errorf("%w: name must be 80 characters or fewer", ErrInvalid)
	}
	minLevel := levels.Info
	if in.MinLevel != nil && *in.MinLevel != "" {
		if !levels.Valid(*in.MinLevel) {
			return Project{}, "", fmt.Errorf("%w: min_level must be one of %s", ErrInvalid, strings.Join(levels.All, ", "))
		}
		minLevel = *in.MinLevel
	}
	notify := true
	if in.Notify != nil {
		notify = *in.Notify
	}
	icon := ""
	if in.Icon != nil {
		icon = strings.TrimSpace(*in.Icon)
	}
	if !ValidIcon(icon) {
		return Project{}, "", fmt.Errorf("%w: icon must be <shape>:<color> (shapes: %s) or a short emoji", ErrInvalid, strings.Join(IconShapes, ", "))
	}
	if icon == "" {
		icon = DefaultIcon(Slugify(name))
	}
	key := auth.NewSecret(auth.PrefixProjectKey)
	now := ids.Now()
	p := Project{ID: ids.New("prj"), Name: name, Icon: icon, Notify: notify, MinLevel: minLevel, CreatedAt: now, UpdatedAt: now}

	base := Slugify(name)
	for i := 0; i < 20; i++ {
		p.Slug = base
		if i > 0 {
			p.Slug = fmt.Sprintf("%s-%d", base, i+1)
		}
		_, err := s.db.ExecContext(ctx, `INSERT INTO projects (`+cols+`, api_key_hash) VALUES (?,?,?,?,?,?,?,?,?)`,
			p.ID, p.Name, p.Slug, p.Icon, boolInt(p.Notify), p.MinLevel, p.CreatedAt, p.UpdatedAt, auth.Hash(key))
		if err == nil {
			return p, key, nil
		}
		if !database.IsUniqueViolation(err) || !strings.Contains(err.Error(), "slug") {
			return Project{}, "", err
		}
	}
	return Project{}, "", fmt.Errorf("%w: could not find a free slug for %q", ErrInvalid, name)
}

// List returns every project, newest first.
func (s *Store) List(ctx context.Context) ([]Project, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+cols+` FROM projects ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Project{}
	for rows.Next() {
		p, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// Get returns a project by id.
func (s *Store) Get(ctx context.Context, id string) (Project, error) {
	p, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM projects WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return p, err
}

// Count returns the number of projects.
func (s *Store) Count(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM projects`).Scan(&n)
	return n, err
}

// First returns the oldest project, or ErrNotFound when none exist.
func (s *Store) First(ctx context.Context) (Project, error) {
	p, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM projects ORDER BY created_at ASC LIMIT 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return p, err
}

// Authenticate resolves a raw project API key to its project.
func (s *Store) Authenticate(ctx context.Context, rawKey string) (Project, error) {
	if !auth.HasPrefix(rawKey, auth.PrefixProjectKey) {
		return Project{}, ErrNotFound
	}
	p, err := scan(s.db.QueryRowContext(ctx, `SELECT `+cols+` FROM projects WHERE api_key_hash = ?`, auth.Hash(rawKey)))
	if errors.Is(err, sql.ErrNoRows) {
		return Project{}, ErrNotFound
	}
	return p, err
}

// Update applies the non-nil fields of in.
func (s *Store) Update(ctx context.Context, id string, in Input) (Project, error) {
	p, err := s.Get(ctx, id)
	if err != nil {
		return Project{}, err
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" || len(n) > 80 {
			return Project{}, fmt.Errorf("%w: name must be 1-80 characters", ErrInvalid)
		}
		p.Name = n
	}
	if in.Icon != nil {
		icon := strings.TrimSpace(*in.Icon)
		if !ValidIcon(icon) {
			return Project{}, fmt.Errorf("%w: icon must be <shape>:<color> (shapes: %s) or a short emoji", ErrInvalid, strings.Join(IconShapes, ", "))
		}
		if icon == "" {
			icon = DefaultIcon(p.Slug)
		}
		p.Icon = icon
	}
	if in.Notify != nil {
		p.Notify = *in.Notify
	}
	if in.MinLevel != nil {
		if !levels.Valid(*in.MinLevel) {
			return Project{}, fmt.Errorf("%w: min_level must be one of %s", ErrInvalid, strings.Join(levels.All, ", "))
		}
		p.MinLevel = *in.MinLevel
	}
	p.UpdatedAt = ids.Now()
	_, err = s.db.ExecContext(ctx, `UPDATE projects SET name=?, icon=?, notify=?, min_level=?, updated_at=? WHERE id=?`,
		p.Name, p.Icon, boolInt(p.Notify), p.MinLevel, p.UpdatedAt, p.ID)
	return p, err
}

// RotateKey generates a new API key for the project and returns the raw key.
func (s *Store) RotateKey(ctx context.Context, id string) (string, error) {
	key := auth.NewSecret(auth.PrefixProjectKey)
	res, err := s.db.ExecContext(ctx, `UPDATE projects SET api_key_hash=?, updated_at=? WHERE id=?`, auth.Hash(key), ids.Now(), id)
	if err != nil {
		return "", err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return "", ErrNotFound
	}
	return key, nil
}

// Delete removes a project and (via cascade) its events.
func (s *Store) Delete(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

var nonSlug = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify lowercases and dashes a name: "My Project!" -> "my-project".
func Slugify(name string) string {
	s := nonSlug.ReplaceAllString(strings.ToLower(name), "-")
	s = strings.Trim(s, "-")
	if s == "" {
		s = "project"
	}
	return s
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
