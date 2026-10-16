// Starter test for env() so `go test ./...` does real work in CI. No DB needed.
package main

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestEnvReturnsFallbackWhenUnset(t *testing.T) {
	os.Unsetenv("DEVBOARD_TEST_KEY")
	if got := env("DEVBOARD_TEST_KEY", "fallback"); got != "fallback" {
		t.Errorf("env() = %q, want %q", got, "fallback")
	}
}

func TestEnvReturnsValueWhenSet(t *testing.T) {
	os.Setenv("DEVBOARD_TEST_KEY", "real")
	defer os.Unsetenv("DEVBOARD_TEST_KEY")
	if got := env("DEVBOARD_TEST_KEY", "fallback"); got != "real" {
		t.Errorf("env() = %q, want %q", got, "real")
	}
}

func TestToStringSlice(t *testing.T) {
	if _, ok := toStringSlice("not an array"); ok {
		t.Error("toStringSlice(non-array) should return ok=false")
	}
	if _, ok := toStringSlice([]interface{}{"a", 1}); ok {
		t.Error("toStringSlice with a non-string element should return ok=false")
	}
	got, ok := toStringSlice([]interface{}{"frontend", "urgent"})
	if !ok {
		t.Fatal("toStringSlice(valid array) should return ok=true")
	}
	if len(got) != 2 || got[0] != "frontend" || got[1] != "urgent" {
		t.Errorf("toStringSlice = %v, want [frontend urgent]", got)
	}
}

func TestInvalidTaskWritesReturn400BeforeDatabase(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/tasks", createTask)
	router.PATCH("/tasks/:id", updateTask)
	for _, body := range []string{
		`{"title":"ok","project_id":1,"status":"invalid"}`,
		`{"title":"ok","project_id":1,"priority":"urgent"}`,
		`{"title":"   ","project_id":1}`,
		`{"title":"ok","project_id":-1}`,
	} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/tasks", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		if response.Code != 400 {
			t.Fatalf("POST %s returned %d", body, response.Code)
		}
	}
	for _, body := range []string{`{"title":null}`, `{"title":42}`, `{"status":[]}`, `{"priority":null}`, `{"description":{}}`, `{"title":" "}`} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPatch, "/tasks/1", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(response, request)
		if response.Code != 400 {
			t.Fatalf("PATCH %s returned %d", body, response.Code)
		}
	}
}

func TestValidTaskFieldValidation(t *testing.T) {
	if err := validateTaskFields(map[string]interface{}{"title": "Fix bug", "status": "done", "priority": "high", "description": nil}); err != nil {
		t.Fatal(err)
	}
}
