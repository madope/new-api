package sink

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/programs/logsagg/internal/aggregate"
	"gorm.io/gorm"
)

func BuildUpsertSQL(rowCount int) string {
	valueRows := make([]string, 0, rowCount)
	for i := 0; i < rowCount; i++ {
		valueRows = append(valueRows, "(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)")
	}
	return "INSERT INTO logs_1m (bucket_ts, user_id, username, model_name, channel_id, channel_name, type, token_id, token_name, group_name, is_stream, count, sum_quota, sum_prompt_tokens, sum_completion_tokens, sum_use_time, avg_use_time, max_use_time) VALUES " +
		strings.Join(valueRows, ",") +
		" ON CONFLICT (bucket_ts, user_id, model_name, channel_id, type, token_id, group_name, is_stream) DO UPDATE SET username = EXCLUDED.username, channel_name = EXCLUDED.channel_name, token_name = EXCLUDED.token_name, count = EXCLUDED.count, sum_quota = EXCLUDED.sum_quota, sum_prompt_tokens = EXCLUDED.sum_prompt_tokens, sum_completion_tokens = EXCLUDED.sum_completion_tokens, sum_use_time = EXCLUDED.sum_use_time, avg_use_time = EXCLUDED.avg_use_time, max_use_time = EXCLUDED.max_use_time"
}

func BuildLogs5MViewSQL() string {
	return `CREATE MATERIALIZED VIEW IF NOT EXISTS logs_5m
WITH (timescaledb.continuous) AS
SELECT
	time_bucket('5 minutes', bucket_ts) AS bucket_ts,
	user_id,
	(ARRAY_AGG(NULLIF(username, '') ORDER BY bucket_ts DESC) FILTER (WHERE NULLIF(username, '') IS NOT NULL))[1] AS username,
	model_name,
	channel_id,
	(ARRAY_AGG(NULLIF(channel_name, '') ORDER BY bucket_ts DESC) FILTER (WHERE NULLIF(channel_name, '') IS NOT NULL))[1] AS channel_name,
	type,
	token_id,
	(ARRAY_AGG(NULLIF(token_name, '') ORDER BY bucket_ts DESC) FILTER (WHERE NULLIF(token_name, '') IS NOT NULL))[1] AS token_name,
	group_name,
	is_stream,
	SUM(count) AS count,
	SUM(sum_quota) AS sum_quota,
	SUM(sum_prompt_tokens) AS sum_prompt_tokens,
	SUM(sum_completion_tokens) AS sum_completion_tokens,
	SUM(sum_use_time) AS sum_use_time,
	SUM(sum_use_time) / NULLIF(SUM(count), 0) AS avg_use_time,
	MAX(max_use_time) AS max_use_time
FROM logs_1m
GROUP BY 1, user_id, model_name, channel_id, type, token_id, group_name, is_stream`
}

type Writer struct {
	DB *gorm.DB
}

func (w Writer) UpsertFacts(facts []aggregate.Fact) error {
	if len(facts) == 0 {
		return nil
	}

	args := make([]any, 0, len(facts)*18)
	for _, fact := range facts {
		args = append(args,
			fact.BucketTS.UTC(),
			fact.UserID,
			fact.Username,
			fact.ModelName,
			fact.ChannelID,
			fact.ChannelName,
			fact.Type,
			fact.TokenID,
			fact.TokenName,
			fact.GroupName,
			fact.IsStream,
			fact.Count,
			fact.SumQuota,
			fact.SumPromptTokens,
			fact.SumCompletionTokens,
			fact.SumUseTime,
			fact.AvgUseTime,
			fact.MaxUseTime,
		)
	}

	if err := w.DB.Exec(BuildUpsertSQL(len(facts)), args...).Error; err != nil {
		return fmt.Errorf("upsert logs_1m facts: %w", err)
	}
	return nil
}
