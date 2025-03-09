package server

import "time"

type HttpConfig struct {
	Port                 int
	ShutdownTimeout      time.Duration
	RequestTimeout       time.Duration
	WhitelistedPaths     []string
	AuthBlacklistedPaths []string
	ProxyServers         []string
	HealthCheckInterval  time.Duration
}
