package service

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/pkg/ratelimit"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestReserveGatewayRateLimitDisabledByDefault(t *testing.T) {
	t.Setenv("NEW_API_RPM_TPM_LIMIT_ENABLED", "false")
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())

	reservation, apiErr := ReserveGatewayRateLimit(c, &relaycommon.RelayInfo{}, 100)

	require.Nil(t, reservation)
	require.Nil(t, apiErr)
}

func TestReserveGatewayRateLimitRejectsTPMAndSetsRetryAfter(t *testing.T) {
	t.Setenv("NEW_API_RPM_TPM_LIMIT_ENABLED", "true")
	t.Setenv("NEW_API_RPM_LIMIT", "10")
	t.Setenv("NEW_API_TPM_LIMIT", "100")
	t.Setenv("NEW_API_RPM_TPM_LIMIT_SCOPES", "channel")
	restore := SetGatewayRateLimitStoreForTest(newGatewayRateLimitMemoryStore(time.Minute))
	defer restore()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyChannelId, 10)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)

	reservation, apiErr := ReserveGatewayRateLimit(c, &relaycommon.RelayInfo{
		OriginModelName: "gpt-4.1",
		TokenId:         20,
		UserId:          30,
		UsingGroup:      "default",
	}, 200)

	require.Nil(t, reservation)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	require.Equal(t, "60", w.Header().Get("Retry-After"))
}

func TestReserveGatewayRateLimitUsesMultipleScopes(t *testing.T) {
	t.Setenv("NEW_API_RPM_TPM_LIMIT_ENABLED", "true")
	t.Setenv("NEW_API_RPM_LIMIT", "1")
	t.Setenv("NEW_API_TPM_LIMIT", "1000")
	t.Setenv("NEW_API_RPM_TPM_LIMIT_SCOPES", "token,user")
	store := newGatewayRateLimitMemoryStore(time.Minute)
	restore := SetGatewayRateLimitStoreForTest(store)
	defer restore()

	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyChannelId, 10)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4.1",
		TokenId:         20,
		UserId:          30,
		UsingGroup:      "default",
	}

	first, apiErr := ReserveGatewayRateLimit(c, info, 100)
	require.Nil(t, apiErr)
	require.NotNil(t, first)
	second, apiErr := ReserveGatewayRateLimit(c, info, 100)
	require.Nil(t, second)
	require.NotNil(t, apiErr)
	require.Equal(t, http.StatusTooManyRequests, apiErr.StatusCode)
	require.Equal(t, int64(1), store.value("rate_limit:openai:gpt-4.1:10:token:20:rpm"))
	require.Equal(t, int64(1), store.value("rate_limit:openai:gpt-4.1:10:user:30:rpm"))
}

func TestReserveGatewayRateLimitObserveModeDoesNotReject(t *testing.T) {
	t.Setenv("NEW_API_RPM_TPM_LIMIT_ENABLED", "true")
	t.Setenv("NEW_API_RPM_LIMIT", "1")
	t.Setenv("NEW_API_TPM_LIMIT", "1000")
	t.Setenv("NEW_API_RPM_TPM_LIMIT_SCOPES", "token")
	t.Setenv("NEW_API_RPM_TPM_LIMIT_MODE", "observe")
	store := newGatewayRateLimitMemoryStore(time.Minute)
	restore := SetGatewayRateLimitStoreForTest(store)
	defer restore()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
	common.SetContextKey(c, constant.ContextKeyChannelId, 10)
	common.SetContextKey(c, constant.ContextKeyChannelType, constant.ChannelTypeOpenAI)
	info := &relaycommon.RelayInfo{
		OriginModelName: "gpt-4.1",
		TokenId:         20,
	}

	first, apiErr := ReserveGatewayRateLimit(c, info, 100)
	require.Nil(t, apiErr)
	require.NotNil(t, first)
	second, apiErr := ReserveGatewayRateLimit(c, info, 100)
	require.Nil(t, apiErr)
	require.NotNil(t, second)
	require.Equal(t, "true", w.Header().Get("X-NewAPI-RateLimit-Observed"))
	require.Equal(t, int64(2), store.value("rate_limit:openai:gpt-4.1:10:token:20:rpm"))
}

func TestReserveGatewayRateLimitStoreFailOpen(t *testing.T) {
	t.Setenv("NEW_API_RPM_TPM_LIMIT_ENABLED", "true")
	t.Setenv("NEW_API_RPM_LIMIT", "1")
	t.Setenv("NEW_API_RPM_TPM_LIMIT_STORE_FAILURE_MODE", "fail_open")
	restore := SetGatewayRateLimitStoreForTest(nil)
	defer restore()
	originalRedisEnabled := common.RedisEnabled
	originalRedis := common.RDB
	common.RedisEnabled = false
	common.RDB = nil
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		common.RDB = originalRedis
	})

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)

	reservation, apiErr := ReserveGatewayRateLimit(c, &relaycommon.RelayInfo{OriginModelName: "gpt-4.1"}, 100)

	require.Nil(t, reservation)
	require.Nil(t, apiErr)
	require.Equal(t, "fail_open", w.Header().Get("X-NewAPI-RateLimit-Store-Failure"))
}

type gatewayRateLimitMemoryStore struct {
	mu     sync.Mutex
	values map[string]int64
	ttl    time.Duration
}

func newGatewayRateLimitMemoryStore(ttl time.Duration) *gatewayRateLimitMemoryStore {
	return &gatewayRateLimitMemoryStore{
		values: make(map[string]int64),
		ttl:    ttl,
	}
}

func (s *gatewayRateLimitMemoryStore) Add(ctx context.Context, key string, delta int64, window time.Duration) (ratelimit.CounterState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] += delta
	if s.values[key] < 0 {
		s.values[key] = 0
	}
	return ratelimit.CounterState{Value: s.values[key], TTL: s.ttl}, nil
}

func (s *gatewayRateLimitMemoryStore) value(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key]
}
