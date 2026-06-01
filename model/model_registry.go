package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ModelProtocolOpenAI           = "openai"
	ModelProtocolOpenAICompatible = "openai-compatible"
	ModelProtocolClaude           = "claude"
	ModelProtocolGemini           = "gemini"
	ModelProtocolAzureOpenAI      = "azure-openai"
	ModelProtocolBedrock          = "bedrock"
	ModelProtocolDashScope        = "dashscope"
	ModelProtocolTask             = "task"
	ModelProtocolLocal            = "local"
)

type ModelRegistry struct {
	Id              int    `json:"id"`
	ExternalModel   string `json:"external_model" gorm:"size:128;not null;uniqueIndex"`
	Provider        string `json:"provider" gorm:"size:64;not null;index"`
	UpstreamModel   string `json:"upstream_model" gorm:"size:128;not null;index"`
	Protocol        string `json:"protocol" gorm:"size:64;not null;index"`
	Capabilities    string `json:"capabilities" gorm:"type:text"`
	ContextWindow   int    `json:"context_window" gorm:"default:0"`
	MaxOutputTokens int    `json:"max_output_tokens" gorm:"default:0"`
	Enabled         bool   `json:"enabled" gorm:"index"`
	Priority        int    `json:"priority" gorm:"default:0;index"`
	CreatedTime     int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime     int64  `json:"updated_time" gorm:"bigint"`
}

func (m *ModelRegistry) BeforeSave(tx *gorm.DB) error {
	m.ExternalModel = strings.TrimSpace(m.ExternalModel)
	m.Provider = strings.TrimSpace(strings.ToLower(m.Provider))
	m.UpstreamModel = strings.TrimSpace(m.UpstreamModel)
	m.Protocol = strings.TrimSpace(strings.ToLower(m.Protocol))
	m.Capabilities = normalizeCSVLikeString(m.Capabilities)
	if m.UpstreamModel == "" {
		m.UpstreamModel = m.ExternalModel
	}
	return nil
}

func (m *ModelRegistry) Insert() error {
	now := common.GetTimestamp()
	m.CreatedTime = now
	m.UpdatedTime = now
	return DB.Select("*").Create(m).Error
}

func (m *ModelRegistry) Update() error {
	m.UpdatedTime = common.GetTimestamp()
	return DB.Model(&ModelRegistry{}).Where("id = ?", m.Id).
		Select("external_model", "provider", "upstream_model", "protocol", "capabilities", "context_window", "max_output_tokens", "enabled", "priority", "updated_time").
		Updates(m).Error
}

func GetModelRegistryByExternalModel(externalModel string) (*ModelRegistry, bool, error) {
	externalModel = strings.TrimSpace(externalModel)
	if externalModel == "" {
		return nil, false, nil
	}
	var registry ModelRegistry
	err := DB.Where("external_model = ? AND enabled = ?", externalModel, true).
		Order("priority DESC").
		First(&registry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &registry, true, nil
}

func normalizeCSVLikeString(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.Split(value, ",")
	seen := make(map[string]struct{}, len(parts))
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		normalized = append(normalized, part)
	}
	return strings.Join(normalized, ",")
}
