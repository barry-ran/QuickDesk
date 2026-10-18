package handler

import (
	"encoding/json"
	"testing"
)

// The endpoint documents "the documented no-policy defaults" and clients render
// their form straight from this response, so a device that never had a policy
// row must produce the same *wire shape* as one that has: real arrays and a real
// approval mode, not Go's zero values.
//
// This is a regression test with a specific origin: the OpenAPI conformance
// smoke failed against a freshly seeded device with
// "access-policy: allowed_windows is not an array" (the server sent `null`) and
// "PUT access-policy changed approval_mode: off != " (the server sent `""`).
// The struct-level assertions below are deliberately made on the JSON, because
// `nil` and `[]` are the same Go value to a careless test and different bytes to
// every client.
func TestPolicyViewWithoutARowReturnsTheDocumentedDefaults(t *testing.T) {
	encoded, err := json.Marshal(policyView("123456789", nil))
	if err != nil {
		t.Fatalf("marshal view: %v", err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(encoded, &raw); err != nil {
		t.Fatalf("unmarshal view: %v", err)
	}

	for _, key := range []string{"allow_ips", "allowed_windows", "exception_dates", "approval_trusted_ips"} {
		value, present := raw[key]
		if !present {
			t.Fatalf("%s is missing from the no-policy response; the schema lists it as a documented array", key)
		}
		if _, isArray := value.([]interface{}); !isArray {
			t.Fatalf("%s = %#v, want an array (JSON null is not an array)", key, value)
		}
	}
	if got := raw["approval_mode"]; got != "off" {
		t.Fatalf("approval_mode = %#v, want \"off\" - the schema enumerates the modes and \"\" is not one", got)
	}
	if _, present := raw["updated_at"]; present {
		t.Fatal("updated_at must be omitted when the owner never saved a policy")
	}
	if got := raw["device_id"]; got != "123456789" {
		t.Fatalf("device_id = %#v, want the requested device", got)
	}
}
