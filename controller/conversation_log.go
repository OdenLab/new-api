package controller

import (
	"errors"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetConversationLogs(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	startTime, err := parseConversationLogTimestamp(c.Query("start_time"))
	if err != nil {
		common.ApiError(c, errors.New("invalid conversation log time filter"))
		return
	}
	endTime, err := parseConversationLogTimestamp(c.Query("end_time"))
	if err != nil || (startTime > 0 && endTime > 0 && startTime > endTime) {
		common.ApiError(c, errors.New("invalid conversation log time filter"))
		return
	}
	filter := model.ConversationLogFilter{
		Username:  truncateFilter(c.Query("username")),
		TokenName: truncateFilter(c.Query("token_name")),
		ModelName: truncateFilter(c.Query("model_name")),
		StartTime: startTime,
		EndTime:   endTime,
	}
	items, total, err := model.GetConversationLogs(pageInfo.GetStartIdx(), pageInfo.GetPageSize(), filter)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func parseConversationLogTimestamp(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	timestamp, err := strconv.ParseInt(value, 10, 64)
	if err != nil || timestamp < 0 {
		return 0, errors.New("invalid timestamp")
	}
	return timestamp, nil
}

func truncateFilter(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) > 128 {
		return string(runes[:128])
	}
	return value
}
