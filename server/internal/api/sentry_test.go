package api

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

// postSentry sends a raw body to a Sentry ingest path with the DSN key in the
// X-Sentry-Auth header, mirroring what a real SDK transport does.
func (e *env) postSentry(path, key string, headers map[string]string, body []byte) resp {
	e.t.Helper()
	req, _ := http.NewRequest("POST", e.srv.URL+path, bytes.NewReader(body))
	req.Header.Set("X-Sentry-Auth", "Sentry sentry_version=7,sentry_client=test/1.0,sentry_key="+key)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatal(err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	out := resp{status: res.StatusCode, raw: raw}
	if len(raw) > 0 {
		_ = json.Unmarshal(raw, &out.body)
	}
	return out
}

// envelope wraps one event payload in the newline-delimited envelope format.
func envelope(eventID string, event any) []byte {
	payload, _ := json.Marshal(event)
	header, _ := json.Marshal(map[string]any{"event_id": eventID})
	item, _ := json.Marshal(map[string]any{"type": "event", "length": len(payload)})
	var b bytes.Buffer
	b.Write(header)
	b.WriteByte('\n')
	b.Write(item)
	b.WriteByte('\n')
	b.Write(payload)
	b.WriteByte('\n')
	return b.Bytes()
}

func (e *env) latestEvent(key string) events0 {
	e.t.Helper()
	r := e.do("GET", "/api/v1/events?limit=1", "", nil)
	if r.status != 200 {
		e.t.Fatalf("list events: %d %s", r.status, r.raw)
	}
	var page struct {
		Events []events0 `json:"events"`
	}
	if err := json.Unmarshal(r.raw, &page); err != nil {
		e.t.Fatal(err)
	}
	if len(page.Events) == 0 {
		e.t.Fatal("no events stored")
	}
	return page.Events[0]
}

// events0 is a minimal view of a stored event for assertions.
type events0 struct {
	Level       string          `json:"level"`
	Title       string          `json:"title"`
	Body        string          `json:"body"`
	Source      string          `json:"source"`
	Type        string          `json:"type"`
	Fingerprint string          `json:"fingerprint"`
	ExternalID  string          `json:"external_id"`
	Data        json.RawMessage `json:"data"`
}

func TestSentryExceptionEnvelope(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("app")

	event := map[string]any{
		"event_id":    "abc123def4567890abcdef1234567890",
		"level":       "error",
		"platform":    "python",
		"environment": "production",
		"release":     "1.4.2",
		"culprit":     "views.checkout in charge",
		"exception": map[string]any{
			"values": []map[string]any{{
				"type":  "ValueError",
				"value": "invalid card",
				"stacktrace": map[string]any{"frames": []map[string]any{
					{"filename": "app/views.py", "function": "charge", "lineno": 42, "in_app": true},
				}},
			}},
		},
	}
	r := e.postSentry("/api/abc/envelope/", key, nil, envelope("abc123def4567890abcdef1234567890", event))
	if r.status != http.StatusOK {
		t.Fatalf("status = %d, want 200 (%s)", r.status, r.raw)
	}
	if id, _ := r.body["id"].(string); id != "abc123def4567890abcdef1234567890" {
		t.Fatalf("echoed id = %q", id)
	}

	got := e.latestEvent(key)
	if got.Level != "error" {
		t.Errorf("level = %q, want error", got.Level)
	}
	if got.Title != "ValueError: invalid card" {
		t.Errorf("title = %q", got.Title)
	}
	if got.Source != "sentry" || got.Type != "exception" {
		t.Errorf("source/type = %q/%q", got.Source, got.Type)
	}
	if got.Fingerprint != "sentry:ValueError" {
		t.Errorf("fingerprint = %q", got.Fingerprint)
	}
	if !strings.Contains(got.Body, "app/views.py:42 in charge") {
		t.Errorf("body missing frame: %q", got.Body)
	}
	if !strings.Contains(got.Body, "env=production") {
		t.Errorf("body missing context: %q", got.Body)
	}
}

func TestSentryMultilineExceptionTitle(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("app")
	// Ruby 4.x sends a multi-line value with a redundant "(ClassName)" suffix.
	event := map[string]any{
		"event_id": "0",
		"exception": map[string]any{"values": []map[string]any{{
			"type":  "ArgumentError",
			"value": "bad billing period (ArgumentError)\n\n  raise ArgumentError\n  ^^^^^",
		}}},
	}
	if r := e.postSentry("/api/1/envelope/", key, nil, envelope("0", event)); r.status != 200 {
		t.Fatalf("status = %d", r.status)
	}
	if got := e.latestEvent(key).Title; got != "ArgumentError: bad billing period" {
		t.Errorf("title = %q, want clean single line", got)
	}
}

func TestSentryLevelMapping(t *testing.T) {
	cases := map[string]string{"fatal": "critical", "warning": "warning", "info": "info", "debug": "info"}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			e := newEnv(t)
			_, key := e.createProject("app")
			event := map[string]any{"event_id": "0", "level": in, "message": "hello"}
			r := e.postSentry("/api/1/envelope/", key, nil, envelope("0", event))
			if r.status != http.StatusOK {
				t.Fatalf("status = %d", r.status)
			}
			got := e.latestEvent(key)
			if got.Level != want {
				t.Errorf("level %q mapped to %q, want %q", in, got.Level, want)
			}
			if got.Title != "hello" || got.Type != "message" {
				t.Errorf("message event: title=%q type=%q", got.Title, got.Type)
			}
		})
	}
}

func TestSentryExplicitFingerprint(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("app")
	event := map[string]any{
		"event_id":    "0",
		"message":     "boom",
		"fingerprint": []string{"{{ default }}", "my-group", "v2"},
	}
	if r := e.postSentry("/api/1/envelope/", key, nil, envelope("0", event)); r.status != 200 {
		t.Fatalf("status = %d", r.status)
	}
	if got := e.latestEvent(key).Fingerprint; got != "my-group:v2" {
		t.Errorf("fingerprint = %q, want my-group:v2", got)
	}
}

func TestSentryGzipEnvelope(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("app")
	// Modern SDKs commonly gzip the envelope body.
	env := envelope("0", map[string]any{
		"event_id": "0", "level": "fatal",
		"exception": map[string]any{"values": []map[string]any{{"type": "PanicError", "value": "kaboom"}}},
	})
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	gz.Write(env)
	gz.Close()

	r := e.postSentry("/api/1/envelope/", key, map[string]string{"Content-Encoding": "gzip"}, buf.Bytes())
	if r.status != http.StatusOK {
		t.Fatalf("status = %d (%s)", r.status, r.raw)
	}
	got := e.latestEvent(key)
	if got.Level != "critical" || got.Title != "PanicError: kaboom" {
		t.Errorf("got level=%q title=%q", got.Level, got.Title)
	}
}

func TestSentryBadKeyRejected(t *testing.T) {
	e := newEnv(t)
	e.createProject("app")
	r := e.postSentry("/api/1/envelope/", "boop_proj_notreal", nil, envelope("0", map[string]any{"message": "x"}))
	if r.status != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", r.status)
	}
}

func TestSentryMissingKeyRejected(t *testing.T) {
	e := newEnv(t)
	// No X-Sentry-Auth header at all.
	req, _ := http.NewRequest("POST", e.srv.URL+"/api/1/envelope/", strings.NewReader("{}"))
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", res.StatusCode)
	}
}

func TestSentryTransactionItemsIgnored(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("app")
	// An envelope carrying only a transaction item stores nothing but still 200s.
	txn, _ := json.Marshal(map[string]any{"type": "transaction", "transaction": "GET /"})
	header, _ := json.Marshal(map[string]any{"event_id": "0"})
	item, _ := json.Marshal(map[string]any{"type": "transaction", "length": len(txn)})
	body := fmt.Appendf(nil, "%s\n%s\n%s\n", header, item, txn)

	if r := e.postSentry("/api/1/envelope/", key, nil, body); r.status != http.StatusOK {
		t.Fatalf("status = %d", r.status)
	}
	if r := e.do("GET", "/api/v1/events?limit=1", "", nil); r.status == 200 {
		var page struct {
			Events []events0 `json:"events"`
		}
		json.Unmarshal(r.raw, &page)
		if len(page.Events) != 0 {
			t.Fatalf("expected no stored events, got %d", len(page.Events))
		}
	}
}
