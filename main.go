package main

import (
	"log"

	"github.com/javor454/balancer/internal/balancer"
)

func main() {
	config, err := balancer.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %s", err)
	}

	balancer, err := balancer.NewBalancer(config)
	if err != nil {
		log.Fatalf("Failed to create balancer: %s", err)
	}

	if err := balancer.Balance(); err != nil {
		log.Fatalf("Failed to balance: %s", err)
	}
}
