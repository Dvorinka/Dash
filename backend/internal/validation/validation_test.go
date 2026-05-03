package validation

import "testing"

func TestAbsoluteHTTP(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"http", "http://localhost:3000", false},
		{"https", "https://example.com", false},
		{"relative", "/pihole", true},
		{"ftp", "ftp://example.com", true},
		{"missing host", "https://", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := AbsoluteHTTP(tt.raw, "url")
			if (err != nil) != tt.wantErr {
				t.Fatalf("AbsoluteHTTP() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestName(t *testing.T) {
	if got, err := Name("  Pi-hole  "); err != nil || got != "Pi-hole" {
		t.Fatalf("Name() = %q, %v", got, err)
	}
	if _, err := Name(""); err == nil {
		t.Fatal("Name() accepted empty value")
	}
}
