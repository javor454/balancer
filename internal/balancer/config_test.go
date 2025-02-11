package balancer

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Test default config when no file exists
	if err := os.Remove("config.json"); err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to remove config file: %v", err)
	}

	config, err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load default config: %v", err)
	}

	if config.Strategy != SingleClient {
		t.Errorf("Expected strategy %v, got %v", SingleClient, config.Strategy)
	}
	if config.Capacity != 10 {
		t.Errorf("Expected capacity 10, got %d", config.Capacity)
	}
	if config.Port != 8080 {
		t.Errorf("Expected port 8080, got %d", config.Port)
	}
	if config.ShutdownTimeout.Duration != 30*time.Second {
		t.Errorf("Expected shutdown timeout 30s, got %v", config.ShutdownTimeout.Duration)
	}
	if config.SessionTimeout.Duration != time.Minute {
		t.Errorf("Expected session timeout 1m, got %v", config.SessionTimeout.Duration)
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid config",
			config: Config{
				Strategy: SingleClient,
				Capacity: 10,
			},
			expectError: false,
		},
		{
			name: "invalid strategy",
			config: Config{
				Strategy: "invalid",
				Capacity: 10,
			},
			expectError: true,
		},
		{
			name: "invalid capacity",
			config: Config{
				Strategy: SingleClient,
				Capacity: 0,
			},
			expectError: true,
		},
		{
			name: "negative capacity",
			config: Config{
				Strategy: SingleClient,
				Capacity: -1,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}
