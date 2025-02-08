package balancer

import (
	"fmt"
	"strings"
)

type Strategy string

const (
	RoundRobin      Strategy = "round-robin"
	SingleClient    Strategy = "single-client"
	BatchProcessing Strategy = "batch"
	WeightedFair    Strategy = "weighted"
)

func (s *Strategy) String() string {
	return string(*s)
}

func Validate(value string) error {
	strategy := Strategy(value)
	switch strategy {
	case RoundRobin, SingleClient, BatchProcessing, WeightedFair:
		return nil
	default:
		// flag package will print the error message to os.Stderr, display the command-line usage information, and then call os.Exit(2)
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
