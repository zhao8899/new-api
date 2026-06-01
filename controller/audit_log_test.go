package controller

import (
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

func setupAuditLogControllerTestDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.AuditEvent{}))

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func TestGetAuditEventsFiltersAndPaginates(t *testing.T) {
	setupAuditLogControllerTestDB(t)
	gin.SetMode(gin.TestMode)

	_, err := model.RecordAuditEvent(model.AuditEventParams{
		ActorID:      10,
		ActorRole:    common.RoleRootUser,
		Action:       "channel.update",
		ResourceType: "channel",
		ResourceID:   "42",
		RequestID:    "req-audit-controller",
		Result:       "success",
	})
	require.NoError(t, err)
	_, err = model.RecordAuditEvent(model.AuditEventParams{
		ActorID:      11,
		Action:       "token.view_key",
		ResourceType: "token",
		ResourceID:   "7",
		RequestID:    "req-other",
		Result:       "success",
	})
	require.NoError(t, err)

	router := gin.New()
	router.GET("/api/log/audit", GetAuditEvents)

	req := httptest.NewRequest(http.MethodGet, "/api/log/audit?request_id=req-audit-controller&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body auditListResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, 1, body.Data.Total)
	require.Len(t, body.Data.Items, 1)
	require.Equal(t, "channel.update", body.Data.Items[0].Action)
	require.Equal(t, "42", body.Data.Items[0].ResourceID)
}

func TestGetAuditEventsRejectsOversizedPageSize(t *testing.T) {
	setupAuditLogControllerTestDB(t)
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/log/audit", GetAuditEvents)

	req := httptest.NewRequest(http.MethodGet, "/api/log/audit?page_size=999", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body auditListResponse
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, 100, body.Data.PageSize)
}

type auditListResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Page     int                 `json:"page"`
		PageSize int                 `json:"page_size"`
		Total    int                 `json:"total"`
		Items    []*model.AuditEvent `json:"items"`
	} `json:"data"`
}
