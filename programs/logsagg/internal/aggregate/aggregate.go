package aggregate

import (
	"sort"
	"time"
)

type RawLog struct {
	ID               int64
	CreatedAt        int64
	UserID           int64
	Username         string
	ModelName        string
	ChannelID        int64
	ChannelName      string
	Type             int
	TokenID          int64
	TokenName        string
	GroupName        string
	IsStream         bool
	Quota            int64
	PromptTokens     int64
	CompletionTokens int64
	UseTime          int64
}

type Fact struct {
	BucketTS            time.Time
	UserID              int64
	Username            string
	ModelName           string
	ChannelID           int64
	ChannelName         string
	Type                int
	TokenID             int64
	TokenName           string
	GroupName           string
	IsStream            bool
	Count               int64
	SumQuota            int64
	SumPromptTokens     int64
	SumCompletionTokens int64
	SumUseTime          int64
	AvgUseTime          int64
	MaxUseTime          int64
	lastSeenID          int64
}

type bucketKey struct {
	BucketTS  time.Time
	UserID    int64
	ModelName string
	ChannelID int64
	Type      int
	TokenID   int64
	GroupName string
	IsStream  bool
}

func BuildBuckets(rows []RawLog) []Fact {
	grouped := make(map[bucketKey]*Fact, len(rows))
	for _, row := range rows {
		bucketTS := time.Unix(row.CreatedAt-row.CreatedAt%60, 0).UTC()
		key := bucketKey{
			BucketTS:  bucketTS,
			UserID:    row.UserID,
			ModelName: row.ModelName,
			ChannelID: row.ChannelID,
			Type:      row.Type,
			TokenID:   row.TokenID,
			GroupName: row.GroupName,
			IsStream:  row.IsStream,
		}
		fact := grouped[key]
		if fact == nil {
			fact = &Fact{
				BucketTS:   bucketTS,
				UserID:     row.UserID,
				ModelName:  row.ModelName,
				ChannelID:  row.ChannelID,
				Type:       row.Type,
				TokenID:    row.TokenID,
				GroupName:  row.GroupName,
				IsStream:   row.IsStream,
				lastSeenID: -1,
			}
			grouped[key] = fact
		}

		fact.Count++
		fact.SumQuota += row.Quota
		fact.SumPromptTokens += row.PromptTokens
		fact.SumCompletionTokens += row.CompletionTokens
		fact.SumUseTime += row.UseTime
		if row.UseTime > fact.MaxUseTime {
			fact.MaxUseTime = row.UseTime
		}
		if row.ID >= fact.lastSeenID {
			if row.Username != "" {
				fact.Username = row.Username
			}
			if row.ChannelName != "" {
				fact.ChannelName = row.ChannelName
			}
			if row.TokenName != "" {
				fact.TokenName = row.TokenName
			}
			fact.lastSeenID = row.ID
		}
	}

	facts := make([]Fact, 0, len(grouped))
	for _, fact := range grouped {
		if fact.Count > 0 {
			fact.AvgUseTime = fact.SumUseTime / fact.Count
		}
		facts = append(facts, *fact)
	}

	sort.Slice(facts, func(i, j int) bool {
		if !facts[i].BucketTS.Equal(facts[j].BucketTS) {
			return facts[i].BucketTS.Before(facts[j].BucketTS)
		}
		if facts[i].UserID != facts[j].UserID {
			return facts[i].UserID < facts[j].UserID
		}
		if facts[i].ModelName != facts[j].ModelName {
			return facts[i].ModelName < facts[j].ModelName
		}
		if facts[i].ChannelID != facts[j].ChannelID {
			return facts[i].ChannelID < facts[j].ChannelID
		}
		if facts[i].Type != facts[j].Type {
			return facts[i].Type < facts[j].Type
		}
		if facts[i].TokenID != facts[j].TokenID {
			return facts[i].TokenID < facts[j].TokenID
		}
		if facts[i].GroupName != facts[j].GroupName {
			return facts[i].GroupName < facts[j].GroupName
		}
		return !facts[i].IsStream && facts[j].IsStream
	})

	return facts
}
