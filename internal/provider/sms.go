package provider

import (
	"unicode/utf8"

	"github.com/goforj/notification"
)

// SMSText bounds a single portable SMS submission without truncating or silently splitting it.
func SMSText(message notification.Message) (string, error) {
	text := Text(message)
	if utf8.RuneCountInString(text) > 1600 {
		return "", notification.ErrRender
	}
	return text, nil
}
