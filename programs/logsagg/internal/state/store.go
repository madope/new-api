package state

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DefaultJobName = "logsagg"

type Progress struct {
	JobName              string    `gorm:"column:job_name;primaryKey"`
	LastID               int64     `gorm:"column:last_id"`
	LastSuccessBucket    time.Time `gorm:"column:last_success_bucket"`
	LastIncrementalRunAt time.Time `gorm:"column:last_incremental_run_at"`
	LastBackfillRunAt    time.Time `gorm:"column:last_backfill_run_at"`
	UpdatedAt            time.Time `gorm:"column:updated_at"`
}

func (Progress) TableName() string {
	return "logs_agg_state"
}

func (p *Progress) Advance(nextID int64, bucket time.Time, now time.Time, backfill bool) error {
	if nextID < p.LastID {
		return fmt.Errorf("next id %d is less than current last_id %d", nextID, p.LastID)
	}
	p.JobName = valueOrDefault(p.JobName, DefaultJobName)
	p.LastID = nextID
	p.LastSuccessBucket = bucket.UTC()
	p.UpdatedAt = now.UTC()
	if backfill {
		p.LastBackfillRunAt = now.UTC()
		return nil
	}
	p.LastIncrementalRunAt = now.UTC()
	return nil
}

type Store struct {
	DB *gorm.DB
}

func (s Store) Load(jobName string) (Progress, error) {
	progress := defaultProgress(jobName)
	err := s.DB.Where("job_name = ?", progress.JobName).First(&progress).Error
	if err == nil {
		return progress, nil
	}
	if err != gorm.ErrRecordNotFound {
		return Progress{}, err
	}
	if err = s.Save(progress); err != nil {
		return Progress{}, err
	}
	return progress, nil
}

func (s Store) Save(progress Progress) error {
	progress.JobName = valueOrDefault(progress.JobName, DefaultJobName)
	progress.UpdatedAt = zeroAwareUTC(progress.UpdatedAt)
	progress.LastSuccessBucket = zeroAwareUTC(progress.LastSuccessBucket)
	progress.LastIncrementalRunAt = zeroAwareUTC(progress.LastIncrementalRunAt)
	progress.LastBackfillRunAt = zeroAwareUTC(progress.LastBackfillRunAt)
	return s.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "job_name"}},
		UpdateAll: true,
	}).Create(&progress).Error
}

func defaultProgress(jobName string) Progress {
	return Progress{
		JobName:           valueOrDefault(jobName, DefaultJobName),
		LastSuccessBucket: time.Unix(0, 0).UTC(),
		UpdatedAt:         time.Now().UTC(),
	}
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func zeroAwareUTC(value time.Time) time.Time {
	if value.IsZero() {
		return value
	}
	return value.UTC()
}
