package model

import (
	"errors"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type RequestTrace struct {
	Id                   int     `json:"id"`
	RequestID            string  `json:"request_id" gorm:"type:varchar(64);not null;uniqueIndex"`
	TraceID              string  `json:"trace_id" gorm:"type:varchar(64);index"`
	UserID               int     `json:"user_id" gorm:"index"`
	TokenID              int     `json:"token_id" gorm:"index"`
	Group                string  `json:"group" gorm:"size:64;index"`
	ExternalModel        string  `json:"external_model" gorm:"size:128;index"`
	InternalModel        string  `json:"internal_model" gorm:"size:128;index"`
	Provider             string  `json:"provider" gorm:"size:64;index"`
	ChannelID            int     `json:"channel_id" gorm:"index"`
	UpstreamModel        string  `json:"upstream_model" gorm:"size:128;index"`
	RetryCount           int     `json:"retry_count" gorm:"default:0"`
	FallbackUsed         bool    `json:"fallback_used"`
	StatusCode           int     `json:"status_code" gorm:"index"`
	UpstreamStatusCode   int     `json:"upstream_status_code" gorm:"index"`
	LatencyMS            int     `json:"latency_ms" gorm:"default:0"`
	PromptTokens         int     `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens     int     `json:"completion_tokens" gorm:"default:0"`
	TotalTokens          int     `json:"total_tokens" gorm:"default:0"`
	EstimatedCost        float64 `json:"estimated_cost" gorm:"default:0"`
	ActualCost           float64 `json:"actual_cost" gorm:"default:0"`
	ErrorType            string  `json:"error_type" gorm:"size:64;index"`
	ErrorMessageRedacted string  `json:"error_message_redacted" gorm:"type:text"`
	CreatedAt            int64   `json:"created_at" gorm:"bigint;index"`
	UpdatedAt            int64   `json:"updated_at" gorm:"bigint;index"`
}

type RequestTraceParams struct {
	RequestID            string
	TraceID              string
	UserID               int
	TokenID              int
	Group                string
	ExternalModel        string
	InternalModel        string
	Provider             string
	ChannelID            int
	UpstreamModel        string
	RetryCount           int
	FallbackUsed         bool
	StatusCode           int
	UpstreamStatusCode   int
	LatencyMS            int
	PromptTokens         int
	CompletionTokens     int
	TotalTokens          int
	EstimatedCost        float64
	ActualCost           float64
	ErrorType            string
	ErrorMessageRedacted string
}

type RequestTraceQuery struct {
	RequestID     string
	TraceID       string
	UserID        int
	TokenID       int
	Group         string
	ExternalModel string
	Provider      string
	ChannelID     int
	ErrorType     string
	StatusCode    int
	StartIdx      int
	Limit         int
}

func (t *RequestTrace) BeforeSave(tx *gorm.DB) error {
	t.RequestID = strings.TrimSpace(t.RequestID)
	t.TraceID = strings.TrimSpace(t.TraceID)
	t.Group = strings.TrimSpace(t.Group)
	t.ExternalModel = strings.TrimSpace(t.ExternalModel)
	t.InternalModel = strings.TrimSpace(t.InternalModel)
	t.Provider = strings.TrimSpace(strings.ToLower(t.Provider))
	t.UpstreamModel = strings.TrimSpace(t.UpstreamModel)
	t.ErrorType = strings.TrimSpace(strings.ToUpper(t.ErrorType))
	t.ErrorMessageRedacted = common.RedactSensitiveText(t.ErrorMessageRedacted)
	if t.TotalTokens <= 0 {
		t.TotalTokens = t.PromptTokens + t.CompletionTokens
	}
	return nil
}

func RecordRequestTrace(params RequestTraceParams) (*RequestTrace, error) {
	requestID := strings.TrimSpace(params.RequestID)
	if requestID == "" {
		return nil, errors.New("request_id is required")
	}

	now := common.GetTimestamp()
	trace := &RequestTrace{}
	err := LOG_DB.Where("request_id = ?", requestID).First(trace).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		trace.CreatedAt = now
	}

	applyRequestTraceParams(trace, params)
	trace.UpdatedAt = now
	if trace.CreatedAt == 0 {
		trace.CreatedAt = now
	}

	if trace.Id > 0 {
		if err := LOG_DB.Save(trace).Error; err != nil {
			return nil, err
		}
		return trace, nil
	}
	if err := LOG_DB.Select("*").Create(trace).Error; err != nil {
		return nil, err
	}
	return trace, nil
}

func GetRequestTraceByRequestID(requestID string) (*RequestTrace, bool, error) {
	requestID = strings.TrimSpace(requestID)
	if requestID == "" {
		return nil, false, nil
	}
	var trace RequestTrace
	err := LOG_DB.Where("request_id = ?", requestID).First(&trace).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &trace, true, nil
}

func ListRequestTraces(query RequestTraceQuery) ([]*RequestTrace, error) {
	tx := buildRequestTraceQuery(query)
	limit := query.Limit
	if limit <= 0 {
		limit = 10
	}
	var traces []*RequestTrace
	err := tx.Order("created_at desc").Limit(limit).Offset(query.StartIdx).Find(&traces).Error
	return traces, err
}

func CountRequestTraces(query RequestTraceQuery) (int64, error) {
	var total int64
	err := buildRequestTraceQuery(query).Count(&total).Error
	return total, err
}

func buildRequestTraceQuery(query RequestTraceQuery) *gorm.DB {
	tx := LOG_DB.Model(&RequestTrace{})
	if requestID := strings.TrimSpace(query.RequestID); requestID != "" {
		tx = tx.Where("request_id = ?", requestID)
	}
	if traceID := strings.TrimSpace(query.TraceID); traceID != "" {
		tx = tx.Where("trace_id = ?", traceID)
	}
	if query.UserID > 0 {
		tx = tx.Where("user_id = ?", query.UserID)
	}
	if query.TokenID > 0 {
		tx = tx.Where("token_id = ?", query.TokenID)
	}
	if group := strings.TrimSpace(query.Group); group != "" {
		tx = tx.Where(commonGroupCol+" = ?", group)
	}
	if modelName := strings.TrimSpace(query.ExternalModel); modelName != "" {
		tx = tx.Where("external_model = ?", modelName)
	}
	if provider := strings.TrimSpace(strings.ToLower(query.Provider)); provider != "" {
		tx = tx.Where("provider = ?", provider)
	}
	if query.ChannelID > 0 {
		tx = tx.Where("channel_id = ?", query.ChannelID)
	}
	if errorType := strings.TrimSpace(strings.ToUpper(query.ErrorType)); errorType != "" {
		tx = tx.Where("error_type = ?", errorType)
	}
	if query.StatusCode > 0 {
		tx = tx.Where("status_code = ?", query.StatusCode)
	}
	return tx
}

func applyRequestTraceParams(trace *RequestTrace, params RequestTraceParams) {
	trace.RequestID = params.RequestID
	trace.TraceID = params.TraceID
	trace.UserID = params.UserID
	trace.TokenID = params.TokenID
	trace.Group = params.Group
	trace.ExternalModel = params.ExternalModel
	trace.InternalModel = params.InternalModel
	trace.Provider = params.Provider
	trace.ChannelID = params.ChannelID
	trace.UpstreamModel = params.UpstreamModel
	trace.RetryCount = params.RetryCount
	trace.FallbackUsed = params.FallbackUsed
	trace.StatusCode = params.StatusCode
	trace.UpstreamStatusCode = params.UpstreamStatusCode
	trace.LatencyMS = params.LatencyMS
	trace.PromptTokens = params.PromptTokens
	trace.CompletionTokens = params.CompletionTokens
	trace.TotalTokens = params.TotalTokens
	trace.EstimatedCost = params.EstimatedCost
	trace.ActualCost = params.ActualCost
	trace.ErrorType = params.ErrorType
	trace.ErrorMessageRedacted = params.ErrorMessageRedacted
}
