package validation

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

func Name(raw string) (string, error) {
	return boundedText(raw, 1, 80, "name")
}

func Label(raw string) (string, error) {
	return boundedText(raw, 1, 40, "label")
}

func AbsoluteHTTP(raw, field string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", fmt.Errorf("%s is required", field)
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("%s must be absolute URL", field)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("%s must use http or https", field)
	}
	return value, nil
}

func OptionalAbsoluteHTTP(raw, field string) (*string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil, nil
	}
	clean, err := AbsoluteHTTP(value, field)
	if err != nil {
		return nil, err
	}
	return &clean, nil
}

func URLKind(kind string) error {
	switch kind {
	case "local", "external", "custom":
		return nil
	default:
		return errors.New("kind must be local, external, or custom")
	}
}

func WidgetType(kind string) error {
	switch kind {
	case "clock", "image", "pihole", "memos":
		return nil
	default:
		return errors.New("widget type must be clock, image, pihole, or memos")
	}
}

func boundedText(raw string, minLen int, maxLen int, field string) (string, error) {
	value := strings.TrimSpace(raw)
	if len(value) < minLen {
		return "", fmt.Errorf("%s is required", field)
	}
	if len(value) > maxLen {
		return "", fmt.Errorf("%s must be at most %d chars", field, maxLen)
	}
	return value, nil
}
