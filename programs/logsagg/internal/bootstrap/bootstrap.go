package bootstrap

import (
	"errors"
	"fmt"
	"log"

	"github.com/QuantumNous/new-api/programs/logsagg/internal/sink"
	"gorm.io/gorm"
)

func BuildTargetPlan() []string {
	return []string{
		`CREATE EXTENSION IF NOT EXISTS timescaledb`,
		`CREATE TABLE IF NOT EXISTS logs_1m (
			bucket_ts TIMESTAMPTZ NOT NULL,
			user_id BIGINT NOT NULL,
			username TEXT NOT NULL DEFAULT '',
			model_name TEXT NOT NULL DEFAULT '',
			channel_id BIGINT NOT NULL DEFAULT 0,
			channel_name TEXT NOT NULL DEFAULT '',
			type INTEGER NOT NULL,
			token_id BIGINT NOT NULL DEFAULT 0,
			token_name TEXT NOT NULL DEFAULT '',
			group_name TEXT NOT NULL DEFAULT '',
			is_stream BOOLEAN NOT NULL DEFAULT false,
			count BIGINT NOT NULL DEFAULT 0,
			sum_quota BIGINT NOT NULL DEFAULT 0,
			sum_prompt_tokens BIGINT NOT NULL DEFAULT 0,
			sum_completion_tokens BIGINT NOT NULL DEFAULT 0,
			sum_use_time BIGINT NOT NULL DEFAULT 0,
			avg_use_time BIGINT NOT NULL DEFAULT 0,
			max_use_time BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY (bucket_ts, user_id, model_name, channel_id, type, token_id, group_name, is_stream)
		)`,
		`SELECT create_hypertable('logs_1m', 'bucket_ts', if_not_exists => TRUE)`,
		`CREATE INDEX IF NOT EXISTS idx_logs_1m_bucket_ts ON logs_1m (bucket_ts DESC)`,
		`CREATE TABLE IF NOT EXISTS logs_agg_state (
			job_name TEXT PRIMARY KEY,
			last_id BIGINT NOT NULL DEFAULT 0,
			last_success_bucket TIMESTAMPTZ NOT NULL DEFAULT '1970-01-01 00:00:00+00',
			last_incremental_run_at TIMESTAMPTZ,
			last_backfill_run_at TIMESTAMPTZ,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
		`INSERT INTO logs_agg_state (job_name, last_id, last_success_bucket, updated_at)
		VALUES ('logsagg', 0, '1970-01-01 00:00:00+00', now())
		ON CONFLICT (job_name) DO NOTHING`,
		sink.BuildLogs5MViewSQL(),
		`SELECT add_continuous_aggregate_policy('logs_5m', start_offset => INTERVAL '1 hour', end_offset => INTERVAL '5 minutes', schedule_interval => INTERVAL '5 minutes')`,
	}
}

func BuildSourceContractPlan() []string {
	return []string{
		`DO $$
		DECLARE missing_count integer;
		DECLARE logs_exists integer;
		BEGIN
			SELECT COUNT(*) INTO logs_exists
			FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'logs';
			IF logs_exists = 0 THEN
				RAISE EXCEPTION 'source table logs does not exist';
			END IF;

			SELECT COUNT(*) INTO missing_count
			FROM (
				SELECT required.column_name
				FROM (VALUES
					('id'),
					('created_at'),
					('user_id'),
					('username'),
					('model_name'),
					('channel_id'),
					('channel_name'),
					('type'),
					('token_id'),
					('token_name'),
					('group'),
					('is_stream'),
					('quota'),
					('prompt_tokens'),
					('completion_tokens'),
					('use_time')
				) AS required(column_name)
				LEFT JOIN information_schema.columns columns
					ON columns.table_schema = 'public'
					AND columns.table_name = 'logs'
					AND columns.column_name = required.column_name
				WHERE columns.column_name IS NULL
			) missing;
			IF missing_count > 0 THEN
				RAISE EXCEPTION 'source table logs is missing required columns';
			END IF;
		END $$`,
	}
}

type Runner struct {
	SourceDB *gorm.DB
	TargetDB *gorm.DB
}

func (r Runner) Run() error {
	for _, statement := range BuildSourceContractPlan() {
		if err := r.SourceDB.Exec(statement).Error; err != nil {
			return fmt.Errorf("bootstrap source contract failed: %w", err)
		}
	}

	for _, statement := range BuildTargetPlan() {
		if err := r.TargetDB.Exec(statement).Error; err != nil {
			if statement == `SELECT add_continuous_aggregate_policy('logs_5m', start_offset => INTERVAL '1 hour', end_offset => INTERVAL '5 minutes', schedule_interval => INTERVAL '5 minutes')` &&
				isDuplicateContinuousAggregatePolicyError(err) {
				log.Printf("logsagg bootstrap: logs_5m continuous aggregate policy already exists, reusing existing policy")
				continue
			}
			return fmt.Errorf("bootstrap exec failed: %w", err)
		}
	}
	return nil
}

type sqlStateCarrier interface {
	SQLState() string
}

func isDuplicateContinuousAggregatePolicyError(err error) bool {
	var carrier sqlStateCarrier
	if !errors.As(err, &carrier) {
		return false
	}
	return carrier.SQLState() == "42710"
}
