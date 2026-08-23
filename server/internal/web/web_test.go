package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSPAFallback(t *testing.T) {
	h := Handler()
	for _, p := range []string{"/", "/events/evt_123", "/settings"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", p, nil))
		if rec.Code != http.StatusOK {
			t.Errorf("%s: status %d", p, rec.Code)
		}
		if ct := rec.Header().Get("Content-Type"); ct == "" || ct[:9] != "text/html" {
			t.Errorf("%s: content-type %q", p, ct)
		}
	}
}
