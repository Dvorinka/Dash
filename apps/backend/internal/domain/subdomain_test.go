package domain

import "testing"

func TestFilterCTNames(t *testing.T) {
	raw := []string{
		"api.example.com\nwww.example.com",
		"*.example.com",
		"example.com",
		"other.com",
		"api.example.com", // dup
		"sub.api.example.com",
		"evil name.example.com",
		"example.com.evil.com",
	}
	got := FilterCTNames(raw, "example.com")
	want := map[string]bool{
		"api.example.com": true, "www.example.com": true,
		"example.com": true, "sub.api.example.com": true,
	}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, n := range got {
		if !want[n] {
			t.Fatalf("unexpected name %q", n)
		}
	}
}
