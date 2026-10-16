package main

import "testing"

func TestValidationRejectsMalformedFields(t *testing.T) {
	cases := []map[string]interface{}{
		{"title": nil}, {"title": " "}, {"title": 42},
		{"status": []string{"todo"}}, {"status": "invalid"},
		{"priority": nil}, {"priority": "urgent"}, {"description": map[string]string{}},
	}
	for _, fields := range cases {
		if err := validateTaskFields(fields); err == nil {
			t.Errorf("accepted invalid fields %#v", fields)
		}
	}
}

func TestValidationAcceptsSupportedFields(t *testing.T) {
	for _, status := range []string{"todo", "in_progress", "blocked", "done"} {
		for _, priority := range []string{"low", "medium", "high"} {
			if err := validateTaskFields(map[string]interface{}{"title": "Fix bug", "status": status, "priority": priority, "description": nil}); err != nil {
				t.Fatal(err)
			}
		}
	}
}
