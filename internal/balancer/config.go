package balancer

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Strategy Strategy `json:"strategy"`
	Capacity int      `json:"capacity"`
}

func LoadConfig() (*Config, error) {
	// Default config
	config := &Config{
		Strategy: SingleClient,
		Capacity: 10,
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
