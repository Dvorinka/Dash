package services

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"dash/backend/internal/store"
	"dash/backend/internal/validation"
	"dash/backend/internal/widgets"
)

func NormalizeService(input store.ServiceInput) (store.ServiceInput, error) {
	name, err := validation.Name(input.Name)
	if err != nil {
		return store.ServiceInput{}, err
	}
	input.Name = name
	if input.GroupID != nil {
		if _, err := uuid.Parse(*input.GroupID); err != nil {
			return store.ServiceInput{}, errors.New("groupId must be a UUID")
		}
	}
	if input.IconAssetID != nil {
		if _, err := uuid.Parse(*input.IconAssetID); err != nil {
			return store.ServiceInput{}, errors.New("iconAssetId must be a UUID")
		}
	}
	if input.IconURL != nil && input.IconAssetID != nil {
		return store.ServiceInput{}, errors.New("iconUrl and iconAssetId are mutually exclusive")
	}

	if input.IconURL != nil {
		iconURL, err := validation.OptionalAbsoluteHTTP(*input.IconURL, "iconUrl")
		if err != nil {
			return store.ServiceInput{}, err
		}
		input.IconURL = iconURL
	}
	if len(input.URLs) == 0 {
		return store.ServiceInput{}, errors.New("service requires at least one URL")
	}

	primaryCount := 0
	seenURLIDs := map[string]struct{}{}
	for i := range input.URLs {
		label, err := validation.Label(input.URLs[i].Label)
		if err != nil {
			return store.ServiceInput{}, err
		}
		if err := validation.URLKind(input.URLs[i].Kind); err != nil {
			return store.ServiceInput{}, err
		}
		serviceURL, err := validation.AbsoluteHTTP(input.URLs[i].URL, "url")
		if err != nil {
			return store.ServiceInput{}, err
		}
		if input.URLs[i].ID != nil {
			if _, err := uuid.Parse(*input.URLs[i].ID); err != nil {
				return store.ServiceInput{}, errors.New("service URL id must be a UUID")
			}
			if _, ok := seenURLIDs[*input.URLs[i].ID]; ok {
				return store.ServiceInput{}, errors.New("duplicate service URL id")
			}
			seenURLIDs[*input.URLs[i].ID] = struct{}{}
		}
		input.URLs[i].Label = label
		input.URLs[i].URL = serviceURL
		if input.URLs[i].IsPrimary {
			primaryCount++
		}
	}
	if primaryCount > 1 {
		return store.ServiceInput{}, errors.New("only one primary URL allowed")
	}
	if primaryCount == 0 {
		input.URLs[0].IsPrimary = true
	}
	return input, nil
}

func NormalizeGroup(input store.GroupInput, requireName bool) (store.GroupInput, error) {
	if input.Name == nil {
		if requireName {
			return store.GroupInput{}, errors.New("name is required")
		}
		return input, nil
	}
	name, err := validation.Name(*input.Name)
	if err != nil {
		return store.GroupInput{}, err
	}
	input.Name = &name
	return input, nil
}

func NormalizeWidget(input store.WidgetInput, requireType bool) (store.WidgetInput, error) {
	if requireType || input.Type != "" {
		if err := validation.WidgetType(input.Type); err != nil {
			return store.WidgetInput{}, err
		}
	}
	if input.Title != "" {
		title, err := validation.Name(input.Title)
		if err != nil {
			return store.WidgetInput{}, err
		}
		input.Title = title
	} else if requireType {
		return store.WidgetInput{}, errors.New("title is required")
	}
	if len(input.Config) == 0 {
		input.Config = json.RawMessage(`{}`)
	}
	if !json.Valid(input.Config) {
		return store.WidgetInput{}, errors.New("config must be valid JSON")
	}
	if input.Type != "" {
		if err := ValidateWidgetConfig(input.Type, input.Config); err != nil {
			return store.WidgetInput{}, err
		}
	}
	return input, nil
}

func NormalizeWidgetPatch(current store.WidgetInstance, input store.WidgetInput) (store.WidgetInput, error) {
	widgetType := input.Type
	if widgetType == "" {
		widgetType = current.Type
	}
	title := input.Title
	if title == "" {
		title = current.Title
	}
	config := input.Config
	if len(config) == 0 {
		config = current.Config
	}
	normalized, err := NormalizeWidget(store.WidgetInput{
		Type:    widgetType,
		Title:   title,
		Enabled: input.Enabled,
		Config:  config,
	}, false)
	if err != nil {
		return store.WidgetInput{}, err
	}
	return normalized, nil
}

func ValidateWidgetConfig(widgetType string, raw json.RawMessage) error {
	tmpl, ok := widgets.GetTemplate(widgetType)
	if !ok {
		return fmt.Errorf("unsupported widget type %q", widgetType)
	}
	if tmpl.Validate != nil {
		return tmpl.Validate(raw)
	}
	return nil
}

func MaskWidget(widget store.WidgetInstance) store.WidgetInstance {
	if (widget.Type != "pihole" && widget.Type != "memos") || len(widget.Config) == 0 {
		return widget
	}
	var cfg map[string]any
	if err := json.Unmarshal(widget.Config, &cfg); err != nil {
		widget.Config = json.RawMessage(`{}`)
		return widget
	}
	if _, ok := cfg["apiToken"]; ok {
		cfg["apiToken"] = "********"
	}
	masked, err := json.Marshal(cfg)
	if err != nil {
		widget.Config = json.RawMessage(`{}`)
		return widget
	}
	widget.Config = masked
	return widget
}

func MaskWidgets(widgets []store.WidgetInstance) []store.WidgetInstance {
	for i := range widgets {
		widgets[i] = MaskWidget(widgets[i])
	}
	return widgets
}
