package model

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"gorm.io/gorm"
)

const (
	ProviderAuthBearer = "bearer"
	ProviderAuthAPIKey = "api_key"
	ProviderAuthAWS    = "aws"
	ProviderAuthNone   = "none"

	ProviderHealthHealthy  = "healthy"
	ProviderHealthDegraded = "degraded"
	ProviderHealthDisabled = "disabled"
)

type ProviderRegistry struct {
	Id           int    `json:"id"`
	Provider     string `json:"provider" gorm:"size:64;not null;uniqueIndex"`
	Protocol     string `json:"protocol" gorm:"size:64;not null;index"`
	BaseURL      string `json:"base_url" gorm:"type:text"`
	AuthType     string `json:"auth_type" gorm:"size:32;not null;default:bearer"`
	Enabled      bool   `json:"enabled" gorm:"index"`
	HealthStatus string `json:"health_status" gorm:"size:32;not null;default:healthy;index"`
	CreatedTime  int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime  int64  `json:"updated_time" gorm:"bigint"`
}

type ProviderMetadata struct {
	ChannelType int
	APIType     int
	Provider    string
	Protocol    string
	BaseURL     string
	AuthType    string
	Task        bool
}

func (p *ProviderRegistry) BeforeSave(tx *gorm.DB) error {
	p.Provider = strings.TrimSpace(strings.ToLower(p.Provider))
	p.Protocol = strings.TrimSpace(strings.ToLower(p.Protocol))
	p.BaseURL = strings.TrimRight(strings.TrimSpace(p.BaseURL), "/")
	p.AuthType = strings.TrimSpace(strings.ToLower(p.AuthType))
	p.HealthStatus = strings.TrimSpace(strings.ToLower(p.HealthStatus))
	if p.AuthType == "" {
		p.AuthType = ProviderAuthBearer
	}
	if p.HealthStatus == "" {
		p.HealthStatus = ProviderHealthHealthy
	}
	return nil
}

func (p *ProviderRegistry) Insert() error {
	now := common.GetTimestamp()
	p.CreatedTime = now
	p.UpdatedTime = now
	return DB.Select("*").Create(p).Error
}

func (p *ProviderRegistry) Update() error {
	p.UpdatedTime = common.GetTimestamp()
	return DB.Model(&ProviderRegistry{}).Where("id = ?", p.Id).
		Select("provider", "protocol", "base_url", "auth_type", "enabled", "health_status", "updated_time").
		Updates(p).Error
}

func GetProviderRegistryByProvider(provider string) (*ProviderRegistry, bool, error) {
	provider = strings.TrimSpace(strings.ToLower(provider))
	if provider == "" {
		return nil, false, nil
	}
	var registry ProviderRegistry
	err := DB.Where("provider = ? AND enabled = ?", provider, true).First(&registry).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &registry, true, nil
}

func GetDefaultProviderMetadataByChannelType(channelType int) ProviderMetadata {
	apiType, _ := common.ChannelType2APIType(channelType)
	meta := ProviderMetadata{
		ChannelType: channelType,
		APIType:     apiType,
		Provider:    strings.ToLower(constant.GetChannelTypeName(channelType)),
		Protocol:    ModelProtocolOpenAICompatible,
		AuthType:    ProviderAuthBearer,
	}
	if channelType >= 0 && channelType < len(constant.ChannelBaseURLs) {
		meta.BaseURL = constant.ChannelBaseURLs[channelType]
	}

	switch channelType {
	case constant.ChannelTypeOpenAI, constant.ChannelTypeCustom, constant.ChannelTypeOpenAIMax,
		constant.ChannelTypeOhMyGPT, constant.ChannelTypeAIProxy, constant.ChannelTypeAPI2GPT,
		constant.ChannelTypeAIGC2D, constant.ChannelTypeSora, constant.ChannelTypeCodex:
		meta.Provider = strings.ToLower(constant.GetChannelTypeName(channelType))
		meta.Protocol = ModelProtocolOpenAI
	case constant.ChannelTypeAzure:
		meta.Provider = "azure-openai"
		meta.Protocol = ModelProtocolAzureOpenAI
	case constant.ChannelTypeAnthropic, constant.ChannelTypeMoonshot:
		if channelType == constant.ChannelTypeMoonshot {
			meta.Provider = "moonshot"
		} else {
			meta.Provider = "anthropic"
		}
		meta.Protocol = ModelProtocolClaude
	case constant.ChannelTypeGemini, constant.ChannelTypeVertexAi:
		if channelType == constant.ChannelTypeVertexAi {
			meta.Provider = "vertex-ai"
		} else {
			meta.Provider = "gemini"
		}
		meta.Protocol = ModelProtocolGemini
	case constant.ChannelTypeDeepSeek:
		meta.Provider = "deepseek"
		meta.Protocol = ModelProtocolOpenAICompatible
	case constant.ChannelTypeAli:
		meta.Provider = "qwen"
		meta.Protocol = ModelProtocolDashScope
	case constant.ChannelTypeAws:
		meta.Provider = "bedrock"
		meta.Protocol = ModelProtocolBedrock
		meta.AuthType = ProviderAuthAWS
	case constant.ChannelTypeOllama, constant.ChannelTypeXinference:
		meta.Provider = strings.ToLower(constant.GetChannelTypeName(channelType))
		meta.Protocol = ModelProtocolLocal
	case constant.ChannelTypeSunoAPI, constant.ChannelTypeKling, constant.ChannelTypeJimeng,
		constant.ChannelTypeVidu, constant.ChannelTypeDoubaoVideo, constant.ChannelTypeReplicate:
		meta.Provider = strings.ToLower(constant.GetChannelTypeName(channelType))
		if channelType == constant.ChannelTypeSunoAPI {
			meta.Provider = "suno"
		}
		meta.Protocol = ModelProtocolTask
		meta.Task = true
	}
	return meta
}
