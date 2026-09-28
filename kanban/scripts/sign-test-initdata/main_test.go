package main

import (
	"testing"

	"kanban/middleware"
)

func TestBuildInitDataProducesVerifiableSignature(t *testing.T) {
	initData, err := buildInitData("api-check-token", 9001, "api-owner", "API Owner", 1780000000)
	if err != nil {
		t.Fatal(err)
	}
	if err := middleware.ValidateInitData(initData, "api-check-token"); err != nil {
		t.Fatalf("ValidateInitData() error = %v", err)
	}
	user, err := middleware.ParseUser(initData)
	if err != nil {
		t.Fatalf("ParseUser() error = %v", err)
	}
	if user.UserID != 9001 || user.DisplayName != "API Owner" {
		t.Fatalf("user = %+v", user)
	}
}
