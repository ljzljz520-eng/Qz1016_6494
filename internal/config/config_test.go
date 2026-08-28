package config

import "testing"

func TestConfigDefaults(t *testing.T) {
	cfg := Load()
	if cfg.Address == "" || cfg.DatabasePath == "" || cfg.MaxBodyBytes <= 0 {
		t.Fatalf("%+v", cfg)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}
