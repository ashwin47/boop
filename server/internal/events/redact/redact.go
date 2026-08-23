// Package redact recursively replaces the values of sensitive keys in JSON-like data.
package redact

import "strings"

// Placeholder replaces redacted values.
const Placeholder = "[REDACTED]"

// DefaultKeys are always redacted.
var DefaultKeys = []string{
	"password",
	"password_confirmation",
	"secret",
	"token",
	"access_token",
	"refresh_token",
	"api_key",
	"authorization",
	"cookie",
	"set-cookie",
	"private_key",
}

// Redactor holds a normalised key set.
type Redactor struct {
	keys map[string]bool
}

// New returns a Redactor for DefaultKeys plus extra. Matching is
// case-insensitive and treats "-" and "_" as equivalent.
func New(extra ...string) *Redactor {
	r := &Redactor{keys: map[string]bool{}}
	for _, k := range DefaultKeys {
		r.keys[normalise(k)] = true
	}
	for _, k := range extra {
		if n := normalise(k); n != "" {
			r.keys[n] = true
		}
	}
	return r
}

// Sensitive reports whether key is on the redaction list.
func (r *Redactor) Sensitive(key string) bool {
	return r.keys[normalise(key)]
}

// Apply returns v with every value under a sensitive key replaced by
// Placeholder, descending into nested maps and slices. The input is not mutated.
func (r *Redactor) Apply(v any) any {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			if r.Sensitive(k) {
				out[k] = Placeholder
			} else {
				out[k] = r.Apply(val)
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = r.Apply(val)
		}
		return out
	default:
		return v
	}
}

func normalise(k string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(k)), "-", "_")
}
