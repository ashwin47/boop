package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chrisgregori/boop/server/internal/apns"
	"github.com/chrisgregori/boop/server/internal/config"
	"github.com/chrisgregori/boop/server/internal/database"
	"github.com/chrisgregori/boop/server/internal/delivery"
	"github.com/chrisgregori/boop/server/internal/devices"
	"github.com/chrisgregori/boop/server/internal/events"
	"github.com/chrisgregori/boop/server/internal/pairing"
	"github.com/chrisgregori/boop/server/internal/projects"
	"github.com/chrisgregori/boop/server/internal/settings"
)

// fakeSender records notifications instead of talking to APNs.
type fakeSender struct {
	sent []sentPush
	err  error
}

type sentPush struct {
	token string
	n     apns.Notification
}

func (f *fakeSender) Send(_ context.Context, token string, n apns.Notification) (string, error) {
	f.sent = append(f.sent, sentPush{token, n})
	if f.err != nil {
		return "", f.err
	}
	return "apns-id-" + fmt.Sprint(len(f.sent)), nil
}

type env struct {
	t      *testing.T
	srv    *httptest.Server
	server *Server
	sender *fakeSender
}

func newEnv(t *testing.T) *env {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "boop.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	dev := devices.New(db)
	sender := &fakeSender{}
	s := &Server{
		Config:     config.Config{DatabasePath: "test.db", RetentionDays: 30, APNS: config.APNS{Environment: "sandbox"}},
		DB:         db,
		Log:        log,
		Settings:   settings.New(db),
		Projects:   projects.New(db),
		Devices:    dev,
		Pairing:    pairing.New(db, dev),
		Events:     events.New(db),
		Dispatcher: delivery.New(db, dev, sender, log),
		StartedAt:  time.Now(),
	}
	srv := httptest.NewServer(s.Handler())
	t.Cleanup(srv.Close)
	return &env{t: t, srv: srv, server: s, sender: sender}
}

type resp struct {
	status int
	body   map[string]any
	raw    []byte
}

func (e *env) do(method, path, bearer string, body any) resp {
	e.t.Helper()
	var rdr io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			rdr = strings.NewReader(b)
		default:
			buf, _ := json.Marshal(b)
			rdr = bytes.NewReader(buf)
		}
	}
	req, _ := http.NewRequest(method, e.srv.URL+path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
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

func (e *env) createProject(name string) (id, key string) {
	e.t.Helper()
	r := e.do("POST", "/api/v1/projects", "", map[string]string{"name": name})
	if r.status != 201 {
		e.t.Fatalf("create project: %d %s", r.status, r.raw)
	}
	return r.body["id"].(string), r.body["api_key"].(string)
}

func (e *env) pairDevice(name string) (id, cred string) {
	e.t.Helper()
	r := e.do("POST", "/api/v1/pairing", "", nil)
	if r.status != 201 {
		e.t.Fatalf("create pairing: %d %s", r.status, r.raw)
	}
	tok := r.body["token"].(string)
	r = e.do("POST", "/api/v1/pairing/exchange", "", map[string]string{"token": tok, "name": name})
	if r.status != 201 {
		e.t.Fatalf("exchange: %d %s", r.status, r.raw)
	}
	return r.body["device"].(map[string]any)["id"].(string), r.body["credential"].(string)
}

func TestHealth(t *testing.T) {
	e := newEnv(t)
	r := e.do("GET", "/health", "", nil)
	if r.status != 200 || r.body["status"] != "ok" {
		t.Fatalf("health: %d %s", r.status, r.raw)
	}
}

func TestProjectLifecycle(t *testing.T) {
	e := newEnv(t)
	r := e.do("POST", "/api/v1/projects", "", map[string]string{"name": "Uini"})
	if r.status != 201 {
		t.Fatalf("create: %d %s", r.status, r.raw)
	}
	id := r.body["id"].(string)
	key := r.body["api_key"].(string)
	if !strings.HasPrefix(key, "boop_proj_") || len(key) < 30 {
		t.Errorf("api key %q", key)
	}
	if r.body["slug"] != "uini" || r.body["notify"] != true || r.body["min_level"] != "info" {
		t.Errorf("project = %v", r.body)
	}

	// Duplicate names get distinct slugs.
	r = e.do("POST", "/api/v1/projects", "", map[string]string{"name": "Uini"})
	if r.status != 201 || r.body["slug"] != "uini-2" {
		t.Errorf("second project: %d %v", r.status, r.body)
	}

	// The raw key is never returned again.
	r = e.do("GET", "/api/v1/projects/"+id, "", nil)
	if r.status != 200 {
		t.Fatalf("get: %d", r.status)
	}
	if _, ok := r.body["api_key"]; ok {
		t.Error("api_key must not be returned on GET")
	}
	if strings.Contains(string(r.raw), "hash") {
		t.Error("hash leaked in response")
	}

	r = e.do("GET", "/api/v1/projects", "", nil)
	if r.status != 200 || len(r.body["projects"].([]any)) != 2 {
		t.Errorf("list: %d %s", r.status, r.raw)
	}

	r = e.do("PATCH", "/api/v1/projects/"+id, "", map[string]any{"name": "Uini Prod", "icon": "🚀", "notify": false, "min_level": "error"})
	if r.status != 200 || r.body["name"] != "Uini Prod" || r.body["notify"] != false || r.body["min_level"] != "error" {
		t.Errorf("patch: %d %s", r.status, r.raw)
	}
	r = e.do("PATCH", "/api/v1/projects/"+id, "", map[string]any{"min_level": "fatal"})
	if r.status != 422 {
		t.Errorf("bad level should be 422, got %d", r.status)
	}
	r = e.do("POST", "/api/v1/projects", "", map[string]string{"name": "   "})
	if r.status != 422 {
		t.Errorf("empty name should be 422, got %d", r.status)
	}
	r = e.do("POST", "/api/v1/projects", "", "{not json")
	if r.status != 400 {
		t.Errorf("bad json should be 400, got %d", r.status)
	}

	// Rotate: old key stops working, new key works.
	r = e.do("POST", "/api/v1/projects/"+id+"/rotate-key", "", nil)
	if r.status != 200 {
		t.Fatalf("rotate: %d %s", r.status, r.raw)
	}
	newKey := r.body["api_key"].(string)
	if e.do("POST", "/api/v1/events", key, map[string]string{"title": "x"}).status != 401 {
		t.Error("old key should be rejected after rotation")
	}
	if e.do("POST", "/api/v1/events", newKey, map[string]string{"title": "x"}).status != 201 {
		t.Error("new key should work")
	}

	// Delete cascades to events.
	r = e.do("DELETE", "/api/v1/projects/"+id, "", nil)
	if r.status != 204 {
		t.Errorf("delete: %d", r.status)
	}
	if e.do("GET", "/api/v1/projects/"+id, "", nil).status != 404 {
		t.Error("deleted project should 404")
	}
	r = e.do("GET", "/api/v1/events", "", nil)
	if n := len(r.body["events"].([]any)); n != 0 {
		t.Errorf("events should cascade-delete, got %d", n)
	}
}

func TestEventAuth(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("Uini")
	body := map[string]string{"title": "Deploy complete"}

	if r := e.do("POST", "/api/v1/events", "", body); r.status != 401 {
		t.Errorf("no key: %d", r.status)
	}
	if r := e.do("POST", "/api/v1/events", "boop_proj_nope", body); r.status != 401 {
		t.Errorf("wrong key: %d", r.status)
	}
	if r := e.do("POST", "/api/v1/events", key, body); r.status != 201 {
		t.Errorf("good key: %d %s", r.status, r.raw)
	}
	// A device credential cannot post events.
	_, cred := e.pairDevice("phone")
	if r := e.do("POST", "/api/v1/events", cred, body); r.status != 401 {
		t.Errorf("device cred posting events: %d", r.status)
	}
	// Neither credential type may perform admin actions.
	if r := e.do("POST", "/api/v1/projects", key, map[string]string{"name": "x"}); r.status != 403 {
		t.Errorf("project key on admin endpoint: %d", r.status)
	}
	if r := e.do("GET", "/api/v1/projects", cred, nil); r.status != 403 {
		t.Errorf("device cred on admin endpoint: %d", r.status)
	}
}

func TestCreateEventValidationAndRedaction(t *testing.T) {
	e := newEnv(t)
	pid, key := e.createProject("Uini")

	// Minimum event.
	r := e.do("POST", "/api/v1/events", key, map[string]string{"title": "Deploy complete"})
	if r.status != 201 || !strings.HasPrefix(r.body["id"].(string), "evt_") || r.body["created_at"] == nil {
		t.Fatalf("minimum event: %d %s", r.status, r.raw)
	}
	id := r.body["id"].(string)
	r = e.do("GET", "/api/v1/events/"+id, "", nil)
	if r.status != 200 || r.body["level"] != "info" || r.body["project_id"] != pid || r.body["project_name"] != "Uini" {
		t.Errorf("get: %d %s", r.status, r.raw)
	}
	if r.body["data"] == nil {
		t.Errorf("data should default to {}: %s", r.raw)
	}

	// Rich event with sensitive data in nested places and unknown fields.
	rich := `{
	  "external_id": "4f9d", "source": "error_tracker", "type": "error", "level": "error",
	  "title": "KeyError", "body": "key :can_palette? not found", "fingerprint": "fp-1",
	  "occurred_at": "2026-08-28T12:51:44Z",
	  "data": {
	    "exception": {"type": "KeyError"},
	    "context": {"user_id": "123", "password": "hunter2", "session": {"access_token": "abc"}},
	    "tags": {"environment": "production"},
	    "totally_custom": {"nested": [1, 2, {"api_key": "leak"}]}
	  }
	}`
	r = e.do("POST", "/api/v1/events", key, rich)
	if r.status != 201 {
		t.Fatalf("rich: %d %s", r.status, r.raw)
	}
	r = e.do("GET", "/api/v1/events/"+r.body["id"].(string), "", nil)
	if r.body["external_id"] != "4f9d" || r.body["source"] != "error_tracker" || r.body["fingerprint"] != "fp-1" || r.body["occurred_at"] != "2026-08-28T12:51:44.000000000Z" {
		t.Errorf("rich fields: %s", r.raw)
	}
	data := r.body["data"].(map[string]any)
	ctxm := data["context"].(map[string]any)
	if ctxm["password"] != "[REDACTED]" || ctxm["session"].(map[string]any)["access_token"] != "[REDACTED]" || ctxm["user_id"] != "123" {
		t.Errorf("context redaction: %v", ctxm)
	}
	if data["totally_custom"].(map[string]any)["nested"].([]any)[2].(map[string]any)["api_key"] != "[REDACTED]" {
		t.Errorf("custom nested redaction: %v", data["totally_custom"])
	}
	if data["exception"].(map[string]any)["type"] != "KeyError" {
		t.Errorf("unknown fields must be preserved: %v", data)
	}

	// Configured extra redaction keys apply.
	if r := e.do("PATCH", "/api/v1/settings", "", map[string]any{"redact_keys": []string{"ssn"}}); r.status != 200 {
		t.Fatalf("settings: %d %s", r.status, r.raw)
	}
	r = e.do("POST", "/api/v1/events", key, `{"title":"x","data":{"ssn":"123","name":"ok"}}`)
	r = e.do("GET", "/api/v1/events/"+r.body["id"].(string), "", nil)
	if d := r.body["data"].(map[string]any); d["ssn"] != "[REDACTED]" || d["name"] != "ok" {
		t.Errorf("custom key redaction: %v", d)
	}

	// Validation failures.
	bad := []string{
		`{}`,
		`{"title":""}`,
		`{"title":"x","level":"fatal"}`,
		`{"title":"x","occurred_at":"yesterday"}`,
		`{"title":"x","data":[1,2]}`,
		`{"title":"` + strings.Repeat("a", 201) + `"}`,
	}
	for _, b := range bad {
		if r := e.do("POST", "/api/v1/events", key, b); r.status != 422 {
			t.Errorf("%s: want 422, got %d %s", b, r.status, r.raw)
		}
	}
	if r := e.do("POST", "/api/v1/events", key, `nope`); r.status != 400 {
		t.Errorf("malformed json: %d", r.status)
	}
	if r := e.do("GET", "/api/v1/events/evt_missing", "", nil); r.status != 404 {
		t.Errorf("missing event: %d", r.status)
	}
}

func TestListEventsFilteringAndCursor(t *testing.T) {
	e := newEnv(t)
	p1, k1 := e.createProject("Uini")
	_, k2 := e.createProject("Infra")
	for i := 0; i < 7; i++ {
		lvl := "info"
		if i%2 == 0 {
			lvl = "error"
		}
		if r := e.do("POST", "/api/v1/events", k1, map[string]string{"title": fmt.Sprintf("u%d", i), "level": lvl, "source": "s1"}); r.status != 201 {
			t.Fatal(r.status)
		}
	}
	for i := 0; i < 3; i++ {
		e.do("POST", "/api/v1/events", k2, map[string]string{"title": fmt.Sprintf("i%d", i), "source": "s2"})
	}

	// Default: newest first, all projects.
	r := e.do("GET", "/api/v1/events", "", nil)
	evs := r.body["events"].([]any)
	if len(evs) != 10 || evs[0].(map[string]any)["title"] != "i2" {
		t.Fatalf("default list: %d %s", len(evs), r.raw)
	}
	if r.body["next_cursor"] != nil {
		t.Errorf("no cursor expected when exhausted")
	}

	// Page through with limit=4.
	var seen []string
	cursor := ""
	for pages := 0; pages < 10; pages++ {
		url := "/api/v1/events?limit=4"
		if cursor != "" {
			url += "&before=" + cursor
		}
		r = e.do("GET", url, "", nil)
		for _, ev := range r.body["events"].([]any) {
			seen = append(seen, ev.(map[string]any)["title"].(string))
		}
		c, _ := r.body["next_cursor"].(string)
		if c == "" {
			break
		}
		cursor = c
	}
	if len(seen) != 10 || strings.Join(seen, ",") != "i2,i1,i0,u6,u5,u4,u3,u2,u1,u0" {
		t.Errorf("paged order: %v", seen)
	}

	// Filters: by project id, by slug, level, source.
	r = e.do("GET", "/api/v1/events?project="+p1, "", nil)
	if len(r.body["events"].([]any)) != 7 {
		t.Errorf("project filter: %s", r.raw)
	}
	r = e.do("GET", "/api/v1/events?project=infra", "", nil)
	if len(r.body["events"].([]any)) != 3 {
		t.Errorf("slug filter: %s", r.raw)
	}
	r = e.do("GET", "/api/v1/events?level=error", "", nil)
	if len(r.body["events"].([]any)) != 4 {
		t.Errorf("level filter: %s", r.raw)
	}
	r = e.do("GET", "/api/v1/events?source=s2&level=info", "", nil)
	if len(r.body["events"].([]any)) != 3 {
		t.Errorf("source filter: %s", r.raw)
	}
	if r := e.do("GET", "/api/v1/events?level=fatal", "", nil); r.status != 422 {
		t.Errorf("bad level: %d", r.status)
	}
	if r := e.do("GET", "/api/v1/events?before=evt_nope", "", nil); r.status != 422 {
		t.Errorf("bad cursor: %d", r.status)
	}
	if r := e.do("GET", "/api/v1/events?limit=0", "", nil); r.status != 422 {
		t.Errorf("bad limit: %d", r.status)
	}

	// Device credentials can read; junk bearer cannot.
	_, cred := e.pairDevice("phone")
	if r := e.do("GET", "/api/v1/events", cred, nil); r.status != 200 {
		t.Errorf("device read: %d", r.status)
	}
	if r := e.do("GET", "/api/v1/events", "boop_dev_junk", nil); r.status != 401 {
		t.Errorf("junk bearer: %d", r.status)
	}
}

func TestPairingAndDevices(t *testing.T) {
	e := newEnv(t)
	r := e.do("POST", "/api/v1/pairing", "", nil)
	if r.status != 201 {
		t.Fatalf("pairing: %d %s", r.status, r.raw)
	}
	tok := r.body["token"].(string)
	qr := r.body["qr"].(map[string]any)
	if !strings.HasPrefix(tok, "pair_") || qr["version"] != float64(1) || qr["token"] != tok || !strings.HasPrefix(qr["server"].(string), "http://") {
		t.Errorf("pairing response: %s", r.raw)
	}
	pairID := r.body["id"].(string)
	if r := e.do("GET", "/api/v1/pairing", "", nil); len(r.body["pairing_tokens"].([]any)) != 1 {
		t.Errorf("pending list: %s", r.raw)
	}

	// Exchange: works once.
	r = e.do("POST", "/api/v1/pairing/exchange", "", map[string]string{"token": tok, "name": "Chris's iPhone", "platform": "ios"})
	if r.status != 201 {
		t.Fatalf("exchange: %d %s", r.status, r.raw)
	}
	cred := r.body["credential"].(string)
	dev := r.body["device"].(map[string]any)
	devID := dev["id"].(string)
	if !strings.HasPrefix(cred, "boop_dev_") || dev["name"] != "Chris's iPhone" || dev["push_registered"] != false {
		t.Errorf("exchange result: %s", r.raw)
	}
	if r := e.do("POST", "/api/v1/pairing/exchange", "", map[string]string{"token": tok}); r.status != 401 {
		t.Errorf("token reuse should fail: %d", r.status)
	}
	if r := e.do("POST", "/api/v1/pairing/exchange", "", map[string]string{"token": "pair_bogus"}); r.status != 401 {
		t.Errorf("bogus token: %d", r.status)
	}

	// Revoke a fresh token; it cannot be exchanged.
	r = e.do("POST", "/api/v1/pairing", "", nil)
	tok2, id2 := r.body["token"].(string), r.body["id"].(string)
	if r := e.do("DELETE", "/api/v1/pairing/"+id2, "", nil); r.status != 204 {
		t.Errorf("revoke: %d", r.status)
	}
	if r := e.do("POST", "/api/v1/pairing/exchange", "", map[string]string{"token": tok2}); r.status != 401 {
		t.Errorf("revoked token: %d", r.status)
	}
	if r := e.do("DELETE", "/api/v1/pairing/"+pairID, "", nil); r.status != 204 {
		t.Errorf("revoke used token should still succeed as a no-op-ish update: %d", r.status)
	}

	// Register APNs token.
	if r := e.do("POST", "/api/v1/devices", "", map[string]string{"device_token": "abc"}); r.status != 401 {
		t.Errorf("register without cred: %d", r.status)
	}
	r = e.do("POST", "/api/v1/devices", cred, map[string]string{"device_token": "tok-1", "app_bundle_id": "com.example.Boop"})
	if r.status != 200 || r.body["push_registered"] != true || r.body["app_bundle_id"] != "com.example.Boop" {
		t.Fatalf("register: %d %s", r.status, r.raw)
	}
	if strings.Contains(string(r.raw), "tok-1") {
		t.Error("device token must not be echoed back")
	}
	// Repeated registration of the same token: still one device.
	e.do("POST", "/api/v1/devices", cred, map[string]string{"device_token": "tok-1"})
	r = e.do("GET", "/api/v1/devices", "", nil)
	if n := len(r.body["devices"].([]any)); n != 1 {
		t.Errorf("devices after re-register: %d", n)
	}
	if ls := r.body["devices"].([]any)[0].(map[string]any)["last_seen_at"]; ls == nil {
		t.Error("last_seen_at should be set after authenticated call")
	}

	// A second pairing that registers the same APNs token replaces the stale device.
	_, cred2 := e.pairDevice("Same phone, re-paired")
	e.do("POST", "/api/v1/devices", cred2, map[string]string{"device_token": "tok-1"})
	r = e.do("GET", "/api/v1/devices", "", nil)
	if n := len(r.body["devices"].([]any)); n != 1 {
		t.Errorf("devices after re-pair with same token: %d %s", n, r.raw)
	}
	if e.do("GET", "/api/v1/events", cred, nil).status != 401 {
		t.Error("stale device credential should be gone")
	}

	// PATCH: a device may only edit itself; the web UI may edit any.
	newID := r.body["devices"].([]any)[0].(map[string]any)["id"].(string)
	_, cred3 := e.pairDevice("other")
	if r := e.do("PATCH", "/api/v1/devices/"+newID, cred3, map[string]string{"name": "hijack"}); r.status != 403 {
		t.Errorf("cross-device patch: %d", r.status)
	}
	if r := e.do("PATCH", "/api/v1/devices/"+newID, cred2, map[string]string{"name": "Renamed"}); r.status != 200 || r.body["name"] != "Renamed" {
		t.Errorf("self patch: %d %s", r.status, r.raw)
	}
	if r := e.do("PATCH", "/api/v1/devices/"+newID, "", map[string]string{"name": "Admin renamed"}); r.status != 200 {
		t.Errorf("admin patch: %d", r.status)
	}
	if r := e.do("DELETE", "/api/v1/devices/"+newID, "", nil); r.status != 204 {
		t.Errorf("delete: %d", r.status)
	}
	if r := e.do("DELETE", "/api/v1/devices/"+devID, "", nil); r.status != 404 {
		t.Errorf("delete already-replaced device: %d", r.status)
	}
}

func TestPairingTokenExpires(t *testing.T) {
	e := newEnv(t)
	now := time.Now()
	e.server.Pairing.SetClock(func() time.Time { return now })
	r := e.do("POST", "/api/v1/pairing", "", nil)
	tok := r.body["token"].(string)
	e.server.Pairing.SetClock(func() time.Time { return now.Add(pairing.TTL + time.Second) })
	if r := e.do("POST", "/api/v1/pairing/exchange", "", map[string]string{"token": tok}); r.status != 401 {
		t.Errorf("expired token should be rejected: %d %s", r.status, r.raw)
	}
	if r := e.do("GET", "/api/v1/pairing", "", nil); len(r.body["pairing_tokens"].([]any)) != 0 {
		t.Errorf("expired token should not be pending")
	}
}

func TestDeliveryFanOut(t *testing.T) {
	e := newEnv(t)
	pid, key := e.createProject("Uini")
	_, c1 := e.pairDevice("phone 1")
	_, c2 := e.pairDevice("phone 2")
	e.pairDevice("unregistered phone")
	e.do("POST", "/api/v1/devices", c1, map[string]string{"device_token": "t1"})
	e.do("POST", "/api/v1/devices", c2, map[string]string{"device_token": "t2"})

	// Test endpoint delivers synchronously.
	r := e.do("POST", "/api/v1/test", "", nil)
	if r.status != 201 {
		t.Fatalf("test: %d %s", r.status, r.raw)
	}
	dl := r.body["deliveries"].([]any)
	if len(dl) != 2 {
		t.Fatalf("deliveries = %d, want 2 (only devices with tokens)", len(dl))
	}
	if dl[0].(map[string]any)["status"] != "sent" || len(e.sender.sent) != 2 {
		t.Errorf("deliveries: %s", r.raw)
	}
	if e.sender.sent[0].n.Title != "Uini · Test boop" || e.sender.sent[0].n.EventID == "" || e.sender.sent[0].n.ProjectID != pid {
		t.Errorf("notification = %+v", e.sender.sent[0].n)
	}
	evID := r.body["event"].(map[string]any)["id"].(string)
	r = e.do("GET", "/api/v1/events/"+evID+"/deliveries", "", nil)
	if len(r.body["deliveries"].([]any)) != 2 {
		t.Errorf("recorded deliveries: %s", r.raw)
	}

	// Ingest path is async; wait for the worker.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.server.Dispatcher.Start(ctx)
	before := len(e.sender.sent)
	e.do("POST", "/api/v1/events", key, map[string]string{"title": "Critical thing", "level": "critical"})
	deadline := time.Now().Add(3 * time.Second)
	for len(e.sender.sent) < before+2 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if len(e.sender.sent) != before+2 {
		t.Fatalf("async delivery: sent %d", len(e.sender.sent)-before)
	}
	if !e.sender.sent[len(e.sender.sent)-1].n.Prominent {
		t.Error("critical should be prominent")
	}

	// Notifications respect project preferences.
	e.do("PATCH", "/api/v1/projects/"+pid, "", map[string]any{"min_level": "error"})
	before = len(e.sender.sent)
	e.do("POST", "/api/v1/events", key, map[string]string{"title": "just info"})
	e.do("POST", "/api/v1/test", "", nil) // success < error
	time.Sleep(100 * time.Millisecond)
	if len(e.sender.sent) != before {
		t.Errorf("min_level ignored: %d new sends", len(e.sender.sent)-before)
	}

	// Status page reflects last push.
	r = e.do("GET", "/api/v1/status", "", nil)
	if r.status != 200 || r.body["devices"] != float64(3) || r.body["pushable_devices"] != float64(2) || r.body["last_push"] == nil {
		t.Errorf("status: %d %s", r.status, r.raw)
	}
	if r.body["apns"].(map[string]any)["configured"] != false {
		t.Errorf("apns should report unconfigured in tests: %s", r.raw)
	}
}

func TestUnregisteredTokenIsCleared(t *testing.T) {
	e := newEnv(t)
	e.createProject("Uini")
	_, c1 := e.pairDevice("phone")
	e.do("POST", "/api/v1/devices", c1, map[string]string{"device_token": "dead"})
	e.sender.err = &apns.Error{Status: 410, Reason: "Unregistered"}
	r := e.do("POST", "/api/v1/test", "", nil)
	dl := r.body["deliveries"].([]any)
	if len(dl) != 1 || dl[0].(map[string]any)["status"] != "failed" {
		t.Fatalf("deliveries: %s", r.raw)
	}
	r = e.do("GET", "/api/v1/devices", "", nil)
	if r.body["devices"].([]any)[0].(map[string]any)["push_registered"] != false {
		t.Errorf("token should be cleared: %s", r.raw)
	}
}

func TestTestNotificationWithoutAPNsOrProject(t *testing.T) {
	e := newEnv(t)
	if r := e.do("POST", "/api/v1/test", "", nil); r.status != 422 {
		t.Errorf("no project: %d %s", r.status, r.raw)
	}
	e.server.Dispatcher = delivery.New(e.server.DB, e.server.Devices, nil, e.server.Log)
	e.createProject("Uini")
	_, c := e.pairDevice("phone")
	e.do("POST", "/api/v1/devices", c, map[string]string{"device_token": "t"})
	r := e.do("POST", "/api/v1/test", "", nil)
	if r.status != 201 || r.body["apns_configured"] != false {
		t.Fatalf("test without apns: %d %s", r.status, r.raw)
	}
	if d := r.body["deliveries"].([]any)[0].(map[string]any); d["status"] != "skipped" {
		t.Errorf("delivery should be skipped: %v", d)
	}
}

func TestSettings(t *testing.T) {
	e := newEnv(t)
	r := e.do("GET", "/api/v1/settings", "", nil)
	if r.status != 200 || r.body["retention_days"] != float64(30) || r.body["setup_completed"] != false {
		t.Fatalf("defaults: %d %s", r.status, r.raw)
	}
	r = e.do("PATCH", "/api/v1/settings", "", map[string]any{"retention_days": 7, "setup_completed": true, "redact_keys": []string{" ssn ", ""}})
	if r.status != 200 || r.body["retention_days"] != float64(7) || r.body["setup_completed"] != true {
		t.Fatalf("patch: %d %s", r.status, r.raw)
	}
	if ks := r.body["redact_keys"].([]any); len(ks) != 1 || ks[0] != "ssn" {
		t.Errorf("redact keys: %v", ks)
	}
	if r := e.do("PATCH", "/api/v1/settings", "", map[string]any{"retention_days": -1}); r.status != 422 {
		t.Errorf("negative retention: %d", r.status)
	}
	r = e.do("GET", "/api/v1/status", "", nil)
	if r.body["retention_days"] != float64(7) || r.body["setup_completed"] != true {
		t.Errorf("status should reflect settings: %s", r.raw)
	}
}

func TestRetentionPrune(t *testing.T) {
	e := newEnv(t)
	_, key := e.createProject("Uini")
	e.do("POST", "/api/v1/events", key, map[string]string{"title": "old"})
	e.do("POST", "/api/v1/events", key, map[string]string{"title": "new"})
	// Backdate the first event's created_at.
	if _, err := e.server.DB.Exec(`UPDATE events SET created_at = '2020-01-01T00:00:00.000000000Z' WHERE title = 'old'`); err != nil {
		t.Fatal(err)
	}
	n, err := e.server.Events.Prune(context.Background(), 0, time.Now())
	if err != nil || n != 0 {
		t.Errorf("retention 0 must not prune: %d %v", n, err)
	}
	n, err = e.server.Events.Prune(context.Background(), 30, time.Now())
	if err != nil || n != 1 {
		t.Errorf("prune: %d %v", n, err)
	}
	r := e.do("GET", "/api/v1/events", "", nil)
	evs := r.body["events"].([]any)
	if len(evs) != 1 || evs[0].(map[string]any)["title"] != "new" {
		t.Errorf("after prune: %s", r.raw)
	}
}

func TestUnknownAPIRouteIsJSON404(t *testing.T) {
	e := newEnv(t)
	r := e.do("GET", "/api/v1/nope", "", nil)
	if r.status != 404 || r.body["error"] != "not_found" {
		t.Errorf("unknown route: %d %s", r.status, r.raw)
	}
}
