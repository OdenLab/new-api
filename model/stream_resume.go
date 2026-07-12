package model

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

const (
	ResumeStatusRunning = "RUNNING"
	ResumeStatusDone    = "DONE"
	ResumeStatusFailed  = "FAILED"
	ResumeStatusExpired = "EXPIRED"
	StreamResumeTTL     = 30 * time.Minute

	StreamResumeMaxPayloadBytes = 8 << 20
)

type StreamResumeRecord struct {
	ID        string `gorm:"primaryKey;type:varchar(64)"`
	UserID    int    `gorm:"index:idx_resume_owner;not null;default:0"`
	TokenID   int    `gorm:"index:idx_resume_owner;not null;default:0"`
	Status    string `gorm:"type:varchar(16);index"`
	Payload   string `gorm:"type:text"`
	CreatedAt int64  `gorm:"bigint;index"`
	UpdatedAt int64  `gorm:"bigint"`
	ExpiresAt int64  `gorm:"bigint;index"`
}

var RESUME_DB *gorm.DB

func ensureResumeDB() error {
	if RESUME_DB != nil {
		return nil
	}
	return InitResumeDB()
}

func InitResumeDB() error {
	if os.Getenv("RESUME_SQL_DSN") == "" {
		db, err := gorm.Open(sqlite.Open("resume.db"), &gorm.Config{PrepareStmt: true})
		if err != nil {
			return err
		}
		RESUME_DB = db
	} else {
		db, _, err := chooseDB("RESUME_SQL_DSN", true)
		if err != nil {
			return err
		}
		RESUME_DB = db
	}
	if err := configureDBPool(RESUME_DB, "RESUME_SQL", 2, 10); err != nil {
		return err
	}
	return RESUME_DB.AutoMigrate(&StreamResumeRecord{})
}

func CreateRunningResumeRecord(id string, userID, tokenID int) error {
	if id == "" {
		return errors.New("resume id is required")
	}
	if err := ensureResumeDB(); err != nil {
		return err
	}
	now := time.Now().Unix()
	rec := &StreamResumeRecord{
		ID:        id,
		UserID:    userID,
		TokenID:   tokenID,
		Status:    ResumeStatusRunning,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now + int64(StreamResumeTTL.Seconds()),
	}
	return RESUME_DB.Create(rec).Error
}

func CompleteResumeRecord(id, payload string) error {
	if len(payload) > StreamResumeMaxPayloadBytes {
		return fmt.Errorf("resume payload exceeds %d bytes", StreamResumeMaxPayloadBytes)
	}
	return updateResumeRecord(id, ResumeStatusDone, payload)
}

func FailResumeRecord(id string) error {
	return updateResumeRecord(id, ResumeStatusFailed, "")
}

func updateResumeRecord(id, status, payload string) error {
	if err := ensureResumeDB(); err != nil {
		return err
	}
	now := time.Now().Unix()
	result := RESUME_DB.Model(&StreamResumeRecord{}).
		Where("id = ? AND status = ?", id, ResumeStatusRunning).
		Updates(map[string]any{
			"status":     status,
			"payload":    payload,
			"updated_at": now,
			"expires_at": now + int64(StreamResumeTTL.Seconds()),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("resume record is no longer running")
	}
	return nil
}

func GetResumeRecord(id string) (*StreamResumeRecord, error) {
	if err := ensureResumeDB(); err != nil {
		return nil, err
	}
	var rec StreamResumeRecord
	if err := RESUME_DB.Where("id = ?", id).First(&rec).Error; err != nil {
		return nil, err
	}
	if rec.ExpiresAt <= time.Now().Unix() {
		rec.Status = ResumeStatusExpired
	}
	return &rec, nil
}

func IsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
