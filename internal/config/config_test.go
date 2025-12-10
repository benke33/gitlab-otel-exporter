package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	// Clean environment
	os.Unsetenv("OTEL_EXPORTER_OTLP_PROTOCOL")
	os.Unsetenv("OTEL_EXPORTER_OTLP_ENDPOINT")
	os.Unsetenv("DEBUG")

	cfg := Load()

	if cfg.Protocol != "http" {
		t.Errorf("expected default protocol 'http', got %s", cfg.Protocol)
	}
	if cfg.Debug != false {
		t.Errorf("expected debug false by default, got %v", cfg.Debug)
	}
}

func TestGetEndpoint(t *testing.T) {
	tests := []struct {
		protocol string
		endpoint string
		want     string
	}{
		{"http", "", "localhost:4318"},
		{"grpc", "", "localhost:4317"},
		{"stdout", "", "stdout"},
		{"http", "custom:1234", "custom:1234"},
	}

	for _, tt := range tests {
		cfg := &Config{
			Protocol: tt.protocol,
			Endpoint: tt.endpoint,
		}
		got := cfg.GetEndpoint()
		if got != tt.want {
			t.Errorf("GetEndpoint() with protocol=%s, endpoint=%s = %s, want %s",
				tt.protocol, tt.endpoint, got, tt.want)
		}
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		key      string
		fallback string
		envValue string
		want     string
	}{
		{"TEST_KEY", "default", "", "default"},
		{"TEST_KEY", "default", "custom", "custom"},
	}

	for _, tt := range tests {
		if tt.envValue != "" {
			_ = os.Setenv(tt.key, tt.envValue)
			defer func() { _ = os.Unsetenv(tt.key) }()
		}
		if got := getEnv(tt.key, tt.fallback); got != tt.want {
			t.Errorf("getEnv(%q, %q) = %q, want %q", tt.key, tt.fallback, got, tt.want)
		}
	}
}
