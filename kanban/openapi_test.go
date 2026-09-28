package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"kanban/config"
	"kanban/services"
)

func TestOpenAPIListsRegisteredAPIRoutes(t *testing.T) {
	spec, err := os.ReadFile(filepath.Join("openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	document := string(spec)
	if !strings.Contains(document, "openapi: 3.1.0") {
		t.Fatal("openapi.yaml must declare OpenAPI 3.1")
	}

	router := setupRouter(&config.Config{}, services.NewMaxBot("", "", ""))
	pathParameters := regexp.MustCompile(`:([A-Za-z0-9_]+)`)
	undocumented := make([]string, 0)
	for _, route := range router.Routes() {
		path := pathParameters.ReplaceAllString(route.Path, `{$1}`)
		if strings.HasPrefix(path, "/api/") {
			path = "/max-kanban" + path
		} else if !strings.HasPrefix(path, "/max-kanban/api/") && path != "/max-kanban/webhook" {
			continue
		}
		if !strings.Contains(document, "\n  "+path+":") {
			undocumented = append(undocumented, route.Method+" "+route.Path)
		}
	}
	if len(undocumented) > 0 {
		t.Fatalf("undocumented routes: %s", strings.Join(undocumented, ", "))
	}

	for _, operation := range []string{
		"validateMaxAuth",
		"createOrganization",
		"createBoard",
		"createTask",
		"moveTask",
		"maxWebhook",
	} {
		if !strings.Contains(document, "operationId: "+operation) {
			t.Fatalf("openapi.yaml must document operation %s", operation)
		}
	}
	for _, required := range []string{
		"name: X-Max-InitData",
		"name: X-Max-Bot-Api-Secret",
		"Organization:",
		"BoardDetail:",
		"TaskDetail:",
		"MaxUpdate:",
	} {
		if !strings.Contains(document, required) {
			t.Fatalf("openapi.yaml must contain %q", required)
		}
	}
}
