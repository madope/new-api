package bootstrap

import (
	"errors"
	"strings"
	"testing"
)

func TestBuildPlanIncludesRequiredObjects(t *testing.T) {
	sqls := BuildTargetPlan()
	joined := strings.Join(sqls, "\n")
	required := []string{
		"CREATE EXTENSION IF NOT EXISTS timescaledb",
		"CREATE TABLE IF NOT EXISTS logs_1m",
		"create_hypertable",
		"CREATE TABLE IF NOT EXISTS logs_agg_state",
		"CREATE MATERIALIZED VIEW IF NOT EXISTS logs_5m",
		"add_continuous_aggregate_policy('logs_5m'",
	}
	for _, item := range required {
		if !strings.Contains(joined, item) {
			t.Fatalf("expected bootstrap plan to contain %q", item)
		}
	}
}

func TestBuildSourceContractPlanChecksLogsTable(t *testing.T) {
	sqls := BuildSourceContractPlan()
	joined := strings.Join(sqls, "\n")
	required := []string{
		"information_schema.columns",
		"table_name = 'logs'",
		"source table logs does not exist",
		"source table logs is missing required columns",
	}
	for _, item := range required {
		if !strings.Contains(joined, item) {
			t.Fatalf("expected source contract plan to contain %q", item)
		}
	}
}

type sqlStateErr struct {
	state string
}

func (e sqlStateErr) Error() string {
	return "sqlstate " + e.state
}

func (e sqlStateErr) SQLState() string {
	return e.state
}

func TestIsDuplicateContinuousAggregatePolicyError(t *testing.T) {
	if !isDuplicateContinuousAggregatePolicyError(sqlStateErr{state: "42710"}) {
		t.Fatalf("expected duplicate policy sqlstate to be ignored")
	}
	if isDuplicateContinuousAggregatePolicyError(sqlStateErr{state: "23505"}) {
		t.Fatalf("expected unrelated sqlstate not to be ignored")
	}
	if isDuplicateContinuousAggregatePolicyError(errors.New("plain error")) {
		t.Fatalf("expected plain error not to be ignored")
	}
}
