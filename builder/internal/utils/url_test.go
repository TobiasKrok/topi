package utils

import (
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		// Valid URLs
		{
			name:     "localhost with port and path",
			url:      "http://localhost:5448/repo/file/file.yaml",
			expected: true,
		},
		{
			name:     "github URL",
			url:      "https://github.com/tobias/topi",
			expected: true,
		},
		{
			name:     "simple https URL",
			url:      "https://example.com",
			expected: true,
		},
		{
			name:     "simple http URL",
			url:      "http://example.com",
			expected: true,
		},
		{
			name:     "IP address with port",
			url:      "http://192.168.1.1:8080/api/v1",
			expected: true,
		},
		{
			name:     "subdomain",
			url:      "https://api.example.com/v1/users",
			expected: true,
		},
		{
			name:     "localhost without port",
			url:      "http://localhost/",
			expected: true,
		},
		{
			name:     "URL with query parameters",
			url:      "https://example.com/search?q=test&limit=10",
			expected: true,
		},
		{
			name:     "URL with fragment",
			url:      "https://example.com/page#section1",
			expected: true,
		},

		// Invalid URLs
		{
			name:     "empty string",
			url:      "",
			expected: false,
		},
		{
			name:     "invalid scheme",
			url:      "ftp://example.com",
			expected: false,
		},
		{
			name:     "no scheme",
			url:      "example.com",
			expected: false,
		},
		{
			name:     "scheme only",
			url:      "https://",
			expected: false,
		},
		{
			name:     "invalid format",
			url:      "invalid-url",
			expected: false,
		},
		{
			name:     "scheme with no host",
			url:      "http://",
			expected: false,
		},
		{
			name:     "malformed URL",
			url:      "http:///path",
			expected: false,
		},
		{
			name:     "spaces in URL",
			url:      "http://exa mple.com",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateURL(tt.url)
			if result != tt.expected {
				t.Errorf("ValidateURL(%q) = %v, expected %v", tt.url, result, tt.expected)
			}
		})
	}
}
