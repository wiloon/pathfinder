package user

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"pathfinder-api/middleware"
	"pathfinder-api/storage"
)

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)
	storage.Init(":memory:")
	middleware.InitSession("test-secret-key-for-tests-only")
	sessionStore = middleware.Store
	os.Exit(m.Run())
}

func newTestRouter() *gin.Engine {
	r := gin.New()
	r.POST("/api/auth/register", Register)
	r.POST("/api/auth/login", Login)
	r.POST("/api/auth/logout", Logout)
	r.GET("/api/auth/verify-email", VerifyEmail)
	return r
}

// --- Unit tests for unexported functions ---

func TestHashPassword_ProducesBcryptHash(t *testing.T) {
	hash, err := hashPassword("mysecret")
	if err != nil {
		t.Fatalf("hashPassword error: %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Errorf("expected bcrypt hash starting with $2, got: %s", hash)
	}
}

func TestCheckPassword(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		correct string
		want    bool
	}{
		{"correct password", "secret123", "secret123", true},
		{"wrong password", "wrongpass", "secret123", false},
		{"empty input", "", "secret123", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hashPassword(tt.correct)
			if err != nil {
				t.Fatalf("hashPassword error: %v", err)
			}
			if got := checkPassword(tt.input, hash); got != tt.want {
				t.Errorf("checkPassword(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateToken_LengthAndUniqueness(t *testing.T) {
	tok1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken error: %v", err)
	}
	if len(tok1) != 64 {
		t.Errorf("expected 64-char hex token, got length %d", len(tok1))
	}
	tok2, _ := generateToken()
	if tok1 == tok2 {
		t.Error("tokens should be unique, got identical values")
	}
}

// --- Integration tests for HTTP handlers ---

func TestRegister_CreatesUserWithPendingStatus(t *testing.T) {
	r := newTestRouter()
	body, _ := json.Marshal(map[string]string{
		"username": "reguser1",
		"password": "password123",
		"email":    "reguser1@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp["success"])
	}
	if resp["status"] != "pending" {
		t.Errorf("expected status=pending, got %v", resp["status"])
	}
}

func TestRegister_RejectsDuplicateUsername(t *testing.T) {
	r := newTestRouter()
	first, _ := json.Marshal(map[string]string{
		"username": "dupname",
		"password": "password123",
		"email":    "dupname@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(first))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	second, _ := json.Marshal(map[string]string{
		"username": "dupname",
		"password": "password456",
		"email":    "other@example.com",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(second))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "Username already taken" {
		t.Errorf("expected 'Username already taken', got %v", resp["message"])
	}
}

func TestRegister_RejectsDuplicateEmail(t *testing.T) {
	r := newTestRouter()
	first, _ := json.Marshal(map[string]string{
		"username": "emailuser1",
		"password": "password123",
		"email":    "shared@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(first))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	second, _ := json.Marshal(map[string]string{
		"username": "emailuser2",
		"password": "password456",
		"email":    "shared@example.com",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(second))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["message"] != "Email already registered" {
		t.Errorf("expected 'Email already registered', got %v", resp["message"])
	}
}

func TestRegister_RejectsMissingFields(t *testing.T) {
	r := newTestRouter()
	body, _ := json.Marshal(map[string]string{"username": "onlyname"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestLogin_SuccessWithCorrectCredentials(t *testing.T) {
	r := newTestRouter()
	reg, _ := json.Marshal(map[string]string{
		"username": "loginuser",
		"password": "correct123",
		"email":    "loginuser@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reg))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	login, _ := json.Marshal(map[string]string{
		"username": "loginuser",
		"password": "correct123",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(login))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["success"] != true {
		t.Errorf("expected success=true, got %v", resp)
	}
	if resp["username"] != "loginuser" {
		t.Errorf("expected username=loginuser, got %v", resp["username"])
	}
}

func TestLogin_RejectsWrongPassword(t *testing.T) {
	r := newTestRouter()
	reg, _ := json.Marshal(map[string]string{
		"username": "wrongpwduser",
		"password": "correct123",
		"email":    "wrongpwd@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reg))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	login, _ := json.Marshal(map[string]string{
		"username": "wrongpwduser",
		"password": "wrongpass",
	})
	req2 := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(login))
	req2.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestLogin_RejectsNonExistentUser(t *testing.T) {
	r := newTestRouter()
	login, _ := json.Marshal(map[string]string{
		"username": "ghost",
		"password": "anypassword",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(login))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestVerifyEmail_ValidTokenActivatesAccount(t *testing.T) {
	r := newTestRouter()
	reg, _ := json.Marshal(map[string]string{
		"username": "verifyuser",
		"password": "password123",
		"email":    "verify@example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(reg))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(httptest.NewRecorder(), req)

	var u storage.User
	storage.DB.Where("username = ?", "verifyuser").First(&u)
	if u.VerificationToken == "" {
		t.Skip("verification token is empty")
	}

	req2 := httptest.NewRequest(http.MethodGet, "/api/auth/verify-email?token="+u.VerificationToken, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var updated storage.User
	storage.DB.Where("username = ?", "verifyuser").First(&updated)
	if updated.Status != "active" {
		t.Errorf("expected status=active after verification, got %s", updated.Status)
	}
}

func TestVerifyEmail_InvalidTokenReturns400(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/verify-email?token=invalid-token-xyz", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestVerifyEmail_MissingTokenReturns400(t *testing.T) {
	r := newTestRouter()
	req := httptest.NewRequest(http.MethodGet, "/api/auth/verify-email", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}
