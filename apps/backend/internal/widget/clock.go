package widget

import (
	"context"
	"encoding/json"
	"fmt"
)

// Clock renders client-side — it registers only so /api/widgets/types lists
// it and the add-widget dialog can offer it like any other type.
type clock struct{}

func init() { register(clock{}) }

func (clock) Meta() Type {
	return Type{
		Type:        "clock",
		Name:        "Clock",
		Description: "Local time in any timezone",
		Local:       true,
		Fields: []Field{
			{Key: "timezone", Label: "Timezone", Placeholder: "Europe/Prague — blank = local"},
		},
	}
}

func (clock) Fetch(context.Context, json.RawMessage) (any, error) {
	return nil, fmt.Errorf("clock renders client-side")
}
