package config

import "testing"

func TestAutoSubmitConfiguration(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
		valid bool
	}{{"", false, true}, {"false", false, true}, {"true", true, true}, {"yes", false, false}, {"1", false, false}} {
		t.Run(tc.value, func(t *testing.T) {
			t.Setenv("AUTODARTS_AUTO_SUBMIT", tc.value)
			cfg := Load()
			if (cfg.Validate() == nil) != tc.valid || cfg.AutoSubmit != tc.want {
				t.Fatalf("value %q: %+v", tc.value, cfg)
			}
		})
	}
}
