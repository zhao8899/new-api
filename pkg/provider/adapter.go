package provider

import (
	"context"
	"net/http"
	"time"
)

const (
	ErrorTypeAuth              = "AUTH_ERROR"
	ErrorTypeRateLimit         = "RATE_LIMIT"
	ErrorTypeServer            = "SERVER_ERROR"
	ErrorTypeTimeout           = "TIMEOUT"
	ErrorTypeBadRequest        = "BAD_REQUEST"
	ErrorTypeContentFilter     = "CONTENT_FILTER"
	ErrorTypeModelNotFound     = "MODEL_NOT_FOUND"
	ErrorTypeInsufficientQuota = "INSUFFICIENT_QUOTA"
	ErrorTypeNetwork           = "NETWORK_ERROR"
)

type Adapter interface {
	Name() string
	Protocol() string
	ValidateRequest(ctx context.Context, req any) error
	ConvertRequest(ctx context.Context, req any) (*http.Request, error)
	ParseResponse(ctx context.Context, resp *http.Response) (*UnifiedResponse, error)
	ParseError(ctx context.Context, resp *http.Response) (*ProviderError, error)
}

type UnifiedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Usage      any
}

type ProviderError struct {
	Type                 string
	Provider             string
	StatusCode           int
	UpstreamCode         string
	MessageRedacted      string
	Retryable            bool
	Switchable           bool
	CircuitBreakerSignal bool
	RefundSuggested      bool
	ClassifiedAt         time.Time
}
