package server

import "time"

type HttpConfig struct {
	Port                int
	ShutdownTimeout     time.Duration
	RequestTimeout      time.Duration
	Whitelist           []string
	ProxyServers        []string
	HealthCheckInterval time.Duration
}
