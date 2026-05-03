package store

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"dash/backend/internal/testutil"
)

func TestDashboardLayoutAndGroupDeleteIntegration(t *testing.T) {
	pool := testutil.TestPool(t)
	st := New(pool)
	ctx := context.Background()

	infra, err := st.CreateGroup(ctx, "Infra")
	if err != nil {
		t.Fatal(err)
	}
	media, err := st.CreateGroup(ctx, "Media")
	if err != nil {
		t.Fatal(err)
	}
	service, err := st.CreateService(ctx, ServiceInput{
		GroupID: &infra.ID,
		Name:    "Pi-hole",
		URLs: []ServiceURLInput{
			{Label: "local", Kind: "local", URL: "http://pihole.local", IsPrimary: true},
			{Label: "wan", Kind: "external", URL: "https://pihole.example.com"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	dashboard, err := st.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(dashboard.Groups) != 2 || len(dashboard.Groups[0].Services) != 1 {
		t.Fatalf("unexpected dashboard: %+v", dashboard)
	}

	dashboard, err = st.ApplyLayout(ctx, LayoutInput{
		GroupIDs:          []string{media.ID, infra.ID},
		WidgetIDs:         []string{},
		UngroupedServices: []string{},
		GroupServices: map[string][]string{
			media.ID: {service.ID},
			infra.ID: {},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if dashboard.Groups[0].ID != media.ID || len(dashboard.Groups[0].Services) != 1 {
		t.Fatalf("service was not moved to media group: %+v", dashboard.Groups)
	}

	if err := st.DeleteGroup(ctx, media.ID, false); !errors.Is(err, ErrConflict) {
		t.Fatalf("DeleteGroup() error = %v, want conflict", err)
	}
	if err := st.DeleteGroup(ctx, media.ID, true); err != nil {
		t.Fatal(err)
	}
	service, err = st.Service(ctx, service.ID)
	if err != nil {
		t.Fatal(err)
	}
	if service.GroupID != nil {
		t.Fatalf("service group id = %v, want nil", *service.GroupID)
	}
}

func TestApplyLayoutRejectsPartialServiceSetIntegration(t *testing.T) {
	pool := testutil.TestPool(t)
	st := New(pool)
	ctx := context.Background()

	service, err := st.CreateService(ctx, ServiceInput{
		Name: "Router",
		URLs: []ServiceURLInput{
			{Label: "local", Kind: "local", URL: "http://router.local", IsPrimary: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = st.CreateService(ctx, ServiceInput{
		Name: "NAS",
		URLs: []ServiceURLInput{
			{Label: "local", Kind: "local", URL: "http://nas.local", IsPrimary: true},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = st.ApplyLayout(ctx, LayoutInput{
		GroupIDs:          []string{},
		WidgetIDs:         []string{},
		UngroupedServices: []string{service.ID},
		GroupServices:     map[string][]string{},
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("ApplyLayout() error = %v, want validation", err)
	}
}

func TestWidgetCacheIntegration(t *testing.T) {
	pool := testutil.TestPool(t)
	st := New(pool)
	ctx := context.Background()

	widget, err := st.CreateWidget(ctx, WidgetInput{
		Type:   "clock",
		Title:  "Clock",
		Config: json.RawMessage(`{"timezones":["Europe/Prague"]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	err = st.SaveWidgetData(ctx, WidgetData{
		WidgetID:  widget.ID,
		Status:    "fresh",
		Data:      json.RawMessage(`{"ok":true}`),
		FetchedAt: &now,
		ExpiresAt: &now,
	})
	if err != nil {
		t.Fatal(err)
	}
	data, err := st.WidgetData(ctx, widget.ID)
	if err != nil {
		t.Fatal(err)
	}
	if data.Status != "fresh" || len(data.Data) == 0 {
		t.Fatalf("unexpected widget data: %+v", data)
	}
}
