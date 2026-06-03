package query

import (
	"strings"
	"testing"
)

func TestNormalizeRequestRejectsInvalidGroupBy(t *testing.T) {
	_, err := NormalizeRequest(Request{
		StartTimestamp: 1748736000,
		EndTimestamp:   1748822400,
		GroupBy:        "user,bad",
	}, 20, 100)
	if err == nil {
		t.Fatalf("expected invalid group_by error")
	}
}

func TestNormalizeRequestAppliesDefaults(t *testing.T) {
	req, err := NormalizeRequest(Request{
		StartTimestamp: 1748736000,
		EndTimestamp:   1748822400,
		UserID:         123,
	}, 20, 100)
	if err != nil {
		t.Fatalf("NormalizeRequest returned error: %v", err)
	}
	if req.Page != 1 {
		t.Fatalf("expected page 1, got %d", req.Page)
	}
	if req.PageSize != 20 {
		t.Fatalf("expected page size 20, got %d", req.PageSize)
	}
	if req.Type == nil || *req.Type != 2 {
		t.Fatalf("expected default type 2, got %+v", req.Type)
	}
	if req.OrderBy != "amount" || req.Order != "desc" {
		t.Fatalf("unexpected order defaults: %s %s", req.OrderBy, req.Order)
	}
}

func TestBuildGroupedItemsSQL(t *testing.T) {
	req, err := NormalizeRequest(Request{
		StartTimestamp: 1748736000,
		EndTimestamp:   1748822400,
		GroupBy:        "user,channel",
		UserID:         123,
		Page:           2,
		PageSize:       3,
		OrderBy:        "quota",
		Order:          "asc",
	}, 20, 100)
	if err != nil {
		t.Fatalf("NormalizeRequest returned error: %v", err)
	}

	sql, args := BuildItemsSQL(req)
	required := []string{
		"SELECT user_id, username, channel_id, channel_name",
		"FROM logs_1m",
		"bucket_ts >= ?",
		"bucket_ts < ?",
		"user_id = ?",
		"type = ?",
		"GROUP BY user_id, username, channel_id, channel_name",
		"ORDER BY quota ASC",
		"LIMIT 3 OFFSET 3",
	}
	for _, item := range required {
		if !strings.Contains(sql, item) {
			t.Fatalf("expected SQL to contain %q, got %s", item, sql)
		}
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestBuildSummarySQLWithoutGroupBy(t *testing.T) {
	req, err := NormalizeRequest(Request{
		StartTimestamp: 1748736000,
		EndTimestamp:   1748822400,
		Username:       "alice",
	}, 20, 100)
	if err != nil {
		t.Fatalf("NormalizeRequest returned error: %v", err)
	}

	sql, args := BuildSummarySQL(req)
	if strings.Contains(sql, "GROUP BY") {
		t.Fatalf("did not expect summary SQL to contain GROUP BY: %s", sql)
	}
	if !strings.Contains(sql, "username = ?") {
		t.Fatalf("expected username filter in SQL: %s", sql)
	}
	if len(args) != 4 {
		t.Fatalf("expected 4 args, got %d", len(args))
	}
}

func TestBuildCountSQLWithoutGroupByReturnsConstant(t *testing.T) {
	req, err := NormalizeRequest(Request{
		StartTimestamp: 1748736000,
		EndTimestamp:   1748822400,
	}, 20, 100)
	if err != nil {
		t.Fatalf("NormalizeRequest returned error: %v", err)
	}

	sql, args := BuildCountSQL(req)
	if sql != "SELECT 1" {
		t.Fatalf("unexpected count SQL: %s", sql)
	}
	if len(args) != 0 {
		t.Fatalf("expected no args, got %d", len(args))
	}
}
