package service

import (
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	providerpkg "github.com/QuantumNous/new-api/pkg/provider"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

func RecordChannelHealthFromRelayError(channelID int, provider string, modelName string, relayErr *types.NewAPIError, latencyMs int) error {
	if channelID <= 0 || relayErr == nil {
		return nil
	}
	providerErr := providerpkg.ClassifyHTTPStatus(provider, relayErr.StatusCode, relayErr.ErrorWithStatusCode())
	if providerErr == nil {
		return nil
	}
	_, err := model.RecordChannelHealthFailure(
		channelID,
		providerErr.Provider,
		modelName,
		providerErr.Type,
		providerErr.MessageRedacted,
		latencyMs,
	)
	return err
}

func RecordChannelHealthSuccess(channelID int, provider string, modelName string, latencyMs int) error {
	if channelID <= 0 {
		return nil
	}
	_, err := model.RecordChannelHealthSuccess(channelID, strings.TrimSpace(strings.ToLower(provider)), modelName, latencyMs)
	return err
}

func RecordChannelHealthSuccessFromContext(c *gin.Context) error {
	if c == nil {
		return nil
	}
	channelID := c.GetInt("channel_id")
	if channelID <= 0 {
		return nil
	}
	channelType := c.GetInt("channel_type")
	meta := model.GetDefaultProviderMetadataByChannelType(channelType)
	modelName := c.GetString("original_model")
	startTime := common.GetContextKeyTime(c, constant.ContextKeyRequestStartTime)
	latencyMs := 0
	if !startTime.IsZero() {
		latencyMs = int(time.Since(startTime).Milliseconds())
	}
	return RecordChannelHealthSuccess(channelID, meta.Provider, modelName, latencyMs)
}
