package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"

	"github.com/goforj/notification"
)

// Signature computes the robot-specific HMAC without exposing signing material.
func Signature(key, message string) string {
	mac := hmac.New(sha256.New, []byte(key))
	_, _ = mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// RobotAck requires the selected native response code to be present and zero.
func RobotAck(field string) func([]byte) error {
	return func(data []byte) error {
		var fields map[string]json.RawMessage
		if json.Unmarshal(data, &fields) != nil {
			return notification.ErrOutcomeUnknown
		}
		var code int
		value, ok := fields[field]
		if !ok || json.Unmarshal(value, &code) != nil || string(value) == "null" {
			return notification.ErrOutcomeUnknown
		}
		if code != 0 {
			return notification.ErrPermanent
		}
		return nil
	}
}
