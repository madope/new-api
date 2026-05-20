package aggregate

import (
	"testing"
	"time"
)

func TestBuildBucketsAggregatesSameMinute(t *testing.T) {
	rows := []RawLog{
		{
			ID:               100,
			CreatedAt:        1715659201,
			UserID:           1,
			Username:         "alice",
			ModelName:        "gpt-4o",
			ChannelID:        11,
			ChannelName:      "openai-main",
			Type:             2,
			TokenID:          9,
			TokenName:        "team-key",
			GroupName:        "default",
			IsStream:         true,
			Quota:            100,
			PromptTokens:     10,
			CompletionTokens: 20,
			UseTime:          5,
		},
		{
			ID:               101,
			CreatedAt:        1715659230,
			UserID:           1,
			Username:         "alice-new",
			ModelName:        "gpt-4o",
			ChannelID:        11,
			ChannelName:      "openai-main-v2",
			Type:             2,
			TokenID:          9,
			TokenName:        "team-key-2",
			GroupName:        "default",
			IsStream:         true,
			Quota:            200,
			PromptTokens:     30,
			CompletionTokens: 40,
			UseTime:          9,
		},
	}

	facts := BuildBuckets(rows)
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	got := facts[0]
	if got.Count != 2 || got.SumQuota != 300 {
		t.Fatalf("unexpected aggregates: %+v", got)
	}
	if got.AvgUseTime != 7 {
		t.Fatalf("expected avg_use_time=7, got %d", got.AvgUseTime)
	}
	if got.MaxUseTime != 9 {
		t.Fatalf("expected max_use_time=9, got %d", got.MaxUseTime)
	}
	if got.Username != "alice-new" || got.ChannelName != "openai-main-v2" || got.TokenName != "team-key-2" {
		t.Fatalf("expected latest non-empty display values, got %+v", got)
	}
	if !got.BucketTS.Equal(time.Unix(1715659200, 0).UTC()) {
		t.Fatalf("unexpected bucket time: %s", got.BucketTS)
	}
}

func TestBuildBucketsKeepsLatestNonEmptyName(t *testing.T) {
	rows := []RawLog{
		{ID: 10, CreatedAt: 1715659201, UserID: 1, Username: "alice", ModelName: "m", ChannelID: 1, ChannelName: "c1", Type: 1, TokenID: 1, TokenName: "t1", GroupName: "g", Quota: 1, UseTime: 1},
		{ID: 11, CreatedAt: 1715659205, UserID: 1, Username: "", ModelName: "m", ChannelID: 1, ChannelName: "", Type: 1, TokenID: 1, TokenName: "", GroupName: "g", Quota: 1, UseTime: 2},
	}

	facts := BuildBuckets(rows)
	if len(facts) != 1 {
		t.Fatalf("expected 1 fact, got %d", len(facts))
	}
	got := facts[0]
	if got.Username != "alice" || got.ChannelName != "c1" || got.TokenName != "t1" {
		t.Fatalf("expected latest non-empty values to be retained, got %+v", got)
	}
}
