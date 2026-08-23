package redact

import (
	"encoding/json"
	"testing"
)

func TestApplyRedactsNestedAndCaseInsensitive(t *testing.T) {
	in := map[string]any{}
	if err := json.Unmarshal([]byte(`{
		"api_key": "abc",
		"Password": "hunter2",
		"headers": {"Authorization": "Bearer x", "Set-Cookie": "a=b", "accept": "json"},
		"list": [{"token": "t"}, "plain", {"nested": {"private_key": "k"}}],
		"custom_field": "keep",
		"safe": 1
	}`), &in); err != nil {
		t.Fatal(err)
	}
	out := New("custom_field").Apply(in).(map[string]any)

	want := map[string]string{"api_key": Placeholder, "Password": Placeholder, "custom_field": Placeholder}
	for k, v := range want {
		if out[k] != v {
			t.Errorf("%s = %v, want %s", k, out[k], v)
		}
	}
	headers := out["headers"].(map[string]any)
	if headers["Authorization"] != Placeholder || headers["Set-Cookie"] != Placeholder {
		t.Errorf("headers not redacted: %v", headers)
	}
	if headers["accept"] != "json" {
		t.Errorf("accept should survive: %v", headers["accept"])
	}
	list := out["list"].([]any)
	if list[0].(map[string]any)["token"] != Placeholder {
		t.Errorf("token in list not redacted")
	}
	if list[1] != "plain" {
		t.Errorf("plain list value changed")
	}
	if list[2].(map[string]any)["nested"].(map[string]any)["private_key"] != Placeholder {
		t.Errorf("deeply nested private_key not redacted")
	}
	if out["safe"] != float64(1) {
		t.Errorf("safe value changed: %v", out["safe"])
	}
	// Input must not be mutated.
	if in["api_key"] != "abc" {
		t.Errorf("input was mutated")
	}
}

func TestSensitiveNormalisesSeparators(t *testing.T) {
	r := New("X-Custom-Secret")
	for _, k := range []string{"set_cookie", "SET-COOKIE", "x_custom_secret", " api_key "} {
		if !r.Sensitive(k) {
			t.Errorf("%q should be sensitive", k)
		}
	}
	if r.Sensitive("username") {
		t.Errorf("username should not be sensitive")
	}
}
