package scheduler

import "testing"

func TestConfig_Validate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"spread valid", Config{Strategy: StrategySpread}, false},
		{"pack valid", Config{Strategy: StrategyPack}, false},
		{"empty invalid", Config{Strategy: ""}, true},
		{"garbage invalid", Config{Strategy: "yolo"}, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestDefaultConfig_IsSpread(t *testing.T) {
	if DefaultConfig().Strategy != StrategySpread {
		t.Fatal("expected default strategy to be spread, preserving pre-Step-7 behavior")
	}
}
