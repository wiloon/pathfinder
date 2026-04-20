package goal_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"pathfinder-api/ai"
	"pathfinder-api/goal"
	"pathfinder-api/middleware"
	"pathfinder-api/storage"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	storage.Init(":memory:")
	ai.Init(ai.Config{})
	middleware.InitServiceToken("tok", "u1")
	os.Exit(m.Run())
}

func newRouter() *gin.Engine {
	r := gin.New()
	injectUser := func(c *gin.Context) { c.Set("user_id", "u1"); c.Next() }
	r.POST("/api/goals", injectUser, goal.CreateGoal)
	r.GET("/api/goals", injectUser, goal.ListGoals)
	r.PUT("/api/goals/:id", injectUser, goal.UpdateGoal)
	r.DELETE("/api/goals/:id", injectUser, goal.DeleteGoal)
	r.PATCH("/api/agent/goals/:id", middleware.ServiceTokenAuth(), goal.PatchGoal)
	return r
}

func postGoal(r *gin.Engine, body map[string]any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/goals", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestCreateGoal_BasicSucceeds(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Learn Go", "weight": 7, "tags": []string{"education"}})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["title"] != "Learn Go" {
		t.Errorf("title = %v, want 'Learn Go'", resp["title"])
	}
	if resp["weight"].(float64) != 7 {
		t.Errorf("weight = %v, want 7", resp["weight"])
	}
}

func TestCreateGoal_DefaultWeight5(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "No weight"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["weight"].(float64) != 5 {
		t.Errorf("weight = %v, want default 5", resp["weight"])
	}
}

func TestCreateGoal_DefaultStatusActive(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Status test"})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["status"] != "active" {
		t.Errorf("status = %v, want 'active'", resp["status"])
	}
}

func TestCreateGoal_MissingTitleReturns400(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"weight": 5})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreateGoal_WeightOutOfRange(t *testing.T) {
	r := newRouter()
	for _, w := range []int{0, 11} {
		rec := postGoal(r, map[string]any{"title": "bad weight", "weight": w})
		if rec.Code != http.StatusBadRequest {
			t.Errorf("weight %d: expected 400, got %d", w, rec.Code)
		}
	}
}

func TestCreateGoal_TagsStoredAsJSON(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Tagged goal", "tags": []string{"career", "health"}})
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	// tags is returned as JSON string from DB
	if resp["tags"] == nil {
		t.Error("expected tags field in response")
	}
}

func TestListGoals_ReturnsCreatedGoals(t *testing.T) {
	r := newRouter()
	postGoal(r, map[string]any{"title": "List test goal"})

	req := httptest.NewRequest(http.MethodGet, "/api/goals", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var goals []map[string]any
	json.Unmarshal(rec.Body.Bytes(), &goals)
	if len(goals) == 0 {
		t.Error("expected at least one goal")
	}
}

func TestUpdateGoal_ChangesTitle(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Original"})
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	body, _ := json.Marshal(map[string]string{"title": "Updated"})
	req := httptest.NewRequest(http.MethodPut, "/api/goals/"+strconv.Itoa(id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated map[string]any
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated["title"] != "Updated" {
		t.Errorf("title = %v, want 'Updated'", updated["title"])
	}
}

func TestUpdateGoal_WeightValidation(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Weight test"})
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	body, _ := json.Marshal(map[string]int{"weight": 11})
	req := httptest.NewRequest(http.MethodPut, "/api/goals/"+strconv.Itoa(id), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestUpdateGoal_NotFoundReturns404(t *testing.T) {
	r := newRouter()
	body, _ := json.Marshal(map[string]string{"title": "Ghost"})
	req := httptest.NewRequest(http.MethodPut, "/api/goals/999999", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteGoal_RemovesGoal(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "To delete"})
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	req := httptest.NewRequest(http.MethodDelete, "/api/goals/"+strconv.Itoa(id), nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var count int64
	storage.DB.Model(&storage.Goal{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Errorf("expected goal deleted, count=%d", count)
	}
}

func TestDeleteGoal_NotFoundReturns404(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodDelete, "/api/goals/999999", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PatchGoal (agent service token) ---

func patchGoalReq(r *gin.Engine, id int, body map[string]any, token string) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPatch, "/api/agent/goals/"+strconv.Itoa(id), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestPatchGoal_UpdatesWeight(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Weight goal"})
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	rec := patchGoalReq(r, id, map[string]any{"weight": 9}, "tok")
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var updated map[string]any
	json.Unmarshal(rec.Body.Bytes(), &updated)
	if updated["weight"].(float64) != 9 {
		t.Errorf("weight = %v, want 9", updated["weight"])
	}
}

func TestPatchGoal_WeightTooHigh(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Range test"})
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	rec := patchGoalReq(r, id, map[string]any{"weight": 11}, "tok")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPatchGoal_MissingWeight(t *testing.T) {
	r := newRouter()
	w := postGoal(r, map[string]any{"title": "Missing weight"})
	var created map[string]any
	json.Unmarshal(w.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	rec := patchGoalReq(r, id, map[string]any{}, "tok")
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestPatchGoal_NotFound(t *testing.T) {
	r := newRouter()
	rec := patchGoalReq(r, 999999, map[string]any{"weight": 5}, "tok")
	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestPatchGoal_NoToken(t *testing.T) {
	r := newRouter()
	rec := patchGoalReq(r, 1, map[string]any{"weight": 5}, "")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
