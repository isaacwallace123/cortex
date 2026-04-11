package sqlite

import (
	"strconv"
	"strings"
)

// Minimal JSON field extractors — avoids encoding/json import and reflection
// for the simple key:value payloads brain emits.

func jsonString(payload, key string) string {
	needle := `"` + key + `"`
	idx := strings.Index(payload, needle)
	if idx < 0 {
		return ""
	}
	rest := payload[idx+len(needle):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return ""
	}
	rest = strings.TrimSpace(rest[colon+1:])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	end := strings.Index(rest[1:], `"`)
	if end < 0 {
		return ""
	}
	return rest[1 : end+1]
}

func jsonInt(payload, key string) int {
	needle := `"` + key + `"`
	idx := strings.Index(payload, needle)
	if idx < 0 {
		return 0
	}
	rest := payload[idx+len(needle):]
	colon := strings.Index(rest, ":")
	if colon < 0 {
		return 0
	}
	rest = strings.TrimSpace(rest[colon+1:])
	end := strings.IndexAny(rest, ",}")
	if end < 0 {
		end = len(rest)
	}
	v, _ := strconv.Atoi(strings.TrimSpace(rest[:end]))
	return v
}
