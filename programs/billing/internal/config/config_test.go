package config

import "testing"

func TestLoadRejectsMissingDSN(t *testing.T) {
	t.Setenv("BILLING_DSN", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected missing dsn error")
	}
}

func TestLoadUsesPreferredDSNAndDefaults(t *testing.T) {
	t.Setenv("BILLING_DSN", "postgres://billing:pass@localhost:5432/billing")
	t.Setenv("BILLING_HOST", "")
	t.Setenv("BILLING_PORT", "")
	t.Setenv("BILLING_DEFAULT_PAGE_SIZE", "")
	t.Setenv("BILLING_MAX_PAGE_SIZE", "")
	t.Setenv("BILLING_READ_TIMEOUT_SECONDS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.DSN != "postgres://billing:pass@localhost:5432/billing" {
		t.Fatalf("unexpected dsn: %s", cfg.DSN)
	}
	if cfg.Host != "127.0.0.1" {
		t.Fatalf("unexpected host: %s", cfg.Host)
	}
	if cfg.Port != 3011 {
		t.Fatalf("unexpected port: %d", cfg.Port)
	}
	if cfg.DefaultPageSize != 20 {
		t.Fatalf("unexpected default page size: %d", cfg.DefaultPageSize)
	}
	if cfg.MaxPageSize != 100 {
		t.Fatalf("unexpected max page size: %d", cfg.MaxPageSize)
	}
	if cfg.ReadTimeoutSeconds != 15 {
		t.Fatalf("unexpected read timeout: %d", cfg.ReadTimeoutSeconds)
	}
}
