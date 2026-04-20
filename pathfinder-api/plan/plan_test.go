package plan_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"pathfinder-api/middleware"
	"pathfinder-api/plan"
	"pathfinder-api/storage"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	storage.Init(":memory:")
	os.Exit(m.Run())
}

func newAgentRouter() *gin.Engine {
	middleware.InitServiceToken("tok", "u1")
	r := gin.New()
	r.POST("/api/agent/tasks", middleware.ServiceTokenAuth(), plan.CreateTask)
	r.PATCH("/api/agent/tasks/:id", middleware.ServiceTokenAuth(), plan.PatchTask)
	return r
}

func post(r *gin.Engine, url string, body any, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateTask_ValidRequest(t *testing.T) {
	r := newAgentRouter()
	w := post(r, "/api/agent/tasks", map[string]any{
		"title": "Write tests",
		"date":  "2026-04-20",
	}, "tok")

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var task map[string]any
	json.Unmarshal(w.Body.Bytes(), &task)
	if task["title"] != "Write tests" {
		t.Errorf("title = %v, want 'Write tests'", task["title"])
	}
	if task["status"] != "pending" {
		t.Errorf("status = %v, want 'pending'", task["status"])
	}
}

func TestCreateTask_DefaultsToToday(t *testing.T) {
	r := newAgentRouter()
	w := post(r, "/api/agent/tasks", map[string]any{"title": "No date"}, "tok")
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
}

func TestCreateTask_MissingTitle(t *testing.T) {
	r := newAgentRouter()
	w := post(r, "/api/agent/tasks", map[string]any{"description": "no title"}, "tok")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateTask_TitleTooLong(t *testing.T) {
	r := newAgentRouter()
	long := make([]byte, 201)
	for i := range long {
		long[i] = 'a'
	}
	w := post(r, "/api/agent/tasks", map[string]any{"title": string(long)}, "tok")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateTask_InvalidDate(t *testing.T) {
	r := newAgentRouter()
	w := post(r, "/api/agent/tasks", map[string]any{"title": "x", "date": "not-a-date"}, "tok")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateTask_SortOrderIncremental(t *testing.T) {
	r := newAgentRouter()
	post(r, "/api/agent/tasks", map[string]any{"title": "first", "date": "2026-05-01"}, "tok")
	w := post(r, "/api/agent/tasks", map[string]any{"title": "second", "date": "2026-05-01"}, "tok")

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var task map[string]any
	json.Unmarshal(w.Body.Bytes(), &task)
	if task["sort_order"].(float64) != 2 {
		t.Errorf("sort_order = %v, want 2", task["sort_order"])
	}
}

func TestCreateTask_NoToken(t *testing.T) {
	r := newAgentRouter()
	w := post(r, "/api/agent/tasks", map[string]any{"title": "x"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func patch(r *gin.Engine, url string, body any, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// createTestTask is a helper that POSTs a task and returns its numeric ID.
func createTestTask(t *testing.T, r *gin.Engine, title, date string) float64 {
	t.Helper()
	w := post(r, "/api/agent/tasks", map[string]any{"title": title, "date": date}, "tok")
	if w.Code != http.StatusCreated {
		t.Fatalf("setup: create task status = %d; body: %s", w.Code, w.Body.String())
	}
	var task map[string]any
	json.Unmarshal(w.Body.Bytes(), &task)
	return task["id"].(float64)
}

func TestPatchTask_UpdateTitle(t *testing.T) {
	r := newAgentRouter()
	id := createTestTask(t, r, "original", "2026-06-01")

	w := patch(r, fmt.Sprintf("/api/agent/tasks/%.0f", id),
		map[string]any{"title": "updated"}, "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var task map[string]any
	json.Unmarshal(w.Body.Bytes(), &task)
	if task["title"] != "updated" {
		t.Errorf("title = %v, want 'updated'", task["title"])
	}
}

func TestPatchTask_UpdateStatus(t *testing.T) {
	r := newAgentRouter()
	id := createTestTask(t, r, "status test", "2026-06-02")

	w := patch(r, fmt.Sprintf("/api/agent/tasks/%.0f", id),
		map[string]any{"status": "done"}, "tok")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d; body: %s", w.Code, w.Body.String())
	}
	var task map[string]any
	json.Unmarshal(w.Body.Bytes(), &task)
	if task["status"] != "done" {
		t.Errorf("status = %v, want 'done'", task["status"])
	}
}

func TestPatchTask_NotFound(t *testing.T) {
	r := newAgentRouter()
	w := patch(r, "/api/agent/tasks/99999", map[string]any{"title": "x"}, "tok")
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestPatchTask_InvalidID(t *testing.T) {
	r := newAgentRouter()
	w := patch(r, "/api/agent/tasks/abc", map[string]any{"title": "x"}, "tok")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestPatchTask_NoToken(t *testing.T) {
	r := newAgentRouter()
	w := patch(r, "/api/agent/tasks/1", map[string]any{"title": "x"}, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
