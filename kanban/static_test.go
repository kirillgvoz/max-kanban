package main

import (
	"io/fs"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"kanban/config"
	"kanban/services"

	"github.com/gin-gonic/gin"
)

func TestFrontendAssets(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := setupRouter(&config.Config{}, services.NewMaxBot("", "", ""))

	index, err := fs.ReadFile(frontendFS, "frontend/dist/index.html")
	if err != nil {
		t.Fatalf("ReadFile(index.html) error = %v", err)
	}
	matches := regexp.MustCompile(`(?:src|href)="(/max-kanban/assets/[^"]+)"`).FindAllStringSubmatch(string(index), -1)
	if len(matches) < 2 {
		t.Fatalf("found %d frontend asset references, want at least 2", len(matches))
	}

	for _, match := range matches {
		assetPath := match[1]
		requestPaths := []string{assetPath, strings.TrimPrefix(assetPath, "/max-kanban")}
		for _, requestPath := range requestPaths {
			request := httptest.NewRequest(http.MethodGet, requestPath, nil)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Errorf("GET %s status = %d, want %d", requestPath, response.Code, http.StatusOK)
				continue
			}
			expected, err := fs.ReadFile(frontendFS, "frontend/dist/"+strings.TrimPrefix(assetPath, "/max-kanban/"))
			if err != nil {
				t.Errorf("ReadFile(%s) error = %v", requestPath, err)
				continue
			}
			if response.Body.String() != string(expected) {
				t.Errorf("GET %s returned fallback content instead of %s", requestPath, requestPath)
			}
			contentType := response.Header().Get("Content-Type")
			if strings.HasSuffix(requestPath, ".js") && !strings.HasPrefix(contentType, "application/javascript") {
				t.Errorf("GET %s Content-Type = %q, want application/javascript", requestPath, contentType)
			}
			if strings.HasSuffix(requestPath, ".css") && !strings.HasPrefix(contentType, "text/css") {
				t.Errorf("GET %s Content-Type = %q, want text/css", requestPath, contentType)
			}
		}
	}

	unknown := httptest.NewRequest(http.MethodGet, "/max-kanban/missing-board-route", nil)
	unknownResponse := httptest.NewRecorder()
	router.ServeHTTP(unknownResponse, unknown)
	if unknownResponse.Body.String() != string(index) {
		t.Error("unknown frontend route did not fall back to index.html")
	}
}
