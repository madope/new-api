package source

import (
	"time"

	"github.com/QuantumNous/new-api/programs/logsagg/internal/aggregate"
	"gorm.io/gorm"
)

const selectColumns = "SELECT id, created_at, user_id, username, model_name, channel_id, channel_name, type, token_id, token_name, \"group\" AS group_name, is_stream, quota, prompt_tokens, completion_tokens, use_time FROM logs"

func BuildIncrementalQuery(lastID int64, limit int) (string, []any) {
	return selectColumns + " WHERE id > ? ORDER BY id ASC LIMIT ?", []any{lastID, limit}
}

func BuildWindowQuery(start, end time.Time) (string, []any) {
	return selectColumns + " WHERE created_at >= ? AND created_at < ? ORDER BY id ASC", []any{start.UTC().Unix(), end.UTC().Unix()}
}

type Reader struct {
	DB *gorm.DB
}

func (r Reader) ReadIncremental(lastID int64, limit int) ([]aggregate.RawLog, error) {
	query, args := BuildIncrementalQuery(lastID, limit)
	var rows []aggregate.RawLog
	err := r.DB.Raw(query, args...).Scan(&rows).Error
	return rows, err
}

func (r Reader) ReadWindow(start, end time.Time) ([]aggregate.RawLog, error) {
	query, args := BuildWindowQuery(start, end)
	var rows []aggregate.RawLog
	err := r.DB.Raw(query, args...).Scan(&rows).Error
	return rows, err
}
