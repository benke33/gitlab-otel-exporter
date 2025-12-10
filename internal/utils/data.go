package utils

import (
	"encoding/json"
	"regexp"
	"strings"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StructToMap converts a struct to map[string]interface{}
func StructToMap(v interface{}) (map[string]interface{}, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// CleanRaw removes ANSI escape codes from map values recursively
func CleanRaw(m map[string]interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			m[k] = StripANSI(val)
		case map[string]interface{}:
			CleanRaw(val)
		case []interface{}:
			for i, item := range val {
				if str, ok := item.(string); ok {
					val[i] = StripANSI(str)
				}
			}
		}
	}
}

// StripANSI removes ANSI escape codes from string
func StripANSI(s string) string {
	if !strings.Contains(s, "\x1b") {
		return s
	}
	return ansiRegex.ReplaceAllString(s, "")
}
