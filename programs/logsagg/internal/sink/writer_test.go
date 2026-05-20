package sink

import (
	"strings"
	"testing"
)

func TestBuildUpsertSQLContainsAllUniqueKeyColumns(t *testing.T) {
	sql := BuildUpsertSQL(2)
	required := []string{
		"bucket_ts", "user_id", "model_name", "channel_id", "type", "token_id", "group_name", "is_stream",
		"ON CONFLICT", "DO UPDATE SET",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Fatalf("expected sql to contain %q, got %s", item, sql)
		}
	}
}

func TestBuildLogs5MViewSQLUsesWeightedAverage(t *testing.T) {
	sql := BuildLogs5MViewSQL()
	if !strings.Contains(sql, "SUM(sum_use_time) / NULLIF(SUM(count), 0)") {
		t.Fatalf("expected weighted average, got %s", sql)
	}
}

func TestBuildLogs5MViewSQLUsesLatestNonEmptyDisplayValues(t *testing.T) {
	sql := BuildLogs5MViewSQL()
	required := []string{
		"ARRAY_AGG(NULLIF(username, '') ORDER BY bucket_ts DESC)",
		"ARRAY_AGG(NULLIF(channel_name, '') ORDER BY bucket_ts DESC)",
		"ARRAY_AGG(NULLIF(token_name, '') ORDER BY bucket_ts DESC)",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Fatalf("expected latest non-empty expression %q, got %s", item, sql)
		}
	}
}
