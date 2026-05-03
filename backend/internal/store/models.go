package store

import (
	"encoding/json"
	"time"
)

type Group struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SortOrder int       `json:"sortOrder"`
	Collapsed bool      `json:"collapsed"`
	Services  []Service `json:"services"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Service struct {
	ID          string       `json:"id"`
	GroupID     *string      `json:"groupId"`
	Name        string       `json:"name"`
	IconURL     *string      `json:"iconUrl"`
	IconAssetID *string      `json:"iconAssetId"`
	SortOrder   int          `json:"sortOrder"`
	URLs        []ServiceURL `json:"urls"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type ServiceURL struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"-"`
	Label     string    `json:"label"`
	Kind      string    `json:"kind"`
	URL       string    `json:"url"`
	SortOrder int       `json:"sortOrder"`
	IsPrimary bool      `json:"isPrimary"`
	CreatedAt time.Time `json:"-"`
	UpdatedAt time.Time `json:"-"`
}

type WidgetInstance struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Title     string          `json:"title"`
	Enabled   bool            `json:"enabled"`
	SortOrder int             `json:"sortOrder"`
	Config    json.RawMessage `json:"config"`
	CreatedAt time.Time       `json:"createdAt"`
	UpdatedAt time.Time       `json:"updatedAt"`
}

type WidgetData struct {
	WidgetID  string          `json:"widgetId"`
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data,omitempty"`
	Error     *string         `json:"error"`
	FetchedAt *time.Time      `json:"fetchedAt"`
	ExpiresAt *time.Time      `json:"expiresAt"`
}

type AssetFile struct {
	ID           string    `json:"id"`
	OriginalName string    `json:"originalName"`
	StoredName   string    `json:"storedName"`
	MimeType     string    `json:"mimeType"`
	SizeBytes    int64     `json:"sizeBytes"`
	PublicPath   string    `json:"publicPath"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Dashboard struct {
	Groups            []Group          `json:"groups"`
	UngroupedServices []Service        `json:"ungroupedServices"`
	Widgets           []WidgetInstance `json:"widgets"`
}

type ServiceURLInput struct {
	ID        *string `json:"id"`
	Label     string  `json:"label"`
	Kind      string  `json:"kind"`
	URL       string  `json:"url"`
	IsPrimary bool    `json:"isPrimary"`
}

type ServiceInput struct {
	GroupID     *string           `json:"groupId"`
	Name        string            `json:"name"`
	IconURL     *string           `json:"iconUrl"`
	IconAssetID *string           `json:"iconAssetId"`
	URLs        []ServiceURLInput `json:"urls"`
}

type GroupInput struct {
	Name      *string `json:"name"`
	Collapsed *bool   `json:"collapsed"`
}

type WidgetInput struct {
	Type    string          `json:"type"`
	Title   string          `json:"title"`
	Enabled *bool           `json:"enabled"`
	Config  json.RawMessage `json:"config"`
}

type LayoutInput struct {
	GroupIDs          []string            `json:"groupIds"`
	WidgetIDs         []string            `json:"widgetIds"`
	UngroupedServices []string            `json:"ungroupedServiceIds"`
	GroupServices     map[string][]string `json:"groupServices"`
}
