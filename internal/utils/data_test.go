package utils

import (
	"testing"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"no ansi", "plain text", "plain text"},
		{"with ansi color", "\x1b[31mred text\x1b[0m", "red text"},
		{"multiple ansi", "\x1b[1mbold\x1b[0m \x1b[32mgreen\x1b[0m", "bold green"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestCleanRaw(t *testing.T) {
	m := map[string]interface{}{
		"plain":  "text",
		"ansi":   "\x1b[31mred\x1b[0m",
		"number":  float64(42),
		"nested": map[string]interface{}{
			"ansi_nested": "\x1b[1mbold\x1b[0m",
		},
		"array": []interface{}{"\x1b[32mgreen\x1b[0m", "plain"},
	}

	CleanRaw(m)

	if m["plain"] != "text" {
		t.Errorf("plain text changed: %v", m["plain"])
	}
	if m["ansi"] != "red" {
		t.Errorf("ansi not stripped: %v", m["ansi"])
	}
	if nested, ok := m["nested"].(map[string]interface{}); ok {
		if nested["ansi_nested"] != "bold" {
			t.Errorf("nested ansi not stripped: %v", nested["ansi_nested"])
		}
	}
	if arr, ok := m["array"].([]interface{}); ok {
		if arr[0] != "green" {
			t.Errorf("array ansi not stripped: %v", arr[0])
		}
	}
}

func TestStructToMap(t *testing.T) {
	type TestStruct struct {
		ID     int    `json:"id"`
		Name   string `json:"name"`
		Status string `json:"status"`
	}

	test := &TestStruct{
		ID:     123,
		Name:   "test",
		Status: "success",
	}

	m, err := StructToMap(test)
	if err != nil {
		t.Fatalf("StructToMap() failed: %v", err)
	}

	if m["id"] != float64(123) {
		t.Errorf("expected id=123, got %v", m["id"])
	}
	if m["name"] != "test" {
		t.Errorf("expected name=test, got %v", m["name"])
	}
}
