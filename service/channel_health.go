package service

import (
	"strings"

	"github.com/QuantumNous/new-api/model"
	providerpkg "github.com/QuantumNous/new-api/pkg/provider"
	"github.com/QuantumNous/new-api/types"
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
