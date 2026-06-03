package query

import (
	"fmt"
	"strings"
	"time"
)

type Request struct {
	StartTimestamp int64
	EndTimestamp   int64
	UserID         int64
	Username       string
	TokenID        int64
	TokenName      string
	ChannelID      int64
	ChannelName    string
	ModelName      string
	GroupName      string
	IsStream       *bool
	Type           *int
	GroupBy        string
	Page           int
	PageSize       int
	OrderBy        string
	Order          string
}

type Grouping struct {
	Name         string
	SelectClause string
	GroupClause  string
}

type SummaryRow struct {
	Requests         int64   `json:"requests"`
	Quota            int64   `json:"quota"`
	Amount           float64 `json:"amount"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	AvgUseTimeMS     int64   `json:"avg_use_time_ms"`
	MaxUseTimeMS     int64   `json:"max_use_time_ms"`
}

var allowedGroupings = map[string]Grouping{
	"user": {
		Name:         "user",
		SelectClause: "user_id, username",
		GroupClause:  "user_id, username",
	},
	"token": {
		Name:         "token",
		SelectClause: "token_id, token_name",
		GroupClause:  "token_id, token_name",
	},
	"channel": {
		Name:         "channel",
		SelectClause: "channel_id, channel_name",
		GroupClause:  "channel_id, channel_name",
	},
	"model": {
		Name:         "model",
		SelectClause: "model_name",
		GroupClause:  "model_name",
	},
	"group": {
		Name:         "group",
		SelectClause: "group_name",
		GroupClause:  "group_name",
	},
	"stream": {
		Name:         "stream",
		SelectClause: "is_stream",
		GroupClause:  "is_stream",
	},
	"type": {
		Name:         "type",
		SelectClause: "type",
		GroupClause:  "type",
	},
}

var allowedOrderBy = map[string]string{
	"amount":            "amount",
	"quota":             "quota",
	"requests":          "requests",
	"total_tokens":      "total_tokens",
	"prompt_tokens":     "prompt_tokens",
	"completion_tokens": "completion_tokens",
	"avg_use_time_ms":   "avg_use_time_ms",
	"max_use_time_ms":   "max_use_time_ms",
}

func NormalizeRequest(req Request, defaultPageSize, maxPageSize int) (Request, error) {
	if req.StartTimestamp <= 0 {
		return Request{}, fmt.Errorf("start_timestamp is required")
	}
	if req.EndTimestamp <= 0 {
		return Request{}, fmt.Errorf("end_timestamp is required")
	}
	if req.EndTimestamp <= req.StartTimestamp {
		return Request{}, fmt.Errorf("end_timestamp must be greater than start_timestamp")
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = defaultPageSize
	}
	if req.PageSize > maxPageSize {
		req.PageSize = maxPageSize
	}
	if req.Type == nil {
		defaultType := 2
		req.Type = &defaultType
	}
	if req.OrderBy == "" {
		req.OrderBy = "amount"
	}
	if _, ok := allowedOrderBy[req.OrderBy]; !ok {
		return Request{}, fmt.Errorf("invalid order_by: %s", req.OrderBy)
	}
	if req.Order == "" {
		req.Order = "desc"
	}
	req.Order = strings.ToLower(req.Order)
	if req.Order != "asc" && req.Order != "desc" {
		return Request{}, fmt.Errorf("invalid order: %s", req.Order)
	}
	if _, err := parseGroupings(req.GroupBy); err != nil {
		return Request{}, err
	}
	return req, nil
}

func BuildSummarySQL(req Request) (string, []any) {
	whereSQL, args := buildWhere(req)
	return "SELECT " + metricsSQL() + " FROM logs_1m" + whereSQL, args
}

func BuildItemsSQL(req Request) (string, []any) {
	groupings, _ := parseGroupings(req.GroupBy)
	whereSQL, args := buildWhere(req)
	prefix := ""
	groupBy := ""
	if len(groupings) > 0 {
		selects := make([]string, 0, len(groupings))
		groups := make([]string, 0, len(groupings))
		for _, grouping := range groupings {
			selects = append(selects, grouping.SelectClause)
			groups = append(groups, grouping.GroupClause)
		}
		prefix = strings.Join(selects, ", ") + ", "
		groupBy = " GROUP BY " + strings.Join(groups, ", ")
	}

	sql := "SELECT " + prefix + metricsSQL() + " FROM logs_1m" + whereSQL + groupBy
	if len(groupings) > 0 {
		sql += fmt.Sprintf(" ORDER BY %s %s", allowedOrderBy[req.OrderBy], strings.ToUpper(req.Order))
	}
	sql += fmt.Sprintf(" LIMIT %d OFFSET %d", req.PageSize, (req.Page-1)*req.PageSize)
	return sql, args
}

func BuildCountSQL(req Request) (string, []any) {
	groupings, _ := parseGroupings(req.GroupBy)
	if len(groupings) == 0 {
		return "SELECT 1", nil
	}
	whereSQL, args := buildWhere(req)
	groups := make([]string, 0, len(groupings))
	for _, grouping := range groupings {
		groups = append(groups, grouping.GroupClause)
	}
	return "SELECT COUNT(*) FROM (SELECT 1 FROM logs_1m" + whereSQL + " GROUP BY " + strings.Join(groups, ", ") + ") grouped", args
}

func ParseGroupNames(groupBy string) ([]string, error) {
	groupings, err := parseGroupings(groupBy)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(groupings))
	for _, grouping := range groupings {
		names = append(names, grouping.Name)
	}
	return names, nil
}

func metricsSQL() string {
	return "COALESCE(SUM(count), 0) AS requests, " +
		"COALESCE(SUM(sum_quota), 0) AS quota, " +
		"COALESCE(SUM(sum_quota), 0) / 500000.0 AS amount, " +
		"COALESCE(SUM(sum_prompt_tokens), 0) AS prompt_tokens, " +
		"COALESCE(SUM(sum_completion_tokens), 0) AS completion_tokens, " +
		"COALESCE(SUM(sum_prompt_tokens + sum_completion_tokens), 0) AS total_tokens, " +
		"COALESCE(SUM(sum_use_time) / NULLIF(SUM(count), 0), 0) AS avg_use_time_ms, " +
		"COALESCE(MAX(max_use_time), 0) AS max_use_time_ms"
}

func parseGroupings(groupBy string) ([]Grouping, error) {
	if strings.TrimSpace(groupBy) == "" {
		return nil, nil
	}
	parts := strings.Split(groupBy, ",")
	result := make([]Grouping, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, raw := range parts {
		name := strings.TrimSpace(raw)
		grouping, ok := allowedGroupings[name]
		if !ok {
			return nil, fmt.Errorf("invalid group_by: %s", name)
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, grouping)
	}
	return result, nil
}

func buildWhere(req Request) (string, []any) {
	clauses := []string{"bucket_ts >= ?", "bucket_ts < ?"}
	args := []any{
		time.Unix(req.StartTimestamp, 0).UTC(),
		time.Unix(req.EndTimestamp, 0).UTC(),
	}
	if req.UserID > 0 {
		clauses = append(clauses, "user_id = ?")
		args = append(args, req.UserID)
	}
	if req.Username != "" {
		clauses = append(clauses, "username = ?")
		args = append(args, req.Username)
	}
	if req.TokenID > 0 {
		clauses = append(clauses, "token_id = ?")
		args = append(args, req.TokenID)
	}
	if req.TokenName != "" {
		clauses = append(clauses, "token_name = ?")
		args = append(args, req.TokenName)
	}
	if req.ChannelID > 0 {
		clauses = append(clauses, "channel_id = ?")
		args = append(args, req.ChannelID)
	}
	if req.ChannelName != "" {
		clauses = append(clauses, "channel_name = ?")
		args = append(args, req.ChannelName)
	}
	if req.ModelName != "" {
		clauses = append(clauses, "model_name = ?")
		args = append(args, req.ModelName)
	}
	if req.GroupName != "" {
		clauses = append(clauses, "group_name = ?")
		args = append(args, req.GroupName)
	}
	if req.IsStream != nil {
		clauses = append(clauses, "is_stream = ?")
		args = append(args, *req.IsStream)
	}
	if req.Type != nil {
		clauses = append(clauses, "type = ?")
		args = append(args, *req.Type)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}
