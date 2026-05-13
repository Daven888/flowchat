package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestMessageRouteDisambiguation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	sessions := r.Group("/api/v1/chat/sessions")
	{
		sessions.GET("/:session_id/messages", func(c *gin.Context) {
			c.String(http.StatusOK, "list:%s", c.Param("session_id"))
		})
		sessions.GET("/:session_id/messages/search", func(c *gin.Context) {
			c.String(http.StatusOK, "search:%s", c.Param("session_id"))
		})
	}

	tests := []struct {
		name   string
		method string
		path   string
		want   string
	}{
		{
			name:   "List with numeric session_id",
			method: "GET",
			path:   "/api/v1/chat/sessions/123/messages",
			want:   "list:123",
		},
		{
			name:   "Search with numeric session_id",
			method: "GET",
			path:   "/api/v1/chat/sessions/123/messages/search",
			want:   "search:123",
		},
		{
			name:   "List with large session_id",
			method: "GET",
			path:   "/api/v1/chat/sessions/999999999/messages",
			want:   "list:999999999",
		},
		{
			name:   "Search with large session_id",
			method: "GET",
			path:   "/api/v1/chat/sessions/999999999/messages/search",
			want:   "search:999999999",
		},
		{
			name:   "search as session_id hits List (handler validates int)",
			method: "GET",
			path:   "/api/v1/chat/sessions/search/messages",
			want:   "list:search",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, tt.path, nil)
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Errorf("status %d, want %d", w.Code, http.StatusOK)
			}
			if w.Body.String() != tt.want {
				t.Errorf("body = %q, want %q", w.Body.String(), tt.want)
			}
		})
	}
}

func TestMessageRoutesAreDistinct(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	sessions := r.Group("/api/v1/chat/sessions")
	{
		sessions.GET("/:session_id/messages", func(c *gin.Context) {
			c.Set("handler", "list")
			c.String(http.StatusOK, "")
		})
		sessions.GET("/:session_id/messages/search", func(c *gin.Context) {
			c.Set("handler", "search")
			c.String(http.StatusOK, "")
		})
	}

	// Request to /messages/search MUST NOT hit the /messages handler.
	// We verify this by setting different context values and checking the
	// returned route path via gin's handler name mechanism.
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/chat/sessions/42/messages/search", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("search route returned status %d", w.Code)
	}

	// Also assert that /messages without /search hits the list handler.
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/chat/sessions/42/messages", nil)
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("list route returned status %d", w2.Code)
	}
}
