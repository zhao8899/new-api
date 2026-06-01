package provider

const defaultMaxRetries = 1

type RetryBudgetOptions struct {
	RetryCount    int
	MaxRetries    int
	StreamStarted bool
	NonIdempotent bool
}

type RetryBudget struct {
	retryCount    int
	maxRetries    int
	streamStarted bool
	nonIdempotent bool
}

type RetryDecision struct {
	Retry          bool
	SwitchChannel  bool
	SameChannel    bool
	NextRetryCount int
	ErrorType      string
	Reason         string
}

func NewRetryBudget(options RetryBudgetOptions) RetryBudget {
	maxRetries := options.MaxRetries
	if maxRetries <= 0 {
		maxRetries = defaultMaxRetries
	}
	return RetryBudget{
		retryCount:    options.RetryCount,
		maxRetries:    maxRetries,
		streamStarted: options.StreamStarted,
		nonIdempotent: options.NonIdempotent,
	}
}

func (b RetryBudget) Decide(providerErr *ProviderError) RetryDecision {
	decision := RetryDecision{}
	if providerErr != nil {
		decision.ErrorType = providerErr.Type
	}

	if providerErr == nil {
		decision.Reason = "missing provider error"
		return decision
	}
	if b.streamStarted {
		decision.Reason = "stream already produced output"
		return decision
	}
	if b.nonIdempotent {
		decision.Reason = "request is not idempotent"
		return decision
	}
	if b.retryCount >= b.maxRetries {
		decision.Reason = "retry budget exhausted"
		return decision
	}

	switch providerErr.Type {
	case ErrorTypeAuth:
		decision.Reason = "auth errors are not retryable"
		return decision
	case ErrorTypeBadRequest:
		decision.Reason = "bad request errors are not retryable"
		return decision
	case ErrorTypeContentFilter:
		decision.Reason = "content filter errors are not retryable"
		return decision
	case ErrorTypeModelNotFound:
		decision.Reason = "model not found errors are not retryable"
		return decision
	}

	if !providerErr.Retryable {
		decision.Reason = "provider error is not retryable"
		return decision
	}

	decision.Retry = true
	decision.SwitchChannel = providerErr.Switchable
	decision.SameChannel = !providerErr.Switchable && providerErr.Type != ErrorTypeRateLimit
	decision.NextRetryCount = b.retryCount + 1
	if providerErr.Type == ErrorTypeRateLimit {
		decision.SwitchChannel = true
		decision.SameChannel = false
	}
	return decision
}
