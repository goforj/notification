package provider

import (
	"strings"
	"testing"

	"github.com/goforj/notification"
)

// TestWireUtilities validates bounds and deliberate internal serialization failures.
func TestWireUtilities(t *testing.T) {
	for _, value := range []string{"", strings.Repeat("x", 4097), string([]byte{255}), "secret\n"} {
		if Token(value) {
			t.Fatal("invalid credential permitted")
		}
	}
	if !Token("token") || Text(notification.Message{Body: "body"}) != "body" || Text(notification.Message{Title: "title", Body: "body"}) != "title\nbody" {
		t.Fatal("wire projection")
	}
	if string(JSON(struct {
		Value string `json:"value"`
	}{"<safe>"})) != `{"value":"\u003csafe\u003e"}` {
		t.Fatal("unsafe JSON encoding")
	}
	defer func() {
		if recover() == nil {
			t.Fatal("programming error hidden")
		}
	}()
	JSON(func() {})
}
