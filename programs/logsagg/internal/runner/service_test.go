package runner

import (
	"context"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/programs/logsagg/internal/aggregate"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/config"
	"github.com/QuantumNous/new-api/programs/logsagg/internal/state"
)

type stubReader struct {
	windowRows []aggregate.RawLog
	start      time.Time
	end        time.Time
}

func (s stubReader) ReadIncremental(lastID int64, limit int) ([]aggregate.RawLog, error) {
	return s.windowRows, nil
}

func (s stubReader) ReadWindow(start, end time.Time) ([]aggregate.RawLog, error) {
	return s.windowRows, nil
}

type stubWriter struct {
	facts []aggregate.Fact
}

func (s *stubWriter) UpsertFacts(facts []aggregate.Fact) error {
	s.facts = append([]aggregate.Fact(nil), facts...)
	return nil
}

func TestRecoveryWindowExpandsForLongDowntime(t *testing.T) {
	now := time.Unix(1715666400, 0).UTC()
	progress := state.Progress{
		LastID:            200,
		LastSuccessBucket: time.Unix(1715659200, 0).UTC(),
	}
	start, end := RecoveryWindow(progress, now, 30*time.Minute)
	if !start.Equal(progress.LastSuccessBucket) {
		t.Fatalf("expected recovery start at last success bucket, got %s", start)
	}
	if !end.Equal(now) {
		t.Fatalf("expected recovery end at now, got %s", end)
	}
}

func TestRecoveryWindowUsesRecentWindowForFreshState(t *testing.T) {
	now := time.Unix(1715666400, 0).UTC()
	progress := state.Progress{}

	start, end := RecoveryWindow(progress, now, 30*time.Minute)
	if !start.Equal(now.Add(-30 * time.Minute)) {
		t.Fatalf("expected fresh state to use recent rewind window, got %s", start)
	}
	if !end.Equal(now) {
		t.Fatalf("expected recovery end at now, got %s", end)
	}
}

func TestRunRewindBuildsFactsAndAdvancesState(t *testing.T) {
	reader := stubReader{
		windowRows: []aggregate.RawLog{
			{ID: 100, CreatedAt: 1715659201, UserID: 1, Username: "alice", ModelName: "gpt-4o", ChannelID: 11, ChannelName: "main", Type: 2, TokenID: 9, TokenName: "key", GroupName: "default", IsStream: true, Quota: 100, PromptTokens: 10, CompletionTokens: 20, UseTime: 5},
			{ID: 101, CreatedAt: 1715659230, UserID: 1, Username: "alice", ModelName: "gpt-4o", ChannelID: 11, ChannelName: "main", Type: 2, TokenID: 9, TokenName: "key", GroupName: "default", IsStream: true, Quota: 50, PromptTokens: 5, CompletionTokens: 10, UseTime: 7},
		},
	}
	writer := &stubWriter{}
	service := Service{
		Config: config.Config{
			RewindWindow: 30 * time.Minute,
		},
		Reader: reader,
		Writer: writer,
	}

	now := time.Unix(1715659800, 0).UTC()
	progress, report, err := service.RunRewind(context.Background(), state.Progress{}, now)
	if err != nil {
		t.Fatalf("RunRewind returned error: %v", err)
	}
	if progress.LastID != 0 {
		t.Fatalf("expected rewind not to advance last_id, got %d", progress.LastID)
	}
	if !progress.LastSuccessBucket.Equal(time.Unix(1715659200, 0).UTC()) {
		t.Fatalf("expected rewind to advance last success bucket, got %s", progress.LastSuccessBucket)
	}
	if len(writer.facts) != 1 {
		t.Fatalf("expected a single aggregated fact, got %d", len(writer.facts))
	}
	if writer.facts[0].SumQuota != 150 {
		t.Fatalf("expected sum quota 150, got %+v", writer.facts[0])
	}
	if report.RowsRead != 2 || report.FactsWritten != 1 || report.MaxRowID != 101 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestRunBackfillUsesExplicitWindow(t *testing.T) {
	start := time.Unix(1715659200, 0).UTC()
	end := start.Add(time.Hour)
	reader := stubReader{
		windowRows: []aggregate.RawLog{
			{ID: 55, CreatedAt: start.Unix() + 1, UserID: 1, Username: "alice", ModelName: "gpt-4o", ChannelID: 11, ChannelName: "main", Type: 2, TokenID: 9, TokenName: "key", GroupName: "default", IsStream: true, Quota: 10, PromptTokens: 1, CompletionTokens: 2, UseTime: 3},
		},
	}
	writer := &stubWriter{}
	service := Service{
		Config: config.Config{},
		Reader: reader,
		Writer: writer,
	}

	progress, report, err := service.RunBackfill(context.Background(), state.Progress{}, start, end, end)
	if err != nil {
		t.Fatalf("RunBackfill returned error: %v", err)
	}
	if progress.LastID != 0 {
		t.Fatalf("expected backfill not to advance last_id, got %d", progress.LastID)
	}
	if !progress.LastBackfillRunAt.IsZero() {
		t.Fatalf("expected explicit backfill not to mutate live progress timestamps, got %s", progress.LastBackfillRunAt)
	}
	if len(writer.facts) != 1 {
		t.Fatalf("expected backfill to write one fact, got %d", len(writer.facts))
	}
	if report.RowsRead != 1 || report.FactsWritten != 1 || report.MaxRowID != 55 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestRunIncrementalReportsBatchStats(t *testing.T) {
	reader := stubReader{
		windowRows: []aggregate.RawLog{
			{ID: 201, CreatedAt: 1715659201, UserID: 1, Username: "alice", ModelName: "gpt-4o", ChannelID: 11, ChannelName: "main", Type: 2, TokenID: 9, TokenName: "key", GroupName: "default", IsStream: true, Quota: 10, PromptTokens: 1, CompletionTokens: 2, UseTime: 3},
		},
	}
	writer := &stubWriter{}
	service := Service{
		Config: config.Config{BatchSize: 100},
		Reader: reader,
		Writer: writer,
	}

	progress, report, err := service.RunIncremental(context.Background(), state.Progress{}, time.Unix(1715659260, 0).UTC())
	if err != nil {
		t.Fatalf("RunIncremental returned error: %v", err)
	}
	if progress.LastID != 201 {
		t.Fatalf("expected last_id 201, got %d", progress.LastID)
	}
	if report.RowsRead != 1 || report.FactsWritten != 1 || report.MaxRowID != 201 {
		t.Fatalf("unexpected report: %+v", report)
	}
}

func TestSplitRangeBreaksBackfillIntoChunks(t *testing.T) {
	start := time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC)
	end := start.Add(65 * time.Minute)

	windows := SplitRange(start, end, 30*time.Minute)
	if len(windows) != 3 {
		t.Fatalf("expected 3 windows, got %d", len(windows))
	}
	if !windows[0].Start.Equal(start) || !windows[0].End.Equal(start.Add(30*time.Minute)) {
		t.Fatalf("unexpected first window: %+v", windows[0])
	}
	if !windows[2].Start.Equal(start.Add(60*time.Minute)) || !windows[2].End.Equal(end) {
		t.Fatalf("unexpected last window: %+v", windows[2])
	}
}

func TestRecoveryWindowsSplitLongDowntime(t *testing.T) {
	progress := state.Progress{
		LastSuccessBucket: time.Date(2026, 5, 14, 0, 0, 0, 0, time.UTC),
	}
	now := time.Date(2026, 5, 14, 2, 0, 0, 0, time.UTC)

	windows := RecoveryWindows(progress, now, 30*time.Minute)
	if len(windows) != 4 {
		t.Fatalf("expected 4 recovery windows, got %d", len(windows))
	}
	if !windows[0].Start.Equal(progress.LastSuccessBucket) || !windows[0].End.Equal(progress.LastSuccessBucket.Add(30*time.Minute)) {
		t.Fatalf("unexpected first recovery window: %+v", windows[0])
	}
	if !windows[3].Start.Equal(time.Date(2026, 5, 14, 1, 30, 0, 0, time.UTC)) || !windows[3].End.Equal(now) {
		t.Fatalf("unexpected last recovery window: %+v", windows[3])
	}
}
