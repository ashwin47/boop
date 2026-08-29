// Sentry SDK compatibility. Point a Sentry DSN at Boop and any existing Sentry
// client reports errors straight into a Boop project. The DSN "public key" is a
// Boop project API key (boop_proj_...) and the numeric project id in the path is
// ignored, so one DSN maps to exactly one project:
//
//	SENTRY_DSN=http://boop_proj_xxxx@your-boop-host/1
//
// Events flow through the same create → redact → silence → push pipeline as the
// native POST /api/v1/events endpoint, so redaction, silences and retention all
// apply unchanged.
package api

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/chrisgreg/boop/server/internal/events"
	"github.com/chrisgreg/boop/server/internal/events/levels"
	"github.com/chrisgreg/boop/server/internal/ids"
	"github.com/chrisgreg/boop/server/internal/projects"
)

// Caps for the untrusted ingest body: the compressed request and the size we are
// willing to hold after decompression. Both are generous for real error events.
const (
	sentryMaxCompressed   = 2 << 20  // 2 MiB on the wire
	sentryMaxDecompressed = 16 << 20 // 16 MiB expanded
)

// sentryEnvelope accepts a Sentry SDK envelope: POST /api/{project_id}/envelope/.
func (s *Server) sentryEnvelope(w http.ResponseWriter, r *http.Request) {
	p, ok := s.sentryAuth(w, r)
	if !ok {
		return
	}
	body, ok := s.sentryBody(w, r)
	if !ok {
		return
	}
	id, n := s.ingestEnvelope(r.Context(), p, body)
	s.Log.Info("sentry.envelope", "project_id", p.ID, "events", n)
	writeJSON(w, http.StatusOK, map[string]string{"id": id})
}

// sentryAuth resolves the DSN key (a Boop project API key) to its project.
func (s *Server) sentryAuth(w http.ResponseWriter, r *http.Request) (projects.Project, bool) {
	key := sentryKey(r)
	if key == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "missing Sentry DSN key")
		return projects.Project{}, false
	}
	p, err := s.Projects.Authenticate(r.Context(), key)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "invalid Sentry DSN key")
		return projects.Project{}, false
	}
	return p, true
}

// sentryKey extracts the DSN public key from the X-Sentry-Auth header (the
// sentry_key=... field) or, failing that, the sentry_key query parameter. The
// header uses commas and spaces between fields, so we split on both.
func sentryKey(r *http.Request) string {
	if h := r.Header.Get("X-Sentry-Auth"); h != "" {
		for _, tok := range strings.FieldsFunc(h, func(c rune) bool { return c == ',' || c == ' ' }) {
			if v, ok := strings.CutPrefix(tok, "sentry_key="); ok {
				return v
			}
		}
	}
	return r.URL.Query().Get("sentry_key")
}

// sentryBody reads and, if needed, decompresses the request body under fixed
// size caps. Sentry SDKs commonly gzip envelopes.
func (s *Server) sentryBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	var reader io.Reader = http.MaxBytesReader(w, r.Body, sentryMaxCompressed)
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("Content-Encoding"))) {
	case "gzip":
		gz, err := gzip.NewReader(reader)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid", "malformed gzip body")
			return nil, false
		}
		defer gz.Close()
		reader = gz
	case "deflate", "zlib":
		zr, err := zlib.NewReader(reader)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid", "malformed deflate body")
			return nil, false
		}
		defer zr.Close()
		reader = zr
	}
	body, err := io.ReadAll(io.LimitReader(reader, sentryMaxDecompressed))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid", "could not read request body")
		return nil, false
	}
	return body, true
}

// ingestEnvelope turns every "event" item in a Sentry envelope into a Boop
// event, returning the first event id seen and how many were stored.
func (s *Server) ingestEnvelope(ctx context.Context, p projects.Project, body []byte) (string, int) {
	firstID, count := "", 0
	for _, it := range splitEnvelope(body) {
		if it.typ != "event" {
			continue // transactions, sessions, attachments, etc. are ignored
		}
		in, evID, ok := s.mapSentryEvent(it.payload)
		if !ok {
			continue
		}
		s.ingestOne(ctx, p, in)
		if firstID == "" {
			firstID = evID
		}
		count++
	}
	return firstID, count
}

// ingestOne stores one mapped event and pushes it unless a silence rule matches.
func (s *Server) ingestOne(ctx context.Context, p projects.Project, in events.Input) {
	e, err := s.Events.Create(ctx, p.ID, in, s.redactor(ctx))
	if err != nil {
		s.Log.Warn("sentry.event_rejected", "project_id", p.ID, "error", err.Error())
		return
	}
	s.Log.Info("event.created", "event_id", e.ID, "project_id", p.ID, "event_level", e.Level, "via", "sentry")
	if silenced := s.applySilence(ctx, &e); !silenced {
		s.Dispatcher.Enqueue(e, p)
	}
}

// ---- envelope parsing ----

type envItem struct {
	typ     string
	payload []byte
}

// splitEnvelope parses the newline-delimited Sentry envelope format: a header
// line, then repeating (item-header, payload) pairs. An item header may carry an
// explicit byte length; otherwise the payload runs to the next newline.
func splitEnvelope(b []byte) []envItem {
	var items []envItem
	// Drop the envelope header line.
	nl := bytes.IndexByte(b, '\n')
	if nl < 0 {
		return items
	}
	b = b[nl+1:]
	for len(b) > 0 {
		var headerLine []byte
		if nl = bytes.IndexByte(b, '\n'); nl < 0 {
			headerLine, b = b, nil
		} else {
			headerLine, b = b[:nl], b[nl+1:]
		}
		if headerLine = bytes.TrimSpace(headerLine); len(headerLine) == 0 {
			continue
		}
		var h struct {
			Type   string `json:"type"`
			Length *int   `json:"length"`
		}
		if err := json.Unmarshal(headerLine, &h); err != nil {
			break // an unparseable item header means we have lost framing
		}
		var payload []byte
		if h.Length != nil && *h.Length >= 0 && *h.Length <= len(b) {
			payload, b = b[:*h.Length], b[*h.Length:]
			if len(b) > 0 && b[0] == '\n' {
				b = b[1:]
			}
		} else if nl = bytes.IndexByte(b, '\n'); nl < 0 {
			payload, b = b, nil
		} else {
			payload, b = b[:nl], b[nl+1:]
		}
		items = append(items, envItem{typ: h.Type, payload: payload})
	}
	return items
}

// ---- event mapping ----

type sentryFrame struct {
	Filename string `json:"filename"`
	Function string `json:"function"`
	Module   string `json:"module"`
	Lineno   int    `json:"lineno"`
	InApp    bool   `json:"in_app"`
}

type sentryException struct {
	Type       string `json:"type"`
	Value      string `json:"value"`
	Module     string `json:"module"`
	Stacktrace struct {
		Frames []sentryFrame `json:"frames"`
	} `json:"stacktrace"`
}

// sentryEvent is the subset of the Sentry event schema Boop reads.
type sentryEvent struct {
	EventID     string          `json:"event_id"`
	Level       string          `json:"level"`
	Platform    string          `json:"platform"`
	Environment string          `json:"environment"`
	Release     string          `json:"release"`
	ServerName  string          `json:"server_name"`
	Transaction string          `json:"transaction"`
	Culprit     string          `json:"culprit"`
	Message     json.RawMessage `json:"message"`   // string or {message, formatted}
	Timestamp   json.RawMessage `json:"timestamp"` // float seconds or RFC3339
	Fingerprint []string        `json:"fingerprint"`
	Tags        json.RawMessage `json:"tags"`
	Logentry    struct {
		Message   string `json:"message"`
		Formatted string `json:"formatted"`
	} `json:"logentry"`
	Exception struct {
		Values []sentryException `json:"values"`
	} `json:"exception"`
	Sdk struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"sdk"`
}

// mapSentryEvent converts a raw Sentry event payload into a Boop event input. It
// reports false when the payload is not a usable event. The returned id is the
// Sentry event id (dashes stripped) to echo back to the SDK.
func (s *Server) mapSentryEvent(raw []byte) (events.Input, string, bool) {
	var ev sentryEvent
	if err := json.Unmarshal(raw, &ev); err != nil {
		return events.Input{}, "", false
	}
	in := events.Input{
		Source:     "sentry",
		ExternalID: clip(ev.EventID, 200),
		Level:      sentryLevel(ev.Level),
	}
	if ts := sentryTime(ev.Timestamp); ts != "" {
		in.OccurredAt = ts
	}

	msg := sentryMessage(ev)
	switch {
	case len(ev.Exception.Values) > 0:
		ex := ev.Exception.Values[len(ev.Exception.Values)-1] // the raised exception is last
		in.Type = "exception"
		// The exception value can be multi-line (e.g. Ruby appends a highlighted
		// source snippet and a redundant "(ClassName)"); a title is one clean line.
		val := strings.TrimSuffix(firstLine(ex.Value), " ("+ex.Type+")")
		if in.Title = joinTypeValue(ex.Type, val); in.Title == "" {
			in.Title = firstLine(msg)
		}
		in.Body = sentryExceptionBody(ex, ev)
		in.Fingerprint = sentryFingerprint(ev, "sentry:"+firstNonEmpty(ex.Type, ex.Module))
		if in.Level == "" {
			in.Level = levels.Error
		}
	case msg != "":
		in.Type = "message"
		in.Title = firstLine(msg)
		in.Body = sentryContextLine(ev)
		in.Fingerprint = sentryFingerprint(ev, "sentry:msg:"+firstLine(msg))
		if in.Level == "" {
			in.Level = levels.Info
		}
	default:
		return events.Input{}, stripDashes(ev.EventID), false
	}
	if in.Title == "" {
		in.Title = firstNonEmpty(ev.Transaction, ev.Culprit, "Sentry event")
	}

	in.Title = clip(in.Title, events.MaxTitle)
	in.Body = clip(in.Body, events.MaxBody)
	in.Fingerprint = clip(in.Fingerprint, 200)
	in.Data = sentryData(ev)
	return in, stripDashes(ev.EventID), true
}

// sentryLevel maps a Sentry level onto a Boop level, or "" when absent/unknown
// so the caller can pick a per-kind default.
func sentryLevel(l string) string {
	switch strings.ToLower(strings.TrimSpace(l)) {
	case "fatal":
		return levels.Critical
	case "error":
		return levels.Error
	case "warning":
		return levels.Warning
	case "info":
		return levels.Info
	case "debug":
		return levels.Info
	default:
		return ""
	}
}

// sentryMessage returns the human message from a logentry or the message field,
// which may be a plain string or a {message, formatted} object.
func sentryMessage(ev sentryEvent) string {
	if ev.Logentry.Formatted != "" {
		return ev.Logentry.Formatted
	}
	if ev.Logentry.Message != "" {
		return ev.Logentry.Message
	}
	if len(ev.Message) == 0 {
		return ""
	}
	var str string
	if json.Unmarshal(ev.Message, &str) == nil && str != "" {
		return str
	}
	var m struct {
		Message   string `json:"message"`
		Formatted string `json:"formatted"`
	}
	if json.Unmarshal(ev.Message, &m) == nil {
		return firstNonEmpty(m.Formatted, m.Message)
	}
	return ""
}

// sentryExceptionBody builds a compact, phone-readable body: where it happened,
// the top stack frame and a little environment context.
func sentryExceptionBody(ex sentryException, ev sentryEvent) string {
	var parts []string
	if loc := firstNonEmpty(ev.Culprit, ev.Transaction); loc != "" {
		parts = append(parts, loc)
	}
	if fr, ok := topFrame(ex.Stacktrace.Frames); ok {
		line := firstNonEmpty(fr.Filename, fr.Module)
		if fr.Lineno > 0 {
			line += ":" + strconv.Itoa(fr.Lineno)
		}
		if fr.Function != "" {
			line += " in " + fr.Function
		}
		if line != "" {
			parts = append(parts, line)
		}
	}
	if ctx := sentryContextLine(ev); ctx != "" {
		parts = append(parts, ctx)
	}
	return strings.Join(parts, "\n")
}

// topFrame returns the frame most likely to be the cause: the last in-app frame,
// or the last frame overall. Sentry orders frames oldest-first.
func topFrame(frames []sentryFrame) (sentryFrame, bool) {
	for i := len(frames) - 1; i >= 0; i-- {
		if frames[i].InApp {
			return frames[i], true
		}
	}
	if len(frames) > 0 {
		return frames[len(frames)-1], true
	}
	return sentryFrame{}, false
}

// sentryContextLine summarises environment/release/server for the body.
func sentryContextLine(ev sentryEvent) string {
	var kv []string
	if ev.Environment != "" {
		kv = append(kv, "env="+ev.Environment)
	}
	if ev.Release != "" {
		kv = append(kv, "release="+ev.Release)
	}
	if ev.ServerName != "" {
		kv = append(kv, "server="+ev.ServerName)
	}
	return strings.Join(kv, " · ")
}

// sentryFingerprint honours an explicit Sentry fingerprint (dropping the
// {{ default }} placeholder) so Boop silence rules can target error groups,
// falling back to a stable derived value.
func sentryFingerprint(ev sentryEvent, fallback string) string {
	var parts []string
	for _, f := range ev.Fingerprint {
		if f = strings.TrimSpace(f); f == "" || f == "{{ default }}" || f == "{{default}}" {
			continue
		}
		parts = append(parts, f)
	}
	if len(parts) > 0 {
		return strings.Join(parts, ":")
	}
	return strings.TrimSuffix(fallback, ":")
}

// sentryTime accepts a float epoch-seconds timestamp or an RFC3339 string and
// returns a Boop-formatted timestamp, or "" when absent/unparseable.
func sentryTime(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var f float64
	if json.Unmarshal(raw, &f) == nil && f > 0 {
		sec := int64(f)
		return ids.Format(time.Unix(sec, int64((f-float64(sec))*1e9)).UTC())
	}
	var str string
	if json.Unmarshal(raw, &str) == nil && str != "" {
		if t, err := ids.Parse(str); err == nil {
			return ids.Format(t)
		}
	}
	return ""
}

// sentryData stores a curated subset of the event for the detail view. The
// redactor runs over it during Create, so secrets in tags are scrubbed.
func sentryData(ev sentryEvent) json.RawMessage {
	m := map[string]any{}
	for k, v := range map[string]string{
		"event_id":    ev.EventID,
		"platform":    ev.Platform,
		"environment": ev.Environment,
		"release":     ev.Release,
		"server_name": ev.ServerName,
		"transaction": ev.Transaction,
		"culprit":     ev.Culprit,
	} {
		if v != "" {
			m[k] = v
		}
	}
	if ev.Sdk.Name != "" {
		m["sdk"] = strings.TrimSuffix(ev.Sdk.Name+"/"+ev.Sdk.Version, "/")
	}
	if len(ev.Tags) > 0 && string(ev.Tags) != "null" {
		m["tags"] = json.RawMessage(ev.Tags)
	}
	if n := len(ev.Exception.Values); n > 0 {
		ex := ev.Exception.Values[n-1]
		e := map[string]any{}
		if ex.Type != "" {
			e["type"] = ex.Type
		}
		if ex.Value != "" {
			e["value"] = ex.Value
		}
		frames := ex.Stacktrace.Frames
		if start := len(frames) - 10; start > 0 { // keep at most the last 10 frames
			frames = frames[start:]
		}
		var out []map[string]any
		for _, fr := range frames {
			out = append(out, map[string]any{"file": fr.Filename, "func": fr.Function, "line": fr.Lineno, "in_app": fr.InApp})
		}
		if len(out) > 0 {
			e["frames"] = out
		}
		if len(e) > 0 {
			m["exception"] = e
		}
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil
	}
	return json.RawMessage(b)
}

// ---- small helpers ----

func joinTypeValue(typ, val string) string {
	typ, val = strings.TrimSpace(typ), strings.TrimSpace(val)
	switch {
	case typ != "" && val != "":
		return typ + ": " + val
	case typ != "":
		return typ
	default:
		return val
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func stripDashes(s string) string { return strings.ReplaceAll(s, "-", "") }

// firstLine returns the first non-empty line of s, trimmed — for one-line titles.
func firstLine(s string) string {
	for _, line := range strings.Split(s, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}

// clip truncates s to at most max bytes without splitting a UTF-8 rune.
func clip(s string, max int) string {
	if len(s) <= max {
		return s
	}
	for max > 0 && !utf8.RuneStart(s[max]) {
		max--
	}
	return s[:max]
}
