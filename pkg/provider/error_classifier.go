package provider

import (
	"context"
	"errors"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
)

func ClassifyHTTPStatus(provider string, statusCode int, message string) *ProviderError {
	err := &ProviderError{
		Provider:        strings.TrimSpace(strings.ToLower(provider)),
		StatusCode:      statusCode,
		MessageRedacted: common.RedactSensitiveText(message),
		ClassifiedAt:    time.Now(),
	}

	normalizedMessage := strings.ToLower(message)
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		err.Type = ErrorTypeAuth
		err.CircuitBreakerSignal = true
		err.RefundSuggested = true
	case statusCode == http.StatusTooManyRequests:
		err.Type = ErrorTypeRateLimit
		err.Retryable = true
		err.Switchable = true
		err.CircuitBreakerSignal = true
		err.RefundSuggested = true
	case statusCode == http.StatusBadRequest:
		if strings.Contains(normalizedMessage, "content policy") ||
			strings.Contains(normalizedMessage, "content_filter") ||
			strings.Contains(normalizedMessage, "safety") {
			err.Type = ErrorTypeContentFilter
			return err
		}
		err.Type = ErrorTypeBadRequest
		err.RefundSuggested = true
	case statusCode == http.StatusNotFound:
		err.Type = ErrorTypeModelNotFound
		err.Switchable = true
		err.CircuitBreakerSignal = true
		err.RefundSuggested = true
	case statusCode >= 500:
		err.Type = ErrorTypeServer
		err.Retryable = true
		err.Switchable = true
		err.CircuitBreakerSignal = true
		err.RefundSuggested = true
	default:
		err.Type = ErrorTypeBadRequest
		if statusCode >= 400 {
			err.RefundSuggested = true
		}
	}
	return err
}

func ClassifyError(provider string, err error) *ProviderError {
	if err == nil {
		return nil
	}
	providerErr := &ProviderError{
		Provider:             strings.TrimSpace(strings.ToLower(provider)),
		MessageRedacted:      common.RedactSensitiveText(err.Error()),
		Retryable:            true,
		Switchable:           true,
		CircuitBreakerSignal: true,
		RefundSuggested:      true,
		ClassifiedAt:         time.Now(),
	}

	var netErr net.Error
	if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &netErr) && netErr.Timeout()) {
		providerErr.Type = ErrorTypeTimeout
		return providerErr
	}

	providerErr.Type = ErrorTypeNetwork
	return providerErr
}
