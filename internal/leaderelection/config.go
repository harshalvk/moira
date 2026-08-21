package leaderelection

import (
	"fmt"
	"time"
)

type Config struct {
	Enabled       bool
	LeaseName     string
	Namespace     string
	Identity      string
	LeaseDuration time.Duration
	RenewDeadline time.Duration
	RetryPeriod   time.Duration
}

// DefaultConfig uses the same timing kube-scheduler ships with by default -
// no reason to device without a measured reason to
func DefaultConfig(identity string) Config {
	return Config{
		Enabled:       true,
		LeaseName:     "moira-scheduler",
		Namespace:     "default",
		Identity:      identity,
		LeaseDuration: 15 * time.Second,
		RenewDeadline: 10 * time.Second,
		RetryPeriod:   2 * time.Second,
	}
}

func (c Config) Validate() error {
	if c.LeaseName == "" {
		return fmt.Errorf("lease name must not be empty")
	}
	if c.Namespace == "" {
		return fmt.Errorf("namespace must not be empty")
	}
	if c.Identity == "" {
		return fmt.Errorf("identity must not be empty")
	}
	if c.RenewDeadline >= c.LeaseDuration {
		return fmt.Errorf("renew deadline (%s) must be less than lease duration (%s)", c.RenewDeadline, c.LeaseDuration)
	}
	if c.RetryPeriod*3 > c.RenewDeadline {
		return fmt.Errorf("retry period (%s) too large relative to renew deadline (%s)", c.RetryPeriod, c.RenewDeadline)
	}
	return nil
}
