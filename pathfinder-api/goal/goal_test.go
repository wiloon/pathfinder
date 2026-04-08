package goal_test

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"

	"pathfinder-api/goal"
	"pathfinder-api/storage"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	storage.Init(":memory:")
	os.Exit(m.Run())
}

func newRouter() *gin.Engine {
	r := gin.New()
	r.POST("/api/goals", goal.CreateGoal)
	r.GET("/api/goals", goal.ListGoals)
	r.PUT("/api/goals/:id", goal.UpdateGoal)
	r.DELETE("/api/goals/:id", goal.DeleteGoal)
	r.PUT("/api/goals/:id/primary", goal.SetPrimaryGoal)
	return r
}

func multipartBody(fields map[string]string) (*bytes.Buffer, string) {
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	mw.Close()
	return &b, mw.FormDataContentType()
}

func TestCreateGoal_SecondaryGoalSucceeds(t *testing.T) {
	r := newRouter()
	body, ct := multipartBody(map[string]string{
		"title":  "Learn Go testing",
		"type":   "secondary",
		"status": "active",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["title"] != "Learn Go testing" {
		t.Errorf("expected title='Learn Go testing', got %v", resp["title"])
	}
	if resp["type"] != "secondary" {
		t.Errorf("expected type=secondary, got %v", resp["type"])
	}
}

func TestCreateGoal_DefaultsTypeToSecondary(t *testing.T) {
	r := newRouter()
	body, ct := multipartBody(map[string]string{"title": "No type goal"})
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["type"] != "secondary" {
		t.Errorf("expected default type=secondary, got %v", resp["type"])
	}
}

func TestCreateGoal_MissingTitleReturns400(t *testing.T) {
	r := newRouter()
	body, ct := multipartBody(map[string]string{"type": "secondary"})
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestListGoals_ReturnsCreatedGoals(t *testing.T) {
	r := newRouter()
	body, ct := multipartBody(map[string]string{
		"title": "List test goal",
		"type":  "secondary",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set("Content-Type", ct)
	r.ServeHTTP(httptest.NewRecorder(), req)

	req2 := httptest.NewRequest(http.MethodGet, "/api/goals", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req2)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var goals []map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &goals)
	if len(goals) == 0 {
		t.Error("expected at least one goal in list")
	}
}

func TestUpdateGoal_ChangesTitle(t *testing.T) {
	r := newRouter()
	body, ct := multipartBody(map[string]string{
		"title": "Original Title",
		"type":  "secondary",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	updateBody, _ := json.Marshal(map[string]string{"title": "Updated Title"})
	req2 := httptest.NewRequest(http.MethodPut, "/api/goals/"+strconv.Itoa(id), bytes.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &updated)
	if updated["title"] != "Updated Title" {
		t.Errorf("expected title='Updated Title', got %v", updated["title"])
	}
}

func TestUpdateGoal_NotFoundReturns404(t *testing.T) {
	r := newRouter()
	updateBody, _ := json.Marshal(map[string]string{"title": "Ghost"})
	req := httptest.NewRequest(http.MethodPut, "/api/goals/999999", bytes.NewReader(updateBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestDeleteGoal_RemovesGoal(t *testing.T) {
	r := newRouter()
	body, ct := multipartBody(map[string]string{
		"title": "Goal to delete",
		"type":  "secondary",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/goals", body)
	req.Header.Set("Content-Type", ct)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created)
	id := int(created["id"].(float64))

	req2 := httptest.NewRequest(http.MethodDelete, "/api/goals/"+strconv.Itoa(id), nil)
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	var count int64
	storage.DB.Model(&storage.Goal{}).Where("id = ?", id).Count(&count)
	if count != 0 {
		t.Errorf("expected goal to be deleted, but count=%d", count)
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
