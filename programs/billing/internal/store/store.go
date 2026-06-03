package store

import (
	"database/sql"
	"fmt"
	"strconv"

	"github.com/QuantumNous/new-api/programs/billing/internal/query"
	"gorm.io/gorm"
)

type Store struct {
	DB *gorm.DB
}

func (s Store) Ping() error {
	sqlDB, err := s.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}

func (s Store) ValidateSchema() error {
	const sqlText = `SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = 'public' AND table_name = 'logs_1m'
AND column_name IN ('bucket_ts','user_id','username','model_name','channel_id','channel_name','type','token_id','token_name','group_name','is_stream','count','sum_quota','sum_prompt_tokens','sum_completion_tokens','sum_use_time','max_use_time')`
	var count int64
	if err := s.DB.Raw(sqlText).Scan(&count).Error; err != nil {
		return fmt.Errorf("validate logs_1m schema: %w", err)
	}
	if count != 17 {
		return fmt.Errorf("validate logs_1m schema: required columns missing")
	}
	return nil
}

func (s Store) QuerySummary(req query.Request) (query.SummaryRow, error) {
	sqlText, args := query.BuildSummarySQL(req)
	var row query.SummaryRow
	if err := s.DB.Raw(sqlText, args...).Scan(&row).Error; err != nil {
		return query.SummaryRow{}, err
	}
	return row, nil
}

func (s Store) QueryItems(req query.Request) ([]map[string]any, error) {
	sqlText, args := query.BuildItemsSQL(req)
	rows, err := s.DB.Raw(sqlText, args...).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRows(rows)
}

func (s Store) CountItems(req query.Request) (int, error) {
	sqlText, args := query.BuildCountSQL(req)
	var count int64
	if err := s.DB.Raw(sqlText, args...).Scan(&count).Error; err != nil {
		return 0, err
	}
	return int(count), nil
}

func scanRows(rows *sql.Rows) ([]map[string]any, error) {
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}
	result := make([]map[string]any, 0)
	for rows.Next() {
		values := make([]any, len(columns))
		scanArgs := make([]any, len(columns))
		for i := range values {
			scanArgs[i] = &values[i]
		}
		if err := rows.Scan(scanArgs...); err != nil {
			return nil, err
		}
		row := make(map[string]any, len(columns))
		for i, column := range columns {
			row[column] = normalizeValue(values[i])
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func normalizeValue(value any) any {
	switch v := value.(type) {
	case []byte:
		text := string(v)
		if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
			return parsed
		}
		if parsed, err := strconv.ParseFloat(text, 64); err == nil {
			return parsed
		}
		if text == "t" {
			return true
		}
		if text == "f" {
			return false
		}
		return text
	default:
		return value
	}
}
