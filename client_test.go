package main

import (
	"testing"
)

func TestClassifyAPIError(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantMsg    string
	}{
		{"401 unauthorized", 401, "authentication failed"},
		{"403 forbidden", 403, "permission denied"},
		{"404 not found", 404, "not found"},
		{"500 server error", 500, "unexpected error (HTTP 500)"},
		{"502 bad gateway", 502, "unexpected error (HTTP 502)"},
		{"429 rate limit", 429, "API request failed (HTTP 429)"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ClassifyAPIError("https://mm.example.com", tt.statusCode)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			msg := err.Error()
			if !containsSubstring(msg, tt.wantMsg) {
				t.Errorf("error = %q, want to contain %q", msg, tt.wantMsg)
			}
		})
	}
}

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"no trailing slash", "https://mm.example.com", "https://mm.example.com"},
		{"single trailing slash", "https://mm.example.com/", "https://mm.example.com"},
		{"multiple trailing slashes", "https://mm.example.com///", "https://mm.example.com"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeURL(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeURL(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
