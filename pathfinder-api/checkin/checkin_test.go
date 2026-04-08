package checkin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pathfinder-api/checkin"
	"pathfinder-api/storage"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	storage.Init(":memory:")
	os.Exit(m.Run())
}

func newRouter() *gin.Engine {
	r := gin.New()
	r.GET("/api/checkin/today", checkin.GetTodayCheckin)
	r.POST("/api/checkin", checkin.SubmitCheckin)
	return r
}

func TestGetTodayCheckin_ReturnsEmptyStructWhenNoData(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/checkin/today", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["user_id"] != "local" {
		t.Errorf("expected user_id=local, got %v", resp["user_id"])
	}
	today := time.Now().Format("2006-01-02")
	if resp["date"] != today {
		t.Errorf("expected date=%s, got %v", today, resp["date"])
	}
}

func TestSubmitCheckin_CreatesCheckin(t *testing.T) {
	r := newRouter()
	today := time.Now().Format("2006-01-02")
	body, _ := json.Marshal(map[string]string{
		"date":           today,
		"completed":      "Finished writing tests",
		"blocked":        "None",
		"tomorrow_focus": "Deploy and monitor",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/checkin", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &resp)

	checkinData, ok := resp["checkin"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected checkin object in response, got %T", resp["checkin"])
	}
	if checkinData["completed"] != "Finished writing tests" {
		t.Errorf("expected completed='Finished writing tests', got %v", checkinData["completed"])
	}
}

func TestSubmitCheckin_UpsertUpdatesExisting(t *testing.T) {
	r := newRouter()
	today := time.Now().Format("2006-01-02")

	first, _ := json.Marshal(map[string]string{
		"date":      today,
		"completed": "First submission",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/checkin", bytes.NewReader(first))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	second, _ := json.Marshal(map[string]string{
		"date":      today,
		"completed": "Updated submission",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/checkin", bytes.NewReader(second))
	req2.Header.Set("Content-Type", "application/json")
	rec2 := httptest.NewRecorder()
	r.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec2.Code)
	}

	var count int64
	storage.DB.Model(&storage.CheckIn{}).Where("user_id = ? AND date = ?", "local", today).Count(&count)
	if count != 1 {
		t.Errorf("expected exactly 1 check-in record after upsert, got %d", count)
	}
}

func TestSubmitCheckin_InvalidBodyReturns400(t *testing.T) {
	r := newRouter()
	req := httptest.NewRequest(http.MethodPost, "/api/checkin", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}
