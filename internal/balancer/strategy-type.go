package balancer

import (
	"fmt"
	"strings"
)

type StrategyType string

const (
	RoundRobin      StrategyType = "round-robin"
	SingleClient    StrategyType = "single-client"
	BatchProcessing StrategyType = "batch"
	WeightedFair    StrategyType = "weighted"
)

func (s *StrategyType) String() string {
	return string(*s)
}

func Validate(value string) error {
	strategy := StrategyType(value)
	switch strategy {
	case RoundRobin, SingleClient, BatchProcessing, WeightedFair:
		return nil
	default:
		// flag package will print the error message to os.Stderr, display the command-line usage information, and then call os.Exit
		return fmt.Errorf("invalid strategy %q, must be one of: %s", value, strings.Join(Strategies(), ", "))
	}
}

func Strategies() []string {
	return []string{
		string(RoundRobin),
		string(SingleClient),
		string(BatchProcessing),
		string(WeightedFair),
	}
}
