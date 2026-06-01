package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type AuditEvent struct {
	Id           int    `json:"id"`
	ActorID      int    `json:"actor_id" gorm:"index"`
	ActorRole    int    `json:"actor_role" gorm:"index"`
	Action       string `json:"action" gorm:"size:128;not null;index"`
	ResourceType string `json:"resource_type" gorm:"size:64;not null;index"`
	ResourceID   string `json:"resource_id" gorm:"size:128;index"`
	SourceIP     string `json:"source_ip" gorm:"size:64;index"`
	RequestID    string `json:"request_id" gorm:"size:64;index"`
	Result       string `json:"result" gorm:"size:32;index"`
	DiffRedacted string `json:"diff_redacted" gorm:"type:text"`
	Method       string `json:"method" gorm:"size:16;index"`
	Path         string `json:"path" gorm:"type:text"`
	StatusCode   int    `json:"status_code" gorm:"index"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;index"`
}

type AuditEventParams struct {
	ActorID      int
	ActorRole    int
	Action       string
	ResourceType string
	ResourceID   string
	SourceIP     string
	RequestID    string
	Result       string
	DiffRedacted string
	Method       string
	Path         string
	StatusCode   int
}

type AuditEventQuery struct {
	RequestID    string
	ActorID      int
	Action       string
	ResourceType string
	ResourceID   string
	StartIdx     int
	Limit        int
}

func RecordAuditEvent(params AuditEventParams) (*AuditEvent, error) {
	event := &AuditEvent{
		ActorID:      params.ActorID,
		ActorRole:    params.ActorRole,
		Action:       normalizeAuditPart(params.Action),
		ResourceType: normalizeAuditPart(params.ResourceType),
		ResourceID:   strings.TrimSpace(params.ResourceID),
		SourceIP:     strings.TrimSpace(params.SourceIP),
		RequestID:    strings.TrimSpace(params.RequestID),
		Result:       normalizeAuditPart(params.Result),
		DiffRedacted: common.RedactSensitiveText(params.DiffRedacted),
		Method:       strings.ToUpper(strings.TrimSpace(params.Method)),
		Path:         strings.TrimSpace(params.Path),
		StatusCode:   params.StatusCode,
		CreatedAt:    common.GetTimestamp(),
	}
	if event.Result == "" {
		event.Result = "unknown"
	}
	if err := LOG_DB.Select("*").Create(event).Error; err != nil {
		return nil, err
	}
	return event, nil
}

func ListAuditEvents(query AuditEventQuery) ([]*AuditEvent, error) {
	tx := buildAuditEventQuery(query)
	limit := query.Limit
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var events []*AuditEvent
	err := tx.Order("id desc").Offset(query.StartIdx).Limit(limit).Find(&events).Error
	return events, err
}

func CountAuditEvents(query AuditEventQuery) (int64, error) {
	var total int64
	err := buildAuditEventQuery(query).Count(&total).Error
	return total, err
}

func buildAuditEventQuery(query AuditEventQuery) *gorm.DB {
	tx := LOG_DB.Model(&AuditEvent{})
	if query.RequestID != "" {
		tx = tx.Where("request_id = ?", strings.TrimSpace(query.RequestID))
	}
	if query.ActorID > 0 {
		tx = tx.Where("actor_id = ?", query.ActorID)
	}
	if query.Action != "" {
		tx = tx.Where("action = ?", normalizeAuditPart(query.Action))
	}
	if query.ResourceType != "" {
		tx = tx.Where("resource_type = ?", normalizeAuditPart(query.ResourceType))
	}
	if query.ResourceID != "" {
		tx = tx.Where("resource_id = ?", strings.TrimSpace(query.ResourceID))
	}
	return tx
}

func normalizeAuditPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, " ", "_")
	return value
}
