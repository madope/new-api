package source

import (
	"testing"
	"time"
)

func TestIncrementalQueryUsesLastID(t *testing.T) {
	query, args := BuildIncrementalQuery(1234, 500)
	if query != "SELECT id, created_at, user_id, username, model_name, channel_id, channel_name, type, token_id, token_name, \"group\" AS group_name, is_stream, quota, prompt_tokens, completion_tokens, use_time FROM logs WHERE id > ? ORDER BY id ASC LIMIT ?" {
		t.Fatalf("unexpected query: %s", query)
	}
	if args[0] != int64(1234) || args[1] != 500 {
		t.Fatalf("unexpected args: %#v", args)
	}
}

func TestWindowQueryUsesUnixSecondsWindow(t *testing.T) {
	start := time.Unix(1715659200, 0).UTC()
	end := start.Add(30 * time.Minute)
	query, args := BuildWindowQuery(start, end)
	if args[0] != start.Unix() || args[1] != end.Unix() {
		t.Fatalf("unexpected args: %#v", args)
	}
	if query == "" {
		t.Fatalf("expected non-empty query")
	}
}
