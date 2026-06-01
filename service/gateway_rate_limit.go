package service

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/pkg/ratelimit"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
)

const (
	gatewayRateLimitEnabledEnv   = "NEW_API_RPM_TPM_LIMIT_ENABLED"
	gatewayRateLimitRPMEnv       = "NEW_API_RPM_LIMIT"
	gatewayRateLimitTPMEnv       = "NEW_API_TPM_LIMIT"
	gatewayRateLimitWindowEnv    = "NEW_API_RPM_TPM_LIMIT_WINDOW_SECONDS"
	gatewayRateLimitScopesEnv    = "NEW_API_RPM_TPM_LIMIT_SCOPES"
	gatewayRateLimitModeEnv      = "NEW_API_RPM_TPM_LIMIT_MODE"
	gatewayRateLimitStoreModeEnv = "NEW_API_RPM_TPM_LIMIT_STORE_FAILURE_MODE"

	gatewayRateLimitModeEnforce   = "enforce"
	gatewayRateLimitModeObserve   = "observe"
	gatewayRateLimitStoreFailOpen = "fail_open"
)

var (
	gatewayRateLimitStoreMu       sync.RWMutex
	gatewayRateLimitStoreOverride ratelimit.Store
)

type GatewayRateLimitReservation struct {
	limiter ratelimit.Limiter
	results []ratelimit.Result
}

func SetGatewayRateLimitStoreForTest(store ratelimit.Store) func() {
	gatewayRateLimitStoreMu.Lock()
	previous := gatewayRateLimitStoreOverride
	gatewayRateLimitStoreOverride = store
	gatewayRateLimitStoreMu.Unlock()

	return func() {
		gatewayRateLimitStoreMu.Lock()
		gatewayRateLimitStoreOverride = previous
		gatewayRateLimitStoreMu.Unlock()
	}
}

func ReserveGatewayRateLimit(c *gin.Context, relayInfo *relaycommon.RelayInfo, estimatedTokens int) (*GatewayRateLimitReservation, *types.NewAPIError) {
	if !common.GetEnvOrDefaultBool(gatewayRateLimitEnabledEnv, false) {
		return nil, nil
	}
	if c == nil || relayInfo == nil {
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("gateway rate limit context is incomplete"), types.ErrorCodeInvalidRequest, http.StatusBadRequest, types.ErrOptionWithSkipRetry())
	}

	rpm := int64(common.GetEnvOrDefault(gatewayRateLimitRPMEnv, 0))
	tpm := int64(common.GetEnvOrDefault(gatewayRateLimitTPMEnv, 0))
	if rpm <= 0 && tpm <= 0 {
		return nil, nil
	}

	store := gatewayRateLimitStore()
	if store == nil {
		if gatewayRateLimitStoreFailureMode() == gatewayRateLimitStoreFailOpen {
			c.Header("X-NewAPI-RateLimit-Store-Failure", gatewayRateLimitStoreFailOpen)
			return nil, nil
		}
		return nil, types.NewErrorWithStatusCode(fmt.Errorf("gateway rate limit requires redis or an injected store"), types.ErrorCodeRateLimitExceeded, http.StatusInternalServerError, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
	}

	limits := ratelimit.Limits{
		RPM:    rpm,
		TPM:    tpm,
		Window: gatewayRateLimitWindow(),
	}
	limiter := ratelimit.NewLimiter(store)
	reservation := &GatewayRateLimitReservation{
		limiter: limiter,
		results: make([]ratelimit.Result, 0, len(gatewayRateLimitScopes())),
	}

	for _, scope := range gatewayRateLimitScopes() {
		scopeKey := gatewayRateLimitScopeKey(scope, relayInfo, c)
		if scopeKey == "" {
			continue
		}
		result, err := limiter.Reserve(c.Request.Context(), ratelimit.Reservation{
			Provider:        gatewayRateLimitProvider(c),
			Model:           relayInfo.OriginModelName,
			ChannelID:       gatewayRateLimitChannelID(c, relayInfo),
			Scope:           scope,
			ScopeKey:        scopeKey,
			EstimatedTokens: estimatedTokens,
		}, limits)
		if err != nil {
			_ = reservation.Release(c.Request.Context(), 0)
			if gatewayRateLimitStoreFailureMode() == gatewayRateLimitStoreFailOpen {
				c.Header("X-NewAPI-RateLimit-Store-Failure", gatewayRateLimitStoreFailOpen)
				return nil, nil
			}
			return nil, types.NewErrorWithStatusCode(fmt.Errorf("gateway rate limit reserve failed: %w", err), types.ErrorCodeRateLimitExceeded, http.StatusInternalServerError, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		if !result.Allowed {
			if gatewayRateLimitMode() == gatewayRateLimitModeObserve {
				c.Header("X-NewAPI-RateLimit-Observed", "true")
				_ = observeGatewayRateLimitExceed(c.Request.Context(), store, result, limits.Window)
				reservation.results = append(reservation.results, result)
				continue
			}
			_ = reservation.Release(c.Request.Context(), 0)
			c.Header("Retry-After", strconv.Itoa(retryAfterSeconds(result.RetryAfter, limits.Window)))
			return nil, types.NewErrorWithStatusCode(fmt.Errorf("gateway rate limit exceeded: %s", result.Reason), types.ErrorCodeRateLimitExceeded, http.StatusTooManyRequests, types.ErrOptionWithSkipRetry(), types.ErrOptionWithNoRecordErrorLog())
		}
		reservation.results = append(reservation.results, result)
	}

	if len(reservation.results) == 0 {
		return nil, nil
	}
	return reservation, nil
}

func (r *GatewayRateLimitReservation) Release(ctx context.Context, actualTokens int) error {
	if r == nil {
		return nil
	}
	for _, result := range r.results {
		if err := r.limiter.Release(ctx, result, actualTokens); err != nil {
			return err
		}
	}
	return nil
}

func gatewayRateLimitStore() ratelimit.Store {
	gatewayRateLimitStoreMu.RLock()
	override := gatewayRateLimitStoreOverride
	gatewayRateLimitStoreMu.RUnlock()
	if override != nil {
		return override
	}
	if common.RedisEnabled && common.RDB != nil {
		return ratelimit.NewRedisStore(common.RDB)
	}
	return nil
}

func gatewayRateLimitWindow() time.Duration {
	seconds := common.GetEnvOrDefault(gatewayRateLimitWindowEnv, 60)
	if seconds <= 0 {
		seconds = 60
	}
	return time.Duration(seconds) * time.Second
}

func gatewayRateLimitMode() string {
	mode := strings.TrimSpace(strings.ToLower(common.GetEnvOrDefaultString(gatewayRateLimitModeEnv, gatewayRateLimitModeEnforce)))
	if mode == gatewayRateLimitModeObserve {
		return gatewayRateLimitModeObserve
	}
	return gatewayRateLimitModeEnforce
}

func gatewayRateLimitStoreFailureMode() string {
	mode := strings.TrimSpace(strings.ToLower(common.GetEnvOrDefaultString(gatewayRateLimitStoreModeEnv, "")))
	if mode == gatewayRateLimitStoreFailOpen {
		return gatewayRateLimitStoreFailOpen
	}
	return "fail_closed"
}

func gatewayRateLimitScopes() []string {
	raw := common.GetEnvOrDefaultString(gatewayRateLimitScopesEnv, strings.Join([]string{
		ratelimit.ScopeProvider,
		ratelimit.ScopeChannel,
		ratelimit.ScopeModel,
		ratelimit.ScopeGroup,
		ratelimit.ScopeToken,
		ratelimit.ScopeUser,
	}, ","))
	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(strings.ToLower(part))
		switch scope {
		case ratelimit.ScopeProvider, ratelimit.ScopeChannel, ratelimit.ScopeModel, ratelimit.ScopeGroup, ratelimit.ScopeToken, ratelimit.ScopeUser:
			scopes = append(scopes, scope)
		}
	}
	return scopes
}

func observeGatewayRateLimitExceed(ctx context.Context, store ratelimit.Store, result ratelimit.Result, window time.Duration) error {
	if store == nil {
		return nil
	}
	if result.RPMKey != "" {
		if _, err := store.Add(ctx, result.RPMKey, 1, window); err != nil {
			return err
		}
	}
	if result.TPMKey != "" && result.EstimatedTokens > 0 {
		if _, err := store.Add(ctx, result.TPMKey, int64(result.EstimatedTokens), window); err != nil {
			return err
		}
	}
	return nil
}

func gatewayRateLimitScopeKey(scope string, relayInfo *relaycommon.RelayInfo, c *gin.Context) string {
	switch scope {
	case ratelimit.ScopeProvider:
		return gatewayRateLimitProvider(c)
	case ratelimit.ScopeChannel:
		channelID := gatewayRateLimitChannelID(c, relayInfo)
		if channelID <= 0 {
			return ""
		}
		return strconv.Itoa(channelID)
	case ratelimit.ScopeModel:
		return relayInfo.OriginModelName
	case ratelimit.ScopeGroup:
		return relayInfo.UsingGroup
	case ratelimit.ScopeToken:
		if relayInfo.TokenId <= 0 {
			return ""
		}
		return strconv.Itoa(relayInfo.TokenId)
	case ratelimit.ScopeUser:
		if relayInfo.UserId <= 0 {
			return ""
		}
		return strconv.Itoa(relayInfo.UserId)
	default:
		return ""
	}
}

func gatewayRateLimitProvider(c *gin.Context) string {
	channelType := common.GetContextKeyInt(c, constant.ContextKeyChannelType)
	return model.GetDefaultProviderMetadataByChannelType(channelType).Provider
}

func gatewayRateLimitChannelID(c *gin.Context, relayInfo *relaycommon.RelayInfo) int {
	channelID := common.GetContextKeyInt(c, constant.ContextKeyChannelId)
	if channelID > 0 {
		return channelID
	}
	if relayInfo != nil && relayInfo.ChannelMeta != nil {
		return relayInfo.ChannelMeta.ChannelId
	}
	return 0
}

func retryAfterSeconds(retryAfter time.Duration, fallback time.Duration) int {
	if retryAfter <= 0 {
		retryAfter = fallback
	}
	if retryAfter <= 0 {
		retryAfter = time.Minute
	}
	seconds := int((retryAfter + time.Second - 1) / time.Second)
	if seconds < 1 {
		return 1
	}
	return seconds
}
