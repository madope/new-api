package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/programs/logsagg/internal/aggregate"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/config"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/state"
)

type Reader interface {
	ReadIncremental(lastID int64, limit int) ([]aggregate.RawLog, error)
	ReadWindow(start, end time.Time) ([]aggregate.RawLog, error)
}

type Writer interface {
	UpsertFacts(facts []aggregate.Fact) error
}

type Bootstrapper interface {
	Run() error
}

type StateStore interface {
	Load(jobName string) (state.Progress, error)
	Save(progress state.Progress) error
}

type Service struct {
	Config     config.Config
	Bootstrap  Bootstrapper
	StateStore StateStore
	Reader     Reader
	Writer     Writer
}

type Window struct {
	Start time.Time
	End   time.Time
}

type RunReport struct {
	RowsRead     int
	FactsWritten int
	MaxRowID     int64
	WindowStart  time.Time
	WindowEnd    time.Time
}

func RecoveryWindow(progress state.Progress, now time.Time, rewindWindow time.Duration) (time.Time, time.Time) {
	shortStart := now.UTC().Add(-rewindWindow)
	if progress.LastSuccessBucket.IsZero() || (progress.LastID == 0 && progress.LastSuccessBucket.Unix() == 0) {
		return shortStart, now.UTC()
	}
	if progress.LastSuccessBucket.Before(shortStart) {
		return progress.LastSuccessBucket.UTC(), now.UTC()
	}
	return shortStart, now.UTC()
}

func RecoveryWindows(progress state.Progress, now time.Time, rewindWindow time.Duration) []Window {
	start, end := RecoveryWindow(progress, now, rewindWindow)
	if progress.LastSuccessBucket.IsZero() || !progress.LastSuccessBucket.Before(now.UTC().Add(-rewindWindow)) {
		return []Window{{Start: start, End: end}}
	}
	return SplitRange(start, end, rewindWindow)
}

func SplitRange(start, end time.Time, step time.Duration) []Window {
	if !end.After(start) {
		return nil
	}
	if step <= 0 {
		step = end.Sub(start)
	}

	windows := make([]Window, 0, int(end.Sub(start)/step)+1)
	cursor := start.UTC()
	end = end.UTC()
	for cursor.Before(end) {
		next := cursor.Add(step)
		if next.After(end) {
			next = end
		}
		windows = append(windows, Window{Start: cursor, End: next})
		cursor = next
	}
	return windows
}

func (s Service) Initialize() (state.Progress, error) {
	if s.Bootstrap != nil {
		if err := s.Bootstrap.Run(); err != nil {
			return state.Progress{}, err
		}
	}
	if s.StateStore == nil {
		return state.Progress{JobName: state.DefaultJobName}, nil
	}
	return s.StateStore.Load(state.DefaultJobName)
}

func (s Service) RunRewind(ctx context.Context, progress state.Progress, now time.Time) (state.Progress, RunReport, error) {
	_ = ctx
	report := RunReport{}
	for _, window := range RecoveryWindows(progress, now, s.Config.RewindWindow) {
		rows, err := s.Reader.ReadWindow(window.Start, window.End)
		if err != nil {
			return progress, report, fmt.Errorf("read rewind window: %w", err)
		}
		facts := aggregate.BuildBuckets(rows)
		if err := s.Writer.UpsertFacts(facts); err != nil {
			return progress, report, fmt.Errorf("write rewind facts: %w", err)
		}
		report.RowsRead += len(rows)
		report.FactsWritten += len(facts)
		report.MaxRowID = maxReportID(report.MaxRowID, rows)
		if report.WindowStart.IsZero() {
			report.WindowStart = window.Start
		}
		report.WindowEnd = window.End
		progress.JobName = state.DefaultJobName
		progress.LastSuccessBucket = processedBucket(window, facts)
		progress.LastBackfillRunAt = now.UTC()
		progress.UpdatedAt = now.UTC()
		if s.StateStore != nil {
			if err := s.StateStore.Save(progress); err != nil {
				return progress, report, fmt.Errorf("save rewind progress: %w", err)
			}
		}
	}
	return progress, report, nil
}

func (s Service) RunIncremental(ctx context.Context, progress state.Progress, now time.Time) (state.Progress, RunReport, error) {
	_ = ctx
	report := RunReport{}
	rows, err := s.Reader.ReadIncremental(progress.LastID, s.Config.BatchSize)
	if err != nil {
		return progress, report, fmt.Errorf("read incremental rows: %w", err)
	}
	facts := aggregate.BuildBuckets(rows)
	if err := s.Writer.UpsertFacts(facts); err != nil {
		return progress, report, fmt.Errorf("write incremental facts: %w", err)
	}
	report.RowsRead = len(rows)
	report.FactsWritten = len(facts)
	report.MaxRowID = maxReportID(0, rows)
	if len(rows) == 0 {
		return progress, report, nil
	}

	lastBucket := facts[len(facts)-1].BucketTS
	lastID := rows[len(rows)-1].ID
	if err := progress.Advance(lastID, lastBucket, now.UTC(), false); err != nil {
		return progress, report, err
	}
	if s.StateStore != nil {
		if err := s.StateStore.Save(progress); err != nil {
			return progress, report, fmt.Errorf("save incremental progress: %w", err)
		}
	}
	return progress, report, nil
}

func (s Service) RunBackfill(ctx context.Context, progress state.Progress, start, end, now time.Time) (state.Progress, RunReport, error) {
	_ = ctx
	_ = now
	report := RunReport{
		WindowStart: start.UTC(),
		WindowEnd:   end.UTC(),
	}
	rows, err := s.Reader.ReadWindow(start.UTC(), end.UTC())
	if err != nil {
		return progress, report, fmt.Errorf("read backfill window: %w", err)
	}
	facts := aggregate.BuildBuckets(rows)
	if err := s.Writer.UpsertFacts(facts); err != nil {
		return progress, report, fmt.Errorf("write backfill facts: %w", err)
	}
	report.RowsRead = len(rows)
	report.FactsWritten = len(facts)
	report.MaxRowID = maxReportID(0, rows)
	return progress, report, nil
}

func processedBucket(window Window, facts []aggregate.Fact) time.Time {
	if len(facts) > 0 {
		return facts[len(facts)-1].BucketTS.UTC()
	}
	end := window.End.UTC().Add(-time.Minute)
	if end.Before(window.Start.UTC()) {
		return window.Start.UTC()
	}
	return end.Truncate(time.Minute)
}

func maxReportID(current int64, rows []aggregate.RawLog) int64 {
	for _, row := range rows {
		if row.ID > current {
			current = row.ID
		}
	}
	return current
}
