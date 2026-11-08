package notification

import (
	"context"
	"strings"
	"unicode/utf8"
)

// TextPayload contains bounded plain text for log and webhook channels.
type TextPayload struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
}

// Validate bounds UTF-8 content before transport encoding.
func (p TextPayload) Validate() error {
	if !utf8.ValidString(p.Title) || !utf8.ValidString(p.Body) || len(p.Title) > 256 || strings.TrimSpace(p.Body) == "" || len(p.Body) > 16384 {
		return ErrRender
	}
	return nil
}

// TextRenderer uses an application-owned renderer to keep business wording out of drivers.
type TextRenderer = RendererFunc[TextPayload]

// Literal returns a reviewed constant text renderer for configured system routes.
func Literal(title, body string) TextRenderer {
	return func(context.Context, RenderRequest) (TextPayload, error) { return TextPayload{title, body}, nil }
}
