package leaderelection

import (
	"testing"
	"time"
)

func TestConfig_Validate(t *testing.T) {
	base := DefaultConfig("replica-1")

	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"default is valid", func(c *Config) {}, false},
		{"empty lease name", func(c *Config) { c.LeaseName = "" }, true},
		{"empty namespace", func(c *Config) { c.Namespace = "" }, true},
		{"empty identity", func(c *Config) { c.Identity = "" }, true},
		{"renew >= lease duration", func(c *Config) { c.RenewDeadline = c.LeaseDuration }, true},
		{"retry period too large", func(c *Config) { c.RetryPeriod = c.RenewDeadline }, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultConfig_UsesGivenIdentity(t *testing.T) {
	cfg := DefaultConfig("my-pod")
	if cfg.Identity != "my-pod" {
		t.Fatalf("expected identity 'my-pod', got %q", cfg.Identity)
	}
	if !cfg.Enabled {
		t.Fatal("expected leader election enabled by default")
	}
}

var _ = time.Second // keep time import if unused in future edits
