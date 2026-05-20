package config

import (
	"strings"
	"testing"
	"time"
)

func TestLoadRejectsInvalidMode(t *testing.T) {
	t.Setenv("LOGSAGG_MODE", "invalid")
	t.Setenv("LOGSAGG_DSN", "postgres://user:pass@localhost:5432/db")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected invalid mode error")
	}
	if !strings.Contains(err.Error(), "invalid LOGSAGG_MODE") {
		t.Fatalf("expected invalid mode error, got %v", err)
	}
}

func TestLoadRejectsMissingDSN(t *testing.T) {
	t.Setenv("LOGSAGG_MODE", "run")
	t.Setenv("LOGSAGG_DSN", "")
	t.Setenv("LOGSAGG_SOURCE_DSN", "")
	t.Setenv("LOGSAGG_TARGET_DSN", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected missing dsn error")
	}
}

func TestLoadUsesDefaultsForRunMode(t *testing.T) {
	t.Setenv("LOGSAGG_DSN", "postgres://user:pass@localhost:5432/db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "run" {
		t.Fatalf("expected default mode run, got %s", cfg.Mode)
	}
	if cfg.IncrementalInterval != 15*time.Second {
		t.Fatalf("expected default incremental interval, got %s", cfg.IncrementalInterval)
	}
	if cfg.RewindInterval != 10*time.Minute {
		t.Fatalf("expected default rewind interval, got %s", cfg.RewindInterval)
	}
	if cfg.RewindWindow != 30*time.Minute {
		t.Fatalf("expected default rewind window, got %s", cfg.RewindWindow)
	}
	if cfg.BatchSize != 500 {
		t.Fatalf("expected default batch size 500, got %d", cfg.BatchSize)
	}
	if cfg.SourceDSN != "postgres://user:pass@localhost:5432/db" {
		t.Fatalf("expected source dsn from LOGSAGG_DSN, got %s", cfg.SourceDSN)
	}
	if cfg.TargetDSN != "postgres://user:pass@localhost:5432/db" {
		t.Fatalf("expected target dsn fallback to source, got %s", cfg.TargetDSN)
	}
}

func TestLoadAcceptsBackfillModeAndParsesWindow(t *testing.T) {
	t.Setenv("LOGSAGG_MODE", "backfill")
	t.Setenv("LOGSAGG_DSN", "postgres://user:pass@localhost:5432/db")
	t.Setenv("LOGSAGG_BACKFILL_START", "2026-05-14T00:00:00Z")
	t.Setenv("LOGSAGG_BACKFILL_END", "2026-05-14T01:00:00Z")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Mode != "backfill" {
		t.Fatalf("expected mode backfill, got %s", cfg.Mode)
	}
	if cfg.BackfillStart == nil || cfg.BackfillEnd == nil {
		t.Fatalf("expected parsed backfill window, got %+v", cfg)
	}
	if !cfg.BackfillStart.Equal(time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected backfill start: %s", cfg.BackfillStart)
	}
	if !cfg.BackfillEnd.Equal(time.Date(2026, 5, 14, 1, 0, 0, 0, time.UTC)) {
		t.Fatalf("unexpected backfill end: %s", cfg.BackfillEnd)
	}
}

func TestLoadUsesSeparateSourceAndTargetDSN(t *testing.T) {
	t.Setenv("LOGSAGG_MODE", "run")
	t.Setenv("LOGSAGG_DSN", "")
	t.Setenv("LOGSAGG_SOURCE_DSN", "postgres://reader:pass@source:5432/source_db")
	t.Setenv("LOGSAGG_TARGET_DSN", "postgres://writer:pass@target:5432/target_db")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.SourceDSN != "postgres://reader:pass@source:5432/source_db" {
		t.Fatalf("unexpected source dsn: %s", cfg.SourceDSN)
	}
	if cfg.TargetDSN != "postgres://writer:pass@target:5432/target_db" {
		t.Fatalf("unexpected target dsn: %s", cfg.TargetDSN)
	}
}

func TestLoadRequiresBackfillWindowForBackfillMode(t *testing.T) {
	t.Setenv("LOGSAGG_MODE", "backfill")
	t.Setenv("LOGSAGG_DSN", "postgres://user:pass@localhost:5432/db")
	t.Setenv("LOGSAGG_BACKFILL_START", "2026-05-14T00:00:00Z")
	t.Setenv("LOGSAGG_BACKFILL_END", "")

	_, err := Load()
	if err == nil {
		t.Fatalf("expected missing backfill end error")
	}
}
