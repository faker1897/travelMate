package chat

import (
	"encoding/json"
	"strings"
)

func normalizeAssistantText(raw string) string {
	text := strings.TrimSpace(raw)
	if text == "" {
		return raw
	}

	for range 3 {
		var payload map[string]any
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			break
		}

		next, ok := firstStringField(payload, "response", "answer", "result", "content")
		if !ok {
			break
		}
		next = strings.TrimSpace(next)
		if next == "" || next == text {
			break
		}
		text = next
	}

	return text
}

func firstStringField(payload map[string]any, fields ...string) (string, bool) {
	for _, field := range fields {
		value, ok := payload[field]
		if !ok {
			continue
		}
		if text, ok := value.(string); ok {
			return text, true
		}
	}
	return "", false
}
