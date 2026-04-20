package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"pathfinder-api/middleware"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newAgentRouter() *gin.Engine {
	middleware.InitServiceToken("test-token-123", "user-42")
	r := gin.New()
	r.GET("/api/agent/ping", middleware.ServiceTokenAuth(), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"user_id": c.GetString("user_id")})
	})
	return r
}

func TestServiceTokenAuth(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		wantStatus int
		wantUserID string
	}{
		{
			name:       "valid token sets user_id",
			authHeader: "Bearer test-token-123",
			wantStatus: http.StatusOK,
			wantUserID: "user-42",
		},
		{
			name:       "wrong token returns 401",
			authHeader: "Bearer wrong-token",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "missing Authorization header returns 401",
			authHeader: "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed header (no Bearer prefix) returns 401",
			authHeader: "test-token-123",
			wantStatus: http.StatusUnauthorized,
		},
	}

	r := newAgentRouter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/agent/ping", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}
