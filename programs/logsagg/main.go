package main

import (
	"context"
	"log"
	"time"

	"github.com/QuantumNous/new-api/programs/logsagg/internal/bootstrap"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/config"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/runner"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/sink"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/source"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/state"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	sourceDB, err := gorm.Open(postgres.Open(cfg.SourceDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("open source postgres connection: %v", err)
	}

	targetDB, err := gorm.Open(postgres.Open(cfg.TargetDSN), &gorm.Config{})
	if err != nil {
		log.Fatalf("open target postgres connection: %v", err)
	}

	service := runner.Service{
		Config:     cfg,
		Bootstrap:  bootstrap.Runner{SourceDB: sourceDB, TargetDB: targetDB},
		StateStore: state.Store{DB: targetDB},
		Reader:     source.Reader{DB: sourceDB},
		Writer:     sink.Writer{DB: targetDB},
	}

	progress, err := service.Initialize()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf(
		"logsagg config: mode=%s source_configured=%t target_configured=%t target_is_source=%t incremental_interval=%s rewind_interval=%s rewind_window=%s batch_size=%d",
		cfg.Mode,
		cfg.SourceDSN != "",
		cfg.TargetDSN != "",
		cfg.SourceDSN == cfg.TargetDSN,
		cfg.IncrementalInterval,
		cfg.RewindInterval,
		cfg.RewindWindow,
		cfg.BatchSize,
	)

	switch cfg.Mode {
	case "run":
		runMode(service, progress)
	case "backfill":
		runBackfillMode(service, progress, cfg)
	default:
		log.Fatalf("unsupported mode: %s", cfg.Mode)
	}
}

func runMode(service runner.Service, progress state.Progress) {
	ctx := context.Background()
	failures := 0

	startedAt := time.Now().UTC()
	progress, report, err := service.RunRewind(ctx, progress, startedAt)
	if err != nil {
		log.Fatal(err)
	}
	logRun("rewind-startup", report, progress, startedAt, time.Since(startedAt), failures)

	incrementalTicker := time.NewTicker(service.Config.IncrementalInterval)
	rewindTicker := time.NewTicker(service.Config.RewindInterval)
	defer incrementalTicker.Stop()
	defer rewindTicker.Stop()

	for {
		select {
		case <-incrementalTicker.C:
			startedAt = time.Now().UTC()
			progress, report, err = service.RunIncremental(ctx, progress, startedAt)
			if err != nil {
				failures++
				log.Printf("logsagg incremental run failed: %v", err)
				continue
			}
			failures = 0
			logRun("incremental", report, progress, startedAt, time.Since(startedAt), failures)
		case <-rewindTicker.C:
			startedAt = time.Now().UTC()
			progress, report, err = service.RunRewind(ctx, progress, startedAt)
			if err != nil {
				failures++
				log.Printf("logsagg rewind run failed: %v", err)
				continue
			}
			failures = 0
			logRun("rewind", report, progress, startedAt, time.Since(startedAt), failures)
		}
	}
}

func runBackfillMode(service runner.Service, progress state.Progress, cfg config.Config) {
	ctx := context.Background()
	windows := runner.SplitRange(*cfg.BackfillStart, *cfg.BackfillEnd, cfg.RewindWindow)
	for _, window := range windows {
		var err error
		startedAt := time.Now().UTC()
		progress, report, err := service.RunBackfill(ctx, progress, window.Start, window.End, startedAt)
		if err != nil {
			log.Fatal(err)
		}
		logRun("backfill", report, progress, startedAt, time.Since(startedAt), 0)
	}
	log.Printf("logsagg backfill complete for %s to %s", cfg.BackfillStart.UTC().Format(time.RFC3339), cfg.BackfillEnd.UTC().Format(time.RFC3339))
}

func logRun(mode string, report runner.RunReport, progress state.Progress, startedAt time.Time, duration time.Duration, failures int) {
	log.Printf(
		"logsagg run=%s started_at=%s duration=%s source_rows=%d facts_written=%d max_row_id=%d window_start=%s window_end=%s last_id=%d last_success_bucket=%s failures=%d",
		mode,
		startedAt.Format(time.RFC3339),
		duration,
		report.RowsRead,
		report.FactsWritten,
		report.MaxRowID,
		formatTime(report.WindowStart),
		formatTime(report.WindowEnd),
		progress.LastID,
		formatTime(progress.LastSuccessBucket),
		failures,
	)
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
