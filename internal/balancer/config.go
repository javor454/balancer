package balancer

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Duration is a wrapper for time.Duration that implements json.Unmarshaler
type Duration struct {
	time.Duration
}

func (d *Duration) UnmarshalJSON(b []byte) error {
	var v interface{}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}

	switch value := v.(type) {
	case string:
		var err error
		d.Duration, err = time.ParseDuration(value)
		if err != nil {
			return err
		}
	case float64:
		d.Duration = time.Duration(value) * time.Second
	default:
		return fmt.Errorf("invalid duration type %T", v)
	}

	return nil
}

type Config struct {
	Strategy        StrategyType `json:"strategy"`
	Capacity        int          `json:"capacity"`
	Port            int          `json:"port"`
	ShutdownTimeout Duration     `json:"shutdown_timeout"`
	SessionTimeout  Duration     `json:"session_timeout"`
	JobDuration     Duration     `json:"job_duration"`     // How long jobs take to process
	CleanupInterval Duration     `json:"cleanup_interval"` // How often to run cleanup
}

func LoadConfig() (*Config, error) {
	// Default config
	config := &Config{
		Strategy:        SingleClient,
		Capacity:        10,
		Port:            8080,
		ShutdownTimeout: Duration{Duration: 30 * time.Second},
		SessionTimeout:  Duration{Duration: 1 * time.Minute},
		JobDuration:     Duration{Duration: 10 * time.Second},
		CleanupInterval: Duration{Duration: 10 * time.Second},
	}

	// Try to load from config.json if it exists
	if _, err := os.Stat("config.json"); err == nil {
		file, err := os.Open("config.json")
		if err != nil {
			return nil, err
		}
		defer file.Close()

		if err := json.NewDecoder(file).Decode(config); err != nil {
			return nil, err
		}
	}

	// Validate config
	if err := config.validate(); err != nil {
		return nil, err
	}

	return config, nil
}

func (c *Config) validate() error {
	if err := Validate(c.Strategy.String()); err != nil {
		return err
	}

	if c.Capacity <= 0 {
		return fmt.Errorf("capacity must be greater than 0")
	}

	return nil
}
