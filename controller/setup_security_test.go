package controller

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupControllerSetupTestDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	gin.SetMode(gin.TestMode)
	constant.Setup = false
	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.DB = db
	model.LOG_DB = db
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.Option{}, &model.Setup{}))
	model.InitOptionMap()

	t.Cleanup(func() {
		constant.Setup = false
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func performSetupRequest(body string, token string) *httptest.ResponseRecorder {
	router := gin.New()
	router.POST("/setup", PostSetup)
	req := httptest.NewRequest(http.MethodPost, "/setup", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("X-Setup-Token", token)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func TestPostSetupRequiresSetupTokenWhenConfigured(t *testing.T) {
	setupControllerSetupTestDB(t)
	t.Setenv("NEW_API_SETUP_TOKEN", "expected-token")
	t.Setenv("NEW_API_SECURITY_MODE", "")

	recorder := performSetupRequest(`{
		"username":"root",
		"password":"strong-password",
		"confirmPassword":"strong-password"
	}`, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "setup token")
	require.False(t, model.RootUserExists())
}

func TestPostSetupAcceptsSetupTokenHeader(t *testing.T) {
	setupControllerSetupTestDB(t)
	t.Setenv("NEW_API_SETUP_TOKEN", "expected-token")
	t.Setenv("NEW_API_SECURITY_MODE", "")

	recorder := performSetupRequest(`{
		"username":"root",
		"password":"strong-password",
		"confirmPassword":"strong-password"
	}`, "expected-token")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), `"success":true`)
	require.True(t, model.RootUserExists())
}

func TestPostSetupRequiresConfiguredSetupTokenInProductionSecurityMode(t *testing.T) {
	setupControllerSetupTestDB(t)
	t.Setenv("NEW_API_SETUP_TOKEN", "")
	t.Setenv("NEW_API_SECURITY_MODE", "production")

	recorder := performSetupRequest(`{
		"username":"root",
		"password":"strong-password",
		"confirmPassword":"strong-password"
	}`, "")

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Contains(t, recorder.Body.String(), "setup token")
	require.False(t, model.RootUserExists())
}
