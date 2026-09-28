package handlers

import "testing"

func TestGenerateSlug(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Рабочая Команда", want: "rabochaya-komanda"},
		{name: "  My Board  ", want: "my-board"},
		{name: "A/B Testing", want: "ab-testing"},
		{name: "!!!", want: "organization"},
	}
	for _, tt := range tests {
		if got := generateSlug(tt.name); got != tt.want {
			t.Errorf("generateSlug(%q) = %q, want %q", tt.name, got, tt.want)
		}
	}
}
