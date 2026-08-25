package config

import (
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	// Ensure a clean environment for the keys we assert on
	for _, key := range []string{"PORT", "ALLOWED_ORIGINS", "MAX_FILE_SIZE", "ALLOWED_EMAIL_DOMAIN"} {
		t.Setenv(key, "")
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Server.Port)
	}
	if len(cfg.Server.AllowedOrigins) == 0 {
		t.Error("expected default AllowedOrigins to be non-empty")
	}
	if cfg.Upload.MaxSize != 10*1024*1024 {
		t.Errorf("MaxSize = %d, want %d", cfg.Upload.MaxSize, 10*1024*1024)
	}
	for _, ext := range cfg.Upload.AllowedTypes {
		if ext == ".webp" {
			t.Error("webp should not be in the allowed upload types (imaging cannot encode it)")
		}
	}
}

func TestLoadAllowedOrigins(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://lostfound.college.edu, https://www.lostfound.college.edu ,")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	want := []string{"https://lostfound.college.edu", "https://www.lostfound.college.edu"}
	if len(cfg.Server.AllowedOrigins) != len(want) {
		t.Fatalf("AllowedOrigins = %v, want %v", cfg.Server.AllowedOrigins, want)
	}
	for i := range want {
		if cfg.Server.AllowedOrigins[i] != want[i] {
			t.Errorf("AllowedOrigins[%d] = %q, want %q", i, cfg.Server.AllowedOrigins[i], want[i])
		}
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("MAX_FILE_SIZE", "1048576")
	t.Setenv("ALLOWED_EMAIL_DOMAIN", "college.edu")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("Port = %q, want 9090", cfg.Server.Port)
	}
	if cfg.Upload.MaxSize != 1048576 {
		t.Errorf("MaxSize = %d, want 1048576", cfg.Upload.MaxSize)
	}
	if cfg.Auth.AllowedEmailDomain != "college.edu" {
		t.Errorf("AllowedEmailDomain = %q, want college.edu", cfg.Auth.AllowedEmailDomain)
	}
}
