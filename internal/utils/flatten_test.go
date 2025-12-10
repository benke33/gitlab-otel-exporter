package utils

import (
	"testing"
)

func TestFlattenMap(t *testing.T) {
	m := map[string]interface{}{
		"simple": "value",
		"number": float64(42),
		"nested": map[string]interface{}{
			"key": "nested_value",
		},
	}

	attrs := FlattenMap("", m)
	if len(attrs) != 3 {
		t.Errorf("expected 3 attributes, got %d", len(attrs))
	}

	found := false
	for _, attr := range attrs {
		if attr.Key == "nested.key" && attr.Value.AsString() == "nested_value" {
			found = true
			break
		}
	}
	if !found {
		t.Error("nested attribute not properly flattened")
	}
}

func TestFlattenMapWithArrays(t *testing.T) {
	m := map[string]interface{}{
		"tags": []interface{}{"tag1", "tag2"},
		"empty": []interface{}{},
	}

	attrs := FlattenMap("", m)

	found := false
	for _, attr := range attrs {
		if attr.Key == "tags" && attr.Value.AsString() == "tag1" {
			found = true
			break
		}
	}
	if !found {
		t.Error("array first element not extracted")
	}
}

func TestFlattenMapWithNilValues(t *testing.T) {
	m := map[string]interface{}{
		"null_value": nil,
		"bool_true":  true,
		"bool_false": false,
	}

	attrs := FlattenMap("", m)

	for _, attr := range attrs {
		if attr.Key == "null_value" && attr.Value.AsString() != "None" {
			t.Errorf("nil value not converted to 'None': %v", attr.Value.AsString())
		}
		if attr.Key == "bool_true" && attr.Value.AsString() != "true" {
			t.Errorf("bool true not converted: %v", attr.Value.AsString())
		}
	}
}
