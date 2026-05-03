package widgets

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"dash/backend/internal/store"
)

type Registry struct {
	store    *store.Store
	client   *http.Client
	cacheTTL time.Duration
}

func NewRegistry(st *store.Store, timeout time.Duration, cacheTTL time.Duration) *Registry {
	return &Registry{
		store:    st,
		client:   &http.Client{Timeout: timeout},
		cacheTTL: cacheTTL,
	}
}

func (r *Registry) Data(ctx context.Context, widget store.WidgetInstance) (store.WidgetData, error) {
	cached, err := r.store.WidgetData(ctx, widget.ID)
	if err == nil && store.Fresh(cached, time.Now()) {
		return cached, nil
	}

	tmpl, ok := GetTemplate(widget.Type)
	if !ok {
		return store.WidgetData{WidgetID: widget.ID, Status: "fresh", Data: json.RawMessage(`{}`)}, nil
	}
	if !tmpl.NeedsDataFetch {
		if err == nil {
			return cached, nil
		}
		return store.WidgetData{WidgetID: widget.ID, Status: "fresh", Data: json.RawMessage(`{}`)}, nil
	}
	return r.Refresh(ctx, widget)
}

func (r *Registry) Refresh(ctx context.Context, widget store.WidgetInstance) (store.WidgetData, error) {
	now := time.Now()

	tmpl, ok := GetTemplate(widget.Type)
	if !ok || !tmpl.NeedsDataFetch {
		data := store.WidgetData{
			WidgetID:  widget.ID,
			Status:    "fresh",
			Data:      json.RawMessage(`{}`),
			FetchedAt: &now,
			ExpiresAt: ptr(now.Add(r.cacheTTL)),
		}
		return data, r.store.SaveWidgetData(ctx, data)
	}

	payload, err := tmpl.Fetch(ctx, r.client, widget.Config)
	if err != nil {
		message := "[ERROR: " + err.Error() + "]"
		if cached, cacheErr := r.store.WidgetData(ctx, widget.ID); cacheErr == nil && len(cached.Data) > 0 {
			cached.Status = "stale"
			cached.Error = &message
			cached.ExpiresAt = ptr(now.Add(r.cacheTTL))
			_ = r.store.SaveWidgetData(ctx, cached)
			return cached, nil
		}
		data := store.WidgetData{
			WidgetID:  widget.ID,
			Status:    "error",
			Error:     &message,
			FetchedAt: &now,
			ExpiresAt: ptr(now.Add(r.cacheTTL)),
		}
		_ = r.store.SaveWidgetData(ctx, data)
		return data, nil
	}

	data := store.WidgetData{
		WidgetID:  widget.ID,
		Status:    "fresh",
		Data:      payload,
		FetchedAt: &now,
		ExpiresAt: ptr(now.Add(r.cacheTTL)),
	}
	return data, r.store.SaveWidgetData(ctx, data)
}

func ptr[T any](value T) *T {
	return &value
}
