package model

import (
	"errors"
	"os"
	"strings"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/bytedance/gopkg/util/gopool"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type ConversationLog struct {
	Id         int    `json:"id" gorm:"primaryKey"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint;index:idx_conv_created_at"`
	UserId     int    `json:"user_id" gorm:"index:idx_conv_user_token"`
	Username   string `json:"username" gorm:"index;default:''"`
	TokenId    int    `json:"token_id" gorm:"index:idx_conv_user_token"`
	TokenName  string `json:"token_name" gorm:"index;default:''"`
	ModelName  string `json:"model_name" gorm:"index;default:''"`
	RequestId  string `json:"request_id" gorm:"type:varchar(64);index;default:''"`
	PromptText string `json:"prompt_text" gorm:"type:text"`
	ReplyText  string `json:"reply_text" gorm:"type:text"`
}

type ConversationLogFilter struct {
	Username  string
	TokenName string
	ModelName string
	StartTime int64
	EndTime   int64
}

var (
	CONV_LOG_DB              = LOG_DB
	lastConversationLogPurge atomic.Int64
)

func ensureConversationLogDB() error {
	if CONV_LOG_DB != nil {
		return nil
	}
	return InitConversationLogDB()
}

func InitConversationLogDB() error {
	if os.Getenv("CONV_LOG_SQL_DSN") == "" {
		db, err := gorm.Open(sqlite.Open("oneapi-conversation-logs.db"), &gorm.Config{PrepareStmt: true})
		if err != nil {
			return err
		}
		CONV_LOG_DB = db
	} else {
		db, _, err := chooseDB("CONV_LOG_SQL_DSN", true)
		if err != nil {
			return err
		}
		CONV_LOG_DB = db
	}
	if err := configureDBPool(CONV_LOG_DB, "CONV_LOG_SQL", 2, 10); err != nil {
		return err
	}
	return CONV_LOG_DB.AutoMigrate(&ConversationLog{})
}

func RecordConversationLogAsync(c ConversationLog) {
	if !common.IsMasterNode || !setting.ConversationLogEnabled {
		return
	}
	if c.PromptText == "" && c.ReplyText == "" {
		return
	}
	if c.CreatedAt == 0 {
		c.CreatedAt = time.Now().Unix()
	}
	maxTextBytes := common.GetEnvOrDefault("CONVERSATION_LOG_MAX_TEXT_BYTES", 64<<10)
	c.PromptText = truncateConversationLogText(c.PromptText, maxTextBytes)
	c.ReplyText = truncateConversationLogText(c.ReplyText, maxTextBytes)
	c.Username = strings.TrimSpace(c.Username)
	c.TokenName = strings.TrimSpace(c.TokenName)
	c.ModelName = strings.TrimSpace(c.ModelName)

	gopool.Go(func() {
		if err := ensureConversationLogDB(); err != nil {
			common.SysError("failed to initialize conversation log database: " + err.Error())
			return
		}
		if err := CONV_LOG_DB.Create(&c).Error; err != nil {
			common.SysError("failed to record conversation log: " + err.Error())
			return
		}
		purgeExpiredConversationLogs()
	})
}

func truncateConversationLogText(value string, maxBytes int) string {
	if maxBytes <= 0 || len(value) <= maxBytes {
		return value
	}
	value = value[:maxBytes]
	for len(value) > 0 && !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value + "\n...[truncated]"
}

func purgeExpiredConversationLogs() {
	retentionDays := common.GetEnvOrDefault("CONVERSATION_LOG_RETENTION_DAYS", 30)
	if retentionDays <= 0 || CONV_LOG_DB == nil {
		return
	}
	now := time.Now().Unix()
	last := lastConversationLogPurge.Load()
	if now-last < int64(time.Hour.Seconds()) || !lastConversationLogPurge.CompareAndSwap(last, now) {
		return
	}
	cutoff := now - int64(retentionDays)*int64((24*time.Hour).Seconds())
	if err := CONV_LOG_DB.Where("created_at < ?", cutoff).Delete(&ConversationLog{}).Error; err != nil {
		common.SysError("failed to purge expired conversation logs: " + err.Error())
	}
}

func GetConversationLogs(offset, limit int, filter ConversationLogFilter) ([]*ConversationLog, int64, error) {
	if err := ensureConversationLogDB(); err != nil {
		return nil, 0, errors.New("conversation log database is unavailable")
	}
	if offset < 0 {
		offset = 0
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	tx := CONV_LOG_DB.Model(&ConversationLog{})
	if filter.Username != "" {
		tx = tx.Where("username = ?", filter.Username)
	}
	if filter.TokenName != "" {
		tx = tx.Where("token_name = ?", filter.TokenName)
	}
	if filter.ModelName != "" {
		tx = tx.Where("model_name = ?", filter.ModelName)
	}
	if filter.StartTime > 0 {
		tx = tx.Where("created_at >= ?", filter.StartTime)
	}
	if filter.EndTime > 0 {
		tx = tx.Where("created_at <= ?", filter.EndTime)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	items := make([]*ConversationLog, 0, limit)
	err := tx.Order("id desc").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}
