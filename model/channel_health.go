package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	ChannelHealthStateHealthy     = "healthy"
	ChannelHealthStateDegraded    = "degraded"
	ChannelHealthStateCooldown    = "cooldown"
	ChannelHealthStateOpenCircuit = "open_circuit"
	ChannelHealthStateDisabled    = "disabled"

	channelHealthErrorAuth      = "AUTH_ERROR"
	channelHealthErrorRateLimit = "RATE_LIMIT"
	channelHealthErrorServer    = "SERVER_ERROR"
	channelHealthErrorTimeout   = "TIMEOUT"
)

type ChannelHealth struct {
	Id               int    `json:"id"`
	ChannelID        int    `json:"channel_id" gorm:"not null;uniqueIndex"`
	Provider         string `json:"provider" gorm:"size:64;index"`
	ModelName        string `json:"model_name" gorm:"size:128;index"`
	SuccessCount     int64  `json:"success_count" gorm:"bigint;default:0"`
	FailureCount     int64  `json:"failure_count" gorm:"bigint;default:0"`
	RateLimitCount   int64  `json:"rate_limit_count" gorm:"bigint;default:0"`
	ServerErrorCount int64  `json:"server_error_count" gorm:"bigint;default:0"`
	TimeoutCount     int64  `json:"timeout_count" gorm:"bigint;default:0"`
	AuthErrorCount   int64  `json:"auth_error_count" gorm:"bigint;default:0"`
	P95Latency       int    `json:"p95_latency" gorm:"default:0"`
	HealthScore      int    `json:"health_score" gorm:"default:100;index"`
	CircuitState     string `json:"circuit_state" gorm:"size:32;not null;default:healthy;index"`
	CooldownUntil    int64  `json:"cooldown_until" gorm:"bigint;index"`
	LastErrorAt      int64  `json:"last_error_at" gorm:"bigint"`
	LastErrorType    string `json:"last_error_type" gorm:"size:64"`
	LastErrorMessage string `json:"last_error_message" gorm:"type:text"`
	CreatedTime      int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime      int64  `json:"updated_time" gorm:"bigint"`
}

func (h *ChannelHealth) BeforeSave(tx *gorm.DB) error {
	h.Provider = strings.TrimSpace(strings.ToLower(h.Provider))
	h.ModelName = strings.TrimSpace(h.ModelName)
	h.CircuitState = strings.TrimSpace(strings.ToLower(h.CircuitState))
	h.LastErrorType = strings.TrimSpace(strings.ToUpper(h.LastErrorType))
	if h.CircuitState == "" {
		h.CircuitState = ChannelHealthStateHealthy
	}
	if h.HealthScore < 0 {
		h.HealthScore = 0
	}
	if h.HealthScore > 100 {
		h.HealthScore = 100
	}
	return nil
}

func GetChannelHealth(channelID int) (*ChannelHealth, bool, error) {
	if channelID <= 0 {
		return nil, false, nil
	}
	var health ChannelHealth
	err := DB.Where("channel_id = ?", channelID).First(&health).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return &health, true, nil
}

func getOrCreateChannelHealth(channelID int, provider string, modelName string) (*ChannelHealth, error) {
	if channelID <= 0 {
		return nil, nil
	}
	health, found, err := GetChannelHealth(channelID)
	if err != nil {
		return nil, err
	}
	if found {
		return health, nil
	}

	now := common.GetTimestamp()
	health = &ChannelHealth{
		ChannelID:    channelID,
		Provider:     provider,
		ModelName:    modelName,
		HealthScore:  100,
		CircuitState: ChannelHealthStateHealthy,
		CreatedTime:  now,
		UpdatedTime:  now,
	}
	if err := DB.Select("*").Create(health).Error; err != nil {
		return nil, err
	}
	return health, nil
}

func RecordChannelHealthSuccess(channelID int, provider string, modelName string, latencyMs int) (*ChannelHealth, error) {
	health, err := getOrCreateChannelHealth(channelID, provider, modelName)
	if err != nil {
		return nil, err
	}
	if health == nil {
		return nil, nil
	}

	health.SuccessCount++
	health.Provider = firstNonEmpty(provider, health.Provider)
	health.ModelName = firstNonEmpty(modelName, health.ModelName)
	health.P95Latency = updateLatencyEstimate(health.P95Latency, latencyMs)
	health.HealthScore = calculateChannelHealthScore(health)
	if health.CircuitState != ChannelHealthStateDisabled {
		health.CircuitState = healthyStateForScore(health.HealthScore, health.P95Latency)
		health.CooldownUntil = 0
	}
	health.UpdatedTime = common.GetTimestamp()

	if err := DB.Save(health).Error; err != nil {
		return nil, err
	}
	return health, nil
}

func RecordChannelHealthFailure(channelID int, provider string, modelName string, errorType string, message string, latencyMs int) (*ChannelHealth, error) {
	health, err := getOrCreateChannelHealth(channelID, provider, modelName)
	if err != nil {
		return nil, err
	}
	if health == nil {
		return nil, nil
	}

	normalizedErrorType := strings.TrimSpace(strings.ToUpper(errorType))
	now := common.GetTimestamp()
	health.FailureCount++
	health.Provider = firstNonEmpty(provider, health.Provider)
	health.ModelName = firstNonEmpty(modelName, health.ModelName)
	health.P95Latency = updateLatencyEstimate(health.P95Latency, latencyMs)
	health.LastErrorAt = now
	health.LastErrorType = normalizedErrorType
	health.LastErrorMessage = common.RedactSensitiveText(message)

	switch normalizedErrorType {
	case channelHealthErrorAuth:
		health.AuthErrorCount++
	case channelHealthErrorRateLimit:
		health.RateLimitCount++
	case channelHealthErrorServer:
		health.ServerErrorCount++
	case channelHealthErrorTimeout:
		health.TimeoutCount++
	}

	applyChannelHealthTransition(health, now)
	health.UpdatedTime = now

	if err := DB.Save(health).Error; err != nil {
		return nil, err
	}
	return health, nil
}

func applyChannelHealthTransition(health *ChannelHealth, now int64) {
	health.HealthScore = calculateChannelHealthScore(health)
	switch {
	case health.AuthErrorCount >= 5:
		health.CircuitState = ChannelHealthStateDisabled
		health.CooldownUntil = 0
	case health.RateLimitCount >= 3:
		health.CircuitState = ChannelHealthStateCooldown
		health.CooldownUntil = now + int64((60 * time.Second).Seconds())
	case health.ServerErrorCount >= 5 || health.TimeoutCount >= 5:
		health.CircuitState = ChannelHealthStateOpenCircuit
		health.CooldownUntil = now + int64((60 * time.Second).Seconds())
	default:
		health.CircuitState = healthyStateForScore(health.HealthScore, health.P95Latency)
	}
}

func calculateChannelHealthScore(health *ChannelHealth) int {
	score := 100
	score -= int(health.FailureCount * 8)
	score -= int(health.RateLimitCount * 4)
	score -= int(health.ServerErrorCount * 6)
	score -= int(health.TimeoutCount * 6)
	score -= int(health.AuthErrorCount * 12)
	if health.P95Latency > 10_000 {
		score -= 10
	}
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func healthyStateForScore(score int, p95Latency int) string {
	if score < 90 || p95Latency > 10_000 {
		return ChannelHealthStateDegraded
	}
	return ChannelHealthStateHealthy
}

func updateLatencyEstimate(existing int, next int) int {
	if next <= 0 {
		return existing
	}
	if existing <= 0 || next > existing {
		return next
	}
	return existing
}

func firstNonEmpty(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value != "" {
		return value
	}
	return fallback
}
