package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ScopeProvider = "provider"
	ScopeChannel  = "channel"
	ScopeModel    = "model"
	ScopeGroup    = "group"
	ScopeToken    = "token"
	ScopeUser     = "user"

	LimitReasonRPM = "RPM_LIMIT"
	LimitReasonTPM = "TPM_LIMIT"
)

type CounterState struct {
	Value int64
	TTL   time.Duration
}

type Store interface {
	Add(ctx context.Context, key string, delta int64, window time.Duration) (CounterState, error)
}

type Limiter struct {
	store Store
}

type Reservation struct {
	Provider        string
	Model           string
	ChannelID       int
	Scope           string
	ScopeKey        string
	EstimatedTokens int
}

type Limits struct {
	RPM    int64
	TPM    int64
	Window time.Duration
}

type Result struct {
	Allowed         bool
	Reason          string
	RetryAfter      time.Duration
	RPMKey          string
	TPMKey          string
	RPMUsed         int64
	TPMUsed         int64
	EstimatedTokens int
}

func NewLimiter(store Store) Limiter {
	return Limiter{store: store}
}

func (l Limiter) Reserve(ctx context.Context, reservation Reservation, limits Limits) (Result, error) {
	if l.store == nil {
		return Result{}, errors.New("rate limit store is required")
	}
	window := limits.Window
	if window <= 0 {
		window = time.Minute
	}
	result := Result{
		Allowed:         true,
		RPMKey:          buildKey(reservation, "rpm"),
		TPMKey:          buildKey(reservation, "tpm"),
		EstimatedTokens: reservation.EstimatedTokens,
	}

	rpmState, err := l.store.Add(ctx, result.RPMKey, 1, window)
	if err != nil {
		return Result{}, err
	}
	result.RPMUsed = rpmState.Value
	if limits.RPM > 0 && rpmState.Value > limits.RPM {
		_, _ = l.store.Add(ctx, result.RPMKey, -1, window)
		result.Allowed = false
		result.Reason = LimitReasonRPM
		result.RetryAfter = retryAfter(rpmState.TTL, window)
		result.RPMUsed = limits.RPM
		return result, nil
	}

	tpmDelta := int64(reservation.EstimatedTokens)
	if tpmDelta < 0 {
		tpmDelta = 0
	}
	tpmState, err := l.store.Add(ctx, result.TPMKey, tpmDelta, window)
	if err != nil {
		_, _ = l.store.Add(ctx, result.RPMKey, -1, window)
		return Result{}, err
	}
	result.TPMUsed = tpmState.Value
	if limits.TPM > 0 && tpmState.Value > limits.TPM {
		_, _ = l.store.Add(ctx, result.RPMKey, -1, window)
		_, _ = l.store.Add(ctx, result.TPMKey, -tpmDelta, window)
		result.Allowed = false
		result.Reason = LimitReasonTPM
		result.RetryAfter = retryAfter(tpmState.TTL, window)
		result.RPMUsed = maxInt64(0, result.RPMUsed-1)
		result.TPMUsed = maxInt64(0, result.TPMUsed-tpmDelta)
		return result, nil
	}

	return result, nil
}

func (l Limiter) Release(ctx context.Context, result Result, actualTokens int) error {
	if l.store == nil {
		return errors.New("rate limit store is required")
	}
	if result.RPMKey != "" {
		if _, err := l.store.Add(ctx, result.RPMKey, -1, time.Minute); err != nil {
			return err
		}
	}
	tpmDelta := result.EstimatedTokens - actualTokens
	if result.TPMKey != "" && tpmDelta > 0 {
		if _, err := l.store.Add(ctx, result.TPMKey, -int64(tpmDelta), time.Minute); err != nil {
			return err
		}
	}
	return nil
}

func buildKey(reservation Reservation, bucket string) string {
	provider := cleanPart(reservation.Provider)
	model := cleanPart(reservation.Model)
	scope := cleanPart(reservation.Scope)
	scopeKey := cleanPart(reservation.ScopeKey)
	if provider == "" {
		provider = "unknown_provider"
	}
	if model == "" {
		model = "unknown_model"
	}
	if scope == "" {
		scope = ScopeProvider
	}
	if scopeKey == "" {
		scopeKey = provider
	}
	return fmt.Sprintf("rate_limit:%s:%s:%d:%s:%s:%s", provider, model, reservation.ChannelID, scope, scopeKey, bucket)
}

func cleanPart(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	value = strings.ReplaceAll(value, ":", "_")
	value = strings.ReplaceAll(value, " ", "_")
	return value
}

func retryAfter(ttl time.Duration, fallback time.Duration) time.Duration {
	if ttl > 0 {
		return ttl
	}
	if fallback > 0 {
		return fallback
	}
	return time.Minute
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
