package server

import "time"

type HttpConfig struct {
	Port                   int
	ShutdownTimeout        time.Duration
	RequestTimeout         time.Duration
	WhitelistedPaths       []string
	AuthBlacklistedPaths   []string
	ProxyServers           []string
	HealthCheckInterval    time.Duration
	MaxCapacity            int
	AcquireCapacityTimeout time.Duration
}

func NewDefaultHttpConfig() *HttpConfig {
	return &HttpConfig{
		Port:                   8080,
		ShutdownTimeout:        10 * time.Second,
		RequestTimeout:         10 * time.Second,
		WhitelistedPaths:       []string{"/dummy", "/register", "/health"},
		AuthBlacklistedPaths:   []string{"/register", "/health"},
		ProxyServers:           []string{"http://wiremock1:8080", "http://wiremock2:8080", "http://wiremock3:8080"},
		HealthCheckInterval:    5 * time.Second,
		MaxCapacity:            5,
		AcquireCapacityTimeout: 10 * time.Second,
	}
}
