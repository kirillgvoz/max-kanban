package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCORSMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(CORSMiddleware([]string{"https://kirillgvoz.ru"}))
	router.GET("/private", func(c *gin.Context) { c.String(200, "ok") })

	allowed := httptest.NewRequest("GET", "http://localhost/private", nil)
	allowed.Header.Set("Origin", "https://kirillgvoz.ru")
	allowedResponse := httptest.NewRecorder()
	router.ServeHTTP(allowedResponse, allowed)
	if allowedResponse.Code != 200 || allowedResponse.Header().Get("Access-Control-Allow-Origin") != "https://kirillgvoz.ru" {
		t.Fatalf("allowed origin status=%d headers=%v", allowedResponse.Code, allowedResponse.Header())
	}

	denied := httptest.NewRequest("GET", "http://localhost/private", nil)
	denied.Header.Set("Origin", "https://example.com")
	deniedResponse := httptest.NewRecorder()
	router.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Code != 403 {
		t.Fatalf("disallowed origin status=%d, want 403", deniedResponse.Code)
	}
}
