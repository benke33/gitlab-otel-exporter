package utils

import (
	"fmt"

	"go.opentelemetry.io/otel/attribute"
)

// FlattenMap converts nested map to OpenTelemetry attributes
func FlattenMap(prefix string, m map[string]interface{}) []attribute.KeyValue {
	var attrs []attribute.KeyValue
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			attrs = append(attrs, FlattenMap(key, val)...)
		case []interface{}:
			if len(val) > 0 {
				if str, ok := val[0].(string); ok {
					attrs = append(attrs, attribute.String(key, str))
				}
			}
		case string:
			attrs = append(attrs, attribute.String(key, val))
		case float64:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%.0f", val)))
		case bool:
			attrs = append(attrs, attribute.String(key, fmt.Sprintf("%v", val)))
		case nil:
			attrs = append(attrs, attribute.String(key, "None"))
		}
	}
	return attrs
}
