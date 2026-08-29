// Package mcp exposes Boop's stored events to AI agents over the Model
// Context Protocol. Every tool is read-only: agents can list, search and
// inspect events and projects, never create or change anything.
package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/chrisgreg/boop/server/internal/events"
	"github.com/chrisgreg/boop/server/internal/events/levels"
	"github.com/chrisgreg/boop/server/internal/projects"
)

// Stores is what the tools read from.
type Stores struct {
	Projects *projects.Store
	Events   *events.Store
}

// DefaultLimit and MaxLimit bound list sizes so a tool result stays small
// enough to fit comfortably in an agent's context.
const (
	DefaultLimit = 25
	MaxLimit     = 100
)

// NewServer builds the MCP server with every tool registered.
func NewServer(st Stores, version string) *sdk.Server {
	s := sdk.NewServer(&sdk.Implementation{Name: "boop", Title: "Boop", Version: version}, &sdk.ServerOptions{
		Instructions: "Boop is a self-hosted notification inbox: applications post events (deploys, errors, payments, cron results) and the owner gets a push. " +
			"These tools are read-only. Events belong to a project; events sharing a fingerprint within a project are the same thing recurring. " +
			"Timestamps are RFC 3339 in UTC. Start with list_projects, then list_events or search_events, and use get_event for full detail.",
	})
	t := &tools{st: st}
	ro := &sdk.ToolAnnotations{ReadOnlyHint: true, IdempotentHint: true, OpenWorldHint: boolPtr(false)}

	sdk.AddTool(s, &sdk.Tool{Name: "list_projects", Title: "List projects", Annotations: ro,
		Description: "List every project (an app or service that sends events), with its id and slug. Use the id or slug as the project argument of the other tools."}, t.listProjects)
	sdk.AddTool(s, &sdk.Tool{Name: "list_events", Title: "List events", Annotations: ro,
		Description: "List recent events newest first, optionally filtered by project, level, source, fingerprint and a since/until time window (RFC 3339). " +
			"Set grouped=true to collapse repeated occurrences of the same fingerprint into one row with a count and first/last seen. " +
			"Results are summaries; call get_event for the full payload. Page with the returned next_cursor as the before argument."}, t.listEvents)
	sdk.AddTool(s, &sdk.Tool{Name: "get_event", Title: "Get event", Annotations: ro,
		Description: "Fetch one event by id with its complete data payload (exception, stacktrace, tags, context, breadcrumbs and anything else the sender attached) and actions."}, t.getEvent)
	sdk.AddTool(s, &sdk.Tool{Name: "search_events", Title: "Search events", Annotations: ro,
		Description: "Case-insensitive substring search across event titles, bodies, sources, fingerprints and data payloads, newest first. Combine with project, level and since/until to narrow it down."}, t.searchEvents)
	sdk.AddTool(s, &sdk.Tool{Name: "get_event_group", Title: "Get event group", Annotations: ro,
		Description: "Everything about one recurring event: the occurrences that share a fingerprint within a project, newest first, with the count and first/last seen. " +
			"Use the fingerprint from list_events (grouped) or an event."}, t.getEventGroup)
	return s
}

// Handler returns an HTTP handler speaking Streamable HTTP. Authentication
// is the caller's job (wrap it); the handler itself is stateless so any
// request can be served by any instance without session bookkeeping.
func Handler(s *sdk.Server, log *slog.Logger) http.Handler {
	return sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return s }, &sdk.StreamableHTTPOptions{
		Stateless: true, JSONResponse: true, Logger: log,
		// The SDK's DNS-rebinding guard rejects requests that arrive on loopback
		// with a non-localhost Host header, which is exactly a reverse proxy on
		// the same box (nginx/Caddy -> 127.0.0.1:8080). The endpoint is bearer
		// authenticated and the rest of the API does no origin checks either.
		DisableLocalhostProtection: true,
	})
}

type tools struct{ st Stores }

// ---- list_projects ----

type ListProjectsIn struct{}

type ProjectOut struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Slug      string `json:"slug"`
	Notify    bool   `json:"notify" jsonschema:"whether pushes are enabled for this project"`
	MinLevel  string `json:"min_level" jsonschema:"lowest level that is pushed to the phone"`
	CreatedAt string `json:"created_at"`
}

type ListProjectsOut struct {
	Projects []ProjectOut `json:"projects"`
}

func (t *tools) listProjects(ctx context.Context, _ *sdk.CallToolRequest, _ ListProjectsIn) (*sdk.CallToolResult, ListProjectsOut, error) {
	ps, err := t.st.Projects.List(ctx)
	if err != nil {
		return nil, ListProjectsOut{}, err
	}
	out := ListProjectsOut{Projects: []ProjectOut{}}
	for _, p := range ps {
		out.Projects = append(out.Projects, ProjectOut{ID: p.ID, Name: p.Name, Slug: p.Slug, Notify: p.Notify, MinLevel: p.MinLevel, CreatedAt: p.CreatedAt})
	}
	return nil, out, nil
}

// ---- list_events / search_events ----

type ListEventsIn struct {
	Project     string `json:"project,omitempty" jsonschema:"project id or slug; omit for all projects"`
	Level       string `json:"level,omitempty" jsonschema:"one of info, success, warning, error, critical"`
	Source      string `json:"source,omitempty" jsonschema:"exact source, e.g. error_tracker or github_actions"`
	Fingerprint string `json:"fingerprint,omitempty" jsonschema:"only occurrences with this exact fingerprint"`
	Since       string `json:"since,omitempty" jsonschema:"RFC 3339 timestamp; only events received at or after this time"`
	Until       string `json:"until,omitempty" jsonschema:"RFC 3339 timestamp; only events received before this time"`
	Silenced    *bool  `json:"silenced,omitempty" jsonschema:"true for only silenced events, false for only pushed ones; omit for both"`
	Grouped     bool   `json:"grouped,omitempty" jsonschema:"collapse repeats of the same fingerprint into one row with a count"`
	Before      string `json:"before,omitempty" jsonschema:"cursor from a previous call's next_cursor"`
	Limit       int    `json:"limit,omitempty" jsonschema:"1-100, default 25"`
}

type SearchEventsIn struct {
	Query   string `json:"query" jsonschema:"text to look for (case-insensitive substring)"`
	Project string `json:"project,omitempty" jsonschema:"project id or slug; omit for all projects"`
	Level   string `json:"level,omitempty" jsonschema:"one of info, success, warning, error, critical"`
	Since   string `json:"since,omitempty" jsonschema:"RFC 3339 timestamp; only events received at or after this time"`
	Until   string `json:"until,omitempty" jsonschema:"RFC 3339 timestamp; only events received before this time"`
	Before  string `json:"before,omitempty" jsonschema:"cursor from a previous call's next_cursor"`
	Limit   int    `json:"limit,omitempty" jsonschema:"1-100, default 25"`
}

// EventSummary is an event without its data payload.
type EventSummary struct {
	ID          string            `json:"id"`
	Project     string            `json:"project" jsonschema:"project name"`
	ProjectID   string            `json:"project_id"`
	Level       string            `json:"level"`
	Title       string            `json:"title"`
	Body        string            `json:"body,omitempty"`
	Source      string            `json:"source,omitempty"`
	Type        string            `json:"type,omitempty"`
	Fingerprint string            `json:"fingerprint,omitempty"`
	ExternalID  string            `json:"external_id,omitempty"`
	OccurredAt  string            `json:"occurred_at"`
	CreatedAt   string            `json:"created_at" jsonschema:"when the server received it"`
	Silenced    bool              `json:"silenced,omitempty"`
	DataKeys    []string          `json:"data_keys,omitempty" jsonschema:"top-level keys present in the data payload (fetch with get_event)"`
	Actions     []events.Action   `json:"actions,omitempty"`
	Group       *events.GroupInfo `json:"group,omitempty" jsonschema:"present in grouped listings: how many times this fingerprint occurred and when"`
}

type EventsOut struct {
	Events     []EventSummary `json:"events"`
	NextCursor string         `json:"next_cursor,omitempty" jsonschema:"pass as before to get the next page"`
}

func (t *tools) listEvents(ctx context.Context, _ *sdk.CallToolRequest, in ListEventsIn) (*sdk.CallToolResult, EventsOut, error) {
	f := events.Filter{ProjectID: in.Project, Level: in.Level, Source: in.Source, Fingerprint: in.Fingerprint, Since: in.Since, Until: in.Until,
		Silenced: in.Silenced, Grouped: in.Grouped, Before: in.Before, Limit: in.Limit}
	return t.list(ctx, f)
}

func (t *tools) searchEvents(ctx context.Context, _ *sdk.CallToolRequest, in SearchEventsIn) (*sdk.CallToolResult, EventsOut, error) {
	if strings.TrimSpace(in.Query) == "" {
		return nil, EventsOut{}, errors.New("query is required")
	}
	f := events.Filter{Query: in.Query, ProjectID: in.Project, Level: in.Level, Since: in.Since, Until: in.Until, Before: in.Before, Limit: in.Limit}
	return t.list(ctx, f)
}

func (t *tools) list(ctx context.Context, f events.Filter) (*sdk.CallToolResult, EventsOut, error) {
	if f.Level != "" && !levels.Valid(f.Level) {
		return nil, EventsOut{}, fmt.Errorf("level must be one of %s", strings.Join(levels.All, ", "))
	}
	if f.Limit <= 0 {
		f.Limit = DefaultLimit
	}
	if f.Limit > MaxLimit {
		f.Limit = MaxLimit
	}
	page, err := t.st.Events.List(ctx, f)
	if err != nil {
		return nil, EventsOut{}, err
	}
	out := EventsOut{Events: []EventSummary{}, NextCursor: page.NextCursor}
	for _, e := range page.Events {
		out.Events = append(out.Events, summarise(e))
	}
	return nil, out, nil
}

func summarise(e events.Event) EventSummary {
	s := EventSummary{ID: e.ID, Project: e.ProjectName, ProjectID: e.ProjectID, Level: e.Level, Title: e.Title, Body: e.Body, Source: e.Source, Type: e.Type,
		Fingerprint: e.Fingerprint, ExternalID: e.ExternalID, OccurredAt: e.OccurredAt, CreatedAt: e.CreatedAt, Silenced: e.Silenced, Actions: e.Actions, Group: e.Group}
	var data map[string]json.RawMessage
	if json.Unmarshal(e.Data, &data) == nil {
		for k := range data {
			s.DataKeys = append(s.DataKeys, k)
		}
	}
	return s
}

// ---- get_event ----

type GetEventIn struct {
	ID string `json:"id" jsonschema:"event id, e.g. evt_abc123"`
}

// EventOut is the full event.
type EventOut struct {
	EventSummary
	ProjectSlug string         `json:"project_slug"`
	Data        map[string]any `json:"data" jsonschema:"the sender's free-form payload; exception, stacktrace, tags, context and breadcrumbs are conventional keys"`
}

func (t *tools) getEvent(ctx context.Context, _ *sdk.CallToolRequest, in GetEventIn) (*sdk.CallToolResult, EventOut, error) {
	e, err := t.st.Events.Get(ctx, strings.TrimSpace(in.ID))
	if err != nil {
		return nil, EventOut{}, err
	}
	return nil, full(e), nil
}

func full(e events.Event) EventOut {
	out := EventOut{EventSummary: summarise(e), ProjectSlug: e.ProjectSlug, Data: map[string]any{}}
	out.DataKeys = nil
	_ = json.Unmarshal(e.Data, &out.Data)
	return out
}

// ---- get_event_group ----

type GetEventGroupIn struct {
	Project     string `json:"project" jsonschema:"project id or slug"`
	Fingerprint string `json:"fingerprint" jsonschema:"the fingerprint shared by the occurrences"`
	Since       string `json:"since,omitempty" jsonschema:"RFC 3339 timestamp; only occurrences received at or after this time"`
	Until       string `json:"until,omitempty" jsonschema:"RFC 3339 timestamp; only occurrences received before this time"`
	Before      string `json:"before,omitempty" jsonschema:"cursor from a previous call's next_cursor"`
	Limit       int    `json:"limit,omitempty" jsonschema:"occurrences to return, 1-100, default 25"`
}

type EventGroupOut struct {
	Project     string         `json:"project"`
	ProjectID   string         `json:"project_id"`
	Fingerprint string         `json:"fingerprint"`
	Count       int            `json:"count" jsonschema:"total occurrences in the window"`
	FirstSeen   string         `json:"first_seen,omitempty"`
	LastSeen    string         `json:"last_seen,omitempty"`
	Latest      *EventOut      `json:"latest,omitempty" jsonschema:"the most recent occurrence in full"`
	Occurrences []EventSummary `json:"occurrences"`
	NextCursor  string         `json:"next_cursor,omitempty"`
}

func (t *tools) getEventGroup(ctx context.Context, _ *sdk.CallToolRequest, in GetEventGroupIn) (*sdk.CallToolResult, EventGroupOut, error) {
	in.Project, in.Fingerprint = strings.TrimSpace(in.Project), strings.TrimSpace(in.Fingerprint)
	if in.Project == "" || in.Fingerprint == "" {
		return nil, EventGroupOut{}, errors.New("project and fingerprint are required")
	}
	if in.Limit <= 0 {
		in.Limit = DefaultLimit
	}
	if in.Limit > MaxLimit {
		in.Limit = MaxLimit
	}
	// One grouped row gives the count and window; then the occurrences.
	head, err := t.st.Events.List(ctx, events.Filter{ProjectID: in.Project, Fingerprint: in.Fingerprint, Since: in.Since, Until: in.Until, Grouped: true, Limit: 1})
	if err != nil {
		return nil, EventGroupOut{}, err
	}
	if len(head.Events) == 0 {
		return nil, EventGroupOut{}, fmt.Errorf("no events with fingerprint %q in project %q", in.Fingerprint, in.Project)
	}
	latest := head.Events[0]
	out := EventGroupOut{Project: latest.ProjectName, ProjectID: latest.ProjectID, Fingerprint: latest.Fingerprint, Occurrences: []EventSummary{}}
	if latest.Group != nil {
		out.Count, out.FirstSeen, out.LastSeen = latest.Group.Count, latest.Group.FirstSeen, latest.Group.LastSeen
	}
	l := full(latest)
	out.Latest = &l
	page, err := t.st.Events.List(ctx, events.Filter{ProjectID: in.Project, Fingerprint: in.Fingerprint, Since: in.Since, Until: in.Until, Before: in.Before, Limit: in.Limit})
	if err != nil {
		return nil, EventGroupOut{}, err
	}
	for _, e := range page.Events {
		out.Occurrences = append(out.Occurrences, summarise(e))
	}
	out.NextCursor = page.NextCursor
	return nil, out, nil
}

func boolPtr(b bool) *bool { return &b }
