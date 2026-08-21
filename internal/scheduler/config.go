package scheduler

import "fmt"

// Strategy selects which node-resources score plugin is active. Exactly
// one is registered at a time - registering both least- and most-allocated
// simultaneously would cancle each other out in the weighted sum
type Strategy string

const (
	StrategySpread Strategy = "spread" // NodeResourcesLeastAllocated
	StrategyPack   Strategy = "pack"   // NodeResourcesMostAllocated
)

type Config struct {
	Strategy Strategy
}

func DefaultConfig() Config {
	return Config{Strategy: StrategySpread}
}

func (c Config) Validate() error {
	switch c.Strategy {
	case StrategySpread, StrategyPack:
		return nil
	default:
		return fmt.Errorf("invalid strategy %q: must be %q or %q", c.Strategy, StrategySpread, StrategyPack)
	}
}
