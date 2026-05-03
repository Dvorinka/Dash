package services

import (
	"encoding/json"
	"strings"
	"testing"

	"dash/backend/internal/store"
)

func TestNormalizeServicePrimaryFallback(t *testing.T) {
	input := store.ServiceInput{
		Name: " Router ",
		URLs: []store.ServiceURLInput{
			{Label: "local", Kind: "local", URL: "http://router.local"},
		},
	}
	got, err := NormalizeService(input)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Router" {
		t.Fatalf("name = %q", got.Name)
	}
	if !got.URLs[0].IsPrimary {
		t.Fatal("first URL not made primary")
	}
}

func TestNormalizeServiceRejectsTwoPrimary(t *testing.T) {
	_, err := NormalizeService(store.ServiceInput{
		Name: "Router",
		URLs: []store.ServiceURLInput{
			{Label: "local", Kind: "local", URL: "http://router.local", IsPrimary: true},
			{Label: "wan", Kind: "external", URL: "https://router.example.com", IsPrimary: true},
		},
	})
	if err == nil {
		t.Fatal("accepted two primary URLs")
	}
}

func TestMaskWidget(t *testing.T) {
	widget := store.WidgetInstance{
		Type:   "pihole",
		Config: json.RawMessage(`{"baseUrl":"http://pihole.local","apiToken":"secret"}`),
	}
	got := MaskWidget(widget)
	if strings.Contains(string(got.Config), "secret") {
		t.Fatalf("token leaked: %s", got.Config)
	}
}

func TestNormalizeWidgetPatchValidatesCurrentType(t *testing.T) {
	current := store.WidgetInstance{
		Type:   "image",
		Title:  "Photo",
		Config: json.RawMessage(`{"imageUrl":"https://example.com/a.png"}`),
	}
	_, err := NormalizeWidgetPatch(current, store.WidgetInput{
		Config: json.RawMessage(`{"imageUrl":"/relative.png"}`),
	})
	if err == nil {
		t.Fatal("accepted invalid image config without explicit type")
	}
}
