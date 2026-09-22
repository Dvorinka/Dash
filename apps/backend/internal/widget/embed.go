package widget

import (
	"context"
	"encoding/json"
	"fmt"
)

// Embed renders a sandboxed iframe client-side — registered only so
// /api/widgets/types lists it. The browser fetches the framed page, not the
// server (remote content may be LAN-only or need the user's cookies).
type embed struct{}

func init() { Register(embed{}) }

func (embed) Meta() Type {
	return Type{
		Type:        "embed",
		Name:        "Embed",
		Description: "Embed any page in a tile",
		Local:       true,
		Fields: []Field{
			{Key: "url", Label: "URL", Placeholder: "https://…", Required: true},
		},
	}
}

func (embed) Fetch(context.Context, json.RawMessage) (any, error) {
	return nil, fmt.Errorf("embed renders client-side")
}
