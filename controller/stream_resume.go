package controller

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

type streamResumeCaptureWriter struct {
	gin.ResponseWriter
	id       string
	buffer   bytes.Buffer
	overflow bool
}

func (w *streamResumeCaptureWriter) Write(data []byte) (int, error) {
	w.capture(data)
	return w.ResponseWriter.Write(data)
}

func (w *streamResumeCaptureWriter) WriteString(data string) (int, error) {
	w.capture([]byte(data))
	return w.ResponseWriter.WriteString(data)
}

func (w *streamResumeCaptureWriter) capture(data []byte) {
	if w.overflow || len(data) == 0 {
		return
	}
	if w.buffer.Len()+len(data) > model.StreamResumeMaxPayloadBytes {
		w.overflow = true
		w.buffer.Reset()
		return
	}
	_, _ = w.buffer.Write(data)
}

func prepareStreamResume(c *gin.Context, request dto.Request, relayFormat types.RelayFormat) (*streamResumeCaptureWriter, bool, error) {
	if relayFormat != types.RelayFormatOpenAI {
		return nil, false, nil
	}
	req, ok := request.(*dto.GeneralOpenAIRequest)
	if !ok || req == nil {
		return nil, false, nil
	}

	enabled := req.StreamResume != nil && *req.StreamResume
	resumeID := strings.TrimSpace(req.ResumeID)
	// These fields are gateway controls and must never be forwarded upstream.
	req.StreamResume = nil
	req.ResumeID = ""
	if !enabled {
		return nil, false, nil
	}
	if req.Stream == nil || !*req.Stream {
		return nil, false, errors.New("stream_resume requires stream=true")
	}

	userID := c.GetInt("id")
	tokenID := c.GetInt("token_id")
	if resumeID != "" {
		rec, err := model.GetResumeRecord(resumeID)
		if err != nil {
			if model.IsRecordNotFound(err) {
				return nil, false, errors.New("resume record not found")
			}
			logger.LogError(c, "failed to load stream resume record: "+err.Error())
			return nil, false, errors.New("stream resume is temporarily unavailable")
		}
		if rec.UserID != userID || rec.TokenID != tokenID {
			return nil, false, errors.New("resume record not found")
		}
		switch rec.Status {
		case model.ResumeStatusRunning:
			return nil, false, errors.New("resume stream is still running")
		case model.ResumeStatusExpired:
			return nil, false, errors.New("resume stream has expired")
		case model.ResumeStatusFailed:
			return nil, false, errors.New("resume stream is unavailable; regenerate the response")
		case model.ResumeStatusDone:
			if rec.Payload == "" {
				return nil, false, errors.New("resume stream is unavailable; regenerate the response")
			}
			c.Header("X-Resume-Id", rec.ID)
			helper.SetEventStreamHeaders(c)
			if _, err := c.Writer.Write([]byte(rec.Payload)); err != nil {
				return nil, false, fmt.Errorf("failed to replay stream: %w", err)
			}
			if err := helper.FlushWriter(c); err != nil {
				return nil, false, fmt.Errorf("failed to flush replayed stream: %w", err)
			}
			return nil, true, nil
		default:
			return nil, false, errors.New("resume stream has an invalid state")
		}
	}

	resumeID = "rs_" + common.GetUUID()
	if err := model.CreateRunningResumeRecord(resumeID, userID, tokenID); err != nil {
		logger.LogError(c, "failed to create stream resume record: "+err.Error())
		return nil, false, errors.New("stream resume is temporarily unavailable")
	}
	c.Header("X-Resume-Id", resumeID)
	capture := &streamResumeCaptureWriter{ResponseWriter: c.Writer, id: resumeID}
	c.Writer = capture
	return capture, false, nil
}

func finalizeStreamResume(c *gin.Context, capture *streamResumeCaptureWriter, success bool) {
	if capture == nil {
		return
	}
	if !success || capture.overflow || capture.buffer.Len() == 0 || c.Request.Context().Err() != nil {
		if err := model.FailResumeRecord(capture.id); err != nil {
			logger.LogError(c, "failed to mark stream resume record as failed: "+err.Error())
		}
		return
	}
	if err := model.CompleteResumeRecord(capture.id, capture.buffer.String()); err != nil {
		logger.LogError(c, "failed to complete stream resume record: "+err.Error())
		_ = model.FailResumeRecord(capture.id)
	}
}
