package cors

import (
	"net/http"
	"testing"
)

func TestCheckerIsAllowed(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		origin  string
		want    bool
	}{
		{"wildcard allows all", []string{"*"}, "http://evil.com", true},
		{"empty origin always allowed", []string{"http://localhost:8080"}, "", true},
		{"exact match", []string{"http://localhost:8080"}, "http://localhost:8080", true},
		{"exact mismatch", []string{"http://localhost:8080"}, "http://localhost:3000", false},
		{"port wildcard matches", []string{"http://localhost:*"}, "http://localhost:5173", true},
		{"port wildcard different port", []string{"http://localhost:*"}, "http://localhost:8080", true},
		{"port wildcard wrong host", []string{"http://localhost:*"}, "http://example.com:5173", false},
		{"port wildcard no port", []string{"http://localhost:*"}, "http://localhost", false},
		{"port wildcard non-digit port", []string{"http://localhost:*"}, "http://localhost:abc", false},
		{"127.0.0.1 wildcard", []string{"http://127.0.0.1:*"}, "http://127.0.0.1:3000", true},
		{"multiple patterns", []string{"http://localhost:*", "http://127.0.0.1:*"}, "http://127.0.0.1:8080", true},
		{"multiple patterns miss", []string{"http://localhost:*", "http://127.0.0.1:*"}, "http://example.com:8080", false},
		{"default origins", DefaultOrigins, "http://localhost:5173", true},
		{"default origins reject external", DefaultOrigins, "http://evil.com:5173", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChecker(tt.origins)
			if got := c.IsAllowed(tt.origin); got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCheckerCORSOrigin(t *testing.T) {
	tests := []struct {
		name    string
		origins []string
		origin  string
		want    string
	}{
		{"wildcard returns star", []string{"*"}, "http://anything.com", "*"},
		{"allowed returns origin", []string{"http://localhost:*"}, "http://localhost:5173", "http://localhost:5173"},
		{"disallowed returns empty", []string{"http://localhost:*"}, "http://evil.com:5173", ""},
		{"empty origin returns empty", []string{"http://localhost:*"}, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChecker(tt.origins)
			if got := c.CORSOrigin(tt.origin); got != tt.want {
				t.Errorf("CORSOrigin(%q) = %q, want %q", tt.origin, got, tt.want)
			}
		})
	}
}

func TestCheckerCheckOriginFunc(t *testing.T) {
	c := NewChecker([]string{"http://localhost:*"})
	fn := c.CheckOriginFunc()

	allowed := &http.Request{Header: http.Header{"Origin": []string{"http://localhost:8080"}}}
	if !fn(allowed) {
		t.Error("expected localhost:8080 to be allowed")
	}

	denied := &http.Request{Header: http.Header{"Origin": []string{"http://evil.com:8080"}}}
	if fn(denied) {
		t.Error("expected evil.com:8080 to be denied")
	}

	noOrigin := &http.Request{Header: http.Header{}}
	if !fn(noOrigin) {
		t.Error("expected no Origin header to be allowed")
	}
}
