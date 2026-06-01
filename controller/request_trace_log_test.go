package controller

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestGetRequestTracesFiltersAndPaginates(t *testing.T) {
	setupAuditLogControllerTestDB(t)
	require.NoError(t, model.LOG_DB.AutoMigrate(&model.RequestTrace{}))
	_, err := model.RecordRequestTrace(model.RequestTraceParams{
		RequestID:     "req-trace-api-1",
		UserID:        10,
		TokenID:       20,
		ExternalModel: "gpt-4.1",
		Provider:      "openai",
		ChannelID:     30,
		StatusCode:    200,
	})
	require.NoError(t, err)
	_, err = model.RecordRequestTrace(model.RequestTraceParams{
		RequestID:            "req-trace-api-2",
		UserID:               11,
		TokenID:              21,
		ExternalModel:        "claude-sonnet",
		Provider:             "anthropic",
		ChannelID:            31,
		StatusCode:           502,
		ErrorType:            "SERVER_ERROR",
		ErrorMessageRedacted: "failed with key sk-abcdef1234567890",
	})
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/log/request_trace", GetRequestTraces)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/log/request_trace?provider=anthropic&error_type=server_error&p=0&page_size=10", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool `json:"success"`
		Data    struct {
			Items []*model.RequestTrace `json:"items"`
			Total int                   `json:"total"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, 1, resp.Data.Total)
	require.Len(t, resp.Data.Items, 1)
	require.Equal(t, "req-trace-api-2", resp.Data.Items[0].RequestID)
	require.NotContains(t, resp.Data.Items[0].ErrorMessageRedacted, "sk-abcdef1234567890")
}

func TestGetRequestTraceByRequestID(t *testing.T) {
	setupAuditLogControllerTestDB(t)
	require.NoError(t, model.LOG_DB.AutoMigrate(&model.RequestTrace{}))
	_, err := model.RecordRequestTrace(model.RequestTraceParams{
		RequestID:     "req-trace-detail",
		UserID:        10,
		Provider:      "openai",
		ExternalModel: "gpt-4.1",
	})
	require.NoError(t, err)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/log/request_trace/:request_id", GetRequestTrace)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/log/request_trace/req-trace-detail", nil)
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp struct {
		Success bool                `json:"success"`
		Data    *model.RequestTrace `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &resp))
	require.True(t, resp.Success)
	require.Equal(t, "req-trace-detail", resp.Data.RequestID)
}
