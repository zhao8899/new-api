package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupChannelHealthControllerTestDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalOptionMap := common.OptionMap
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.ChannelHealth{}, &model.Option{}))
	common.OptionMapRWMutex.Lock()
	common.OptionMap = map[string]string{
		model.ChannelHealthCircuitModeOption: "observe",
	}
	common.OptionMapRWMutex.Unlock()

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.OptionMapRWMutex.Lock()
		common.OptionMap = originalOptionMap
		common.OptionMapRWMutex.Unlock()
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestGetChannelHealthRecordsFiltersAndPaginates(t *testing.T) {
	setupChannelHealthControllerTestDB(t)
	gin.SetMode(gin.TestMode)

	for i := 0; i < 3; i++ {
		_, err := model.RecordChannelHealthFailure(801, "gemini", "gemini-2.5-pro", "RATE_LIMIT", "quota exceeded", 120)
		require.NoError(t, err)
	}
	_, err := model.RecordChannelHealthSuccess(802, "openai", "gpt-4.1", 90)
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/channel/health", GetChannelHealthRecords)

	req := httptest.NewRequest(http.MethodGet, "/api/channel/health?provider=gemini&circuit_state=cooldown&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body channelHealthListResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, 1, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, 801, body.Data.Items[0].ChannelID)
}

func TestUpdateChannelHealthModePersistsOption(t *testing.T) {
	setupChannelHealthControllerTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/channel/health/mode", GetChannelHealthMode)
	router.PUT("/api/channel/health/mode", UpdateChannelHealthMode)

	req := httptest.NewRequest(http.MethodPut, "/api/channel/health/mode", bytes.NewBufferString(`{"mode":"enforce"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var updateBody channelHealthModeResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &updateBody))
	require.True(t, updateBody.Success)
	require.Equal(t, "enforce", updateBody.Data.Mode)
	require.Equal(t, "enforce", model.ChannelHealthCircuitMode())

	req = httptest.NewRequest(http.MethodGet, "/api/channel/health/mode", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var getBody channelHealthModeResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &getBody))
	require.True(t, getBody.Success)
	require.Equal(t, "enforce", getBody.Data.Mode)
}

func TestUpdateChannelHealthModeRejectsInvalidMode(t *testing.T) {
	setupChannelHealthControllerTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.PUT("/api/channel/health/mode", UpdateChannelHealthMode)

	req := httptest.NewRequest(http.MethodPut, "/api/channel/health/mode", bytes.NewBufferString(`{"mode":"block"}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body channelHealthModeResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Success)
}

type channelHealthListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Page     int                    `json:"page"`
		PageSize int                    `json:"page_size"`
		Total    int                    `json:"total"`
		Items    []*model.ChannelHealth `json:"items"`
	} `json:"data"`
}

type channelHealthModeResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    struct {
		Mode string `json:"mode"`
	} `json:"data"`
}
