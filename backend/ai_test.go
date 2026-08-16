// AI handler tests. These stand up an httptest.Server standing in for
// api.anthropic.com and point the SDK client at it via option.WithBaseURL,
// so they run with no real API key and no network call — see
// setAIForTest in ai.go.
//
// Deliberately scoped to the DB-free /ai/suggest-priority endpoint (and
// input validation on /ai/generate-tasks, which 400s before touching AI or
// the DB) — same "no DB needed" philosophy as main_test.go. The DB-touching
// AI endpoints (per-task assistant, generate-tasks' insert, NL search) are
// exercised against the real running stack via `make smoke` / the manual
// curl checklist in the README, not this unit suite.
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/gin-gonic/gin"
)

func newTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	registerAIRoutes(r)
	return r
}

// stubAnthropicServer mimics the Messages API response envelope closely
// enough for our handlers, which only ever read resp.Content[0].
func stubAnthropicServer(t *testing.T, text string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":          "msg_test",
			"type":        "message",
			"role":        "assistant",
			"model":       "claude-opus-5",
			"stop_reason": "end_turn",
			"content": []map[string]any{
				{"type": "text", "text": text},
			},
			"usage": map[string]any{"input_tokens": 10, "output_tokens": 10},
		})
	}))
	t.Cleanup(srv.Close)
	return srv
}

func useTestAIClient(t *testing.T, baseURL string) {
	t.Helper()
	client := anthropic.NewClient(option.WithAPIKey("test-key"), option.WithBaseURL(baseURL))
	setAIForTest(client, true)
	t.Cleanup(func() { setAIForTest(anthropic.Client{}, false) })
}

func postJSON(r *gin.Engine, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestAISuggestPriority_ReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	setAIForTest(anthropic.Client{}, false)
	r := newTestRouter()

	w := postJSON(r, "/ai/suggest-priority", map[string]string{"title": "Fix bug"})

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
	var got map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["error"] != "ai_unavailable" {
		t.Errorf("error = %q, want ai_unavailable", got["error"])
	}
}

func TestAISuggestPriority_ParsesValidStructuredResponse(t *testing.T) {
	srv := stubAnthropicServer(t, `{"priority":"high","tags":["security","urgent"]}`)
	useTestAIClient(t, srv.URL)
	r := newTestRouter()

	w := postJSON(r, "/ai/suggest-priority", map[string]string{
		"title":       "Patch CVE",
		"description": "urgent security fix",
	})

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
	var got struct {
		Priority string   `json:"priority"`
		Tags     []string `json:"tags"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if got.Priority != "high" {
		t.Errorf("priority = %q, want high", got.Priority)
	}
	if len(got.Tags) != 2 {
		t.Errorf("tags = %v, want 2 tags", got.Tags)
	}
}

func TestAISuggestPriority_MalformedModelOutputReturnsBadGateway(t *testing.T) {
	srv := stubAnthropicServer(t, `not valid json`)
	useTestAIClient(t, srv.URL)
	r := newTestRouter()

	w := postJSON(r, "/ai/suggest-priority", map[string]string{"title": "Anything"})

	if w.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusBadGateway, w.Body.String())
	}
	var got map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["error"] != "ai_invalid_response" {
		t.Errorf("error = %q, want ai_invalid_response", got["error"])
	}
}

func TestAISuggestPriority_RequiresTitle(t *testing.T) {
	setAIForTest(anthropic.Client{}, true)
	r := newTestRouter()

	w := postJSON(r, "/ai/suggest-priority", map[string]string{})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestAIGenerateTasks_RequiresProjectIDAndPrompt(t *testing.T) {
	// Enabled but with bad input — must 400 before ever attempting an AI call
	// (there's no stub server here, so a call attempt would fail loudly).
	setAIForTest(anthropic.Client{}, true)
	r := newTestRouter()

	w := postJSON(r, "/ai/generate-tasks", map[string]any{})

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
}

func TestAIGenerateTasks_ReturnsServiceUnavailableWhenDisabled(t *testing.T) {
	setAIForTest(anthropic.Client{}, false)
	r := newTestRouter()

	w := postJSON(r, "/ai/generate-tasks", map[string]any{"project_id": 1, "prompt": "ship v1"})

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d; body=%s", w.Code, http.StatusServiceUnavailable, w.Body.String())
	}
}

func TestBuildNLSearchPrompt_SchemaListsAllFilterFields(t *testing.T) {
	// aiSearch's whitelist validation (dropping fields that don't match
	// validTaskStatus/validTaskPriority/a parseable date) runs against a live
	// DB query and is exercised via `make smoke` / the README's manual curl
	// checklist rather than here. This test just pins the schema's field
	// names so a rename in ai.go's filter struct doesn't silently drift from
	// what the model is asked to produce.
	_, _, schema := buildNLSearchPrompt("overdue high priority tasks", "2026-08-16")
	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("schema.properties is not a map")
	}
	for _, field := range []string{"status", "priority", "due_before", "due_after", "keyword"} {
		if _, ok := props[field]; !ok {
			t.Errorf("schema is missing expected field %q", field)
		}
	}
}
