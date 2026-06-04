package controller

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupEpayNotifyTestDB(t *testing.T) {
	t.Helper()

	originalDB := model.DB
	originalLogDB := model.LOG_DB
	originalUsingSQLite := common.UsingSQLite
	originalUsingMySQL := common.UsingMySQL
	originalUsingPostgreSQL := common.UsingPostgreSQL
	originalRedisEnabled := common.RedisEnabled
	originalPayAddress := operation_setting.PayAddress
	originalEpayID := operation_setting.EpayId
	originalEpayKey := operation_setting.EpayKey
	originalPayMethods := operation_setting.PayMethods
	paymentSetting := operation_setting.GetPaymentSetting()
	originalConfirmed := paymentSetting.ComplianceConfirmed
	originalTermsVersion := paymentSetting.ComplianceTermsVersion

	common.UsingSQLite = true
	common.UsingMySQL = false
	common.UsingPostgreSQL = false
	common.RedisEnabled = false
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.TopUp{}, &model.Log{}))
	model.DB = db
	model.LOG_DB = db

	operation_setting.PayAddress = "https://pay.example.com"
	operation_setting.EpayId = "10001"
	operation_setting.EpayKey = "epay-secret"
	operation_setting.PayMethods = []map[string]string{
		{"name": "支付宝", "type": "alipay", "color": "#1677FF"},
		{"name": "微信支付", "type": "wxpay", "color": "#07C160"},
	}
	paymentSetting.ComplianceConfirmed = true
	paymentSetting.ComplianceTermsVersion = operation_setting.CurrentComplianceTermsVersion

	t.Cleanup(func() {
		model.DB = originalDB
		model.LOG_DB = originalLogDB
		common.UsingSQLite = originalUsingSQLite
		common.UsingMySQL = originalUsingMySQL
		common.UsingPostgreSQL = originalUsingPostgreSQL
		common.RedisEnabled = originalRedisEnabled
		operation_setting.PayAddress = originalPayAddress
		operation_setting.EpayId = originalEpayID
		operation_setting.EpayKey = originalEpayKey
		operation_setting.PayMethods = originalPayMethods
		paymentSetting.ComplianceConfirmed = originalConfirmed
		paymentSetting.ComplianceTermsVersion = originalTermsVersion
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
}

func createEpayNotifyOrder(t *testing.T, tradeNo string, paymentMethod string) {
	t.Helper()
	require.NoError(t, model.DB.Create(&model.User{
		Id:       901,
		Username: "epay_user",
		Status:   common.UserStatusEnabled,
		Quota:    0,
	}).Error)
	require.NoError(t, (&model.TopUp{
		UserId:          901,
		Amount:          2,
		Money:           14.60,
		TradeNo:         tradeNo,
		PaymentMethod:   paymentMethod,
		PaymentProvider: model.PaymentProviderEpay,
		CreateTime:      time.Now().Unix(),
		Status:          common.TopUpStatusPending,
	}).Insert())
}

func signedEpayNotifyForm(tradeNo string, paymentMethod string) url.Values {
	params := map[string]string{
		"pid":          operation_setting.EpayId,
		"type":         paymentMethod,
		"out_trade_no": tradeNo,
		"trade_no":     "epay-" + tradeNo,
		"name":         "TUC2",
		"money":        "14.60",
		"trade_status": epay.StatusTradeSuccess,
	}
	signed := epay.GenerateParams(params, operation_setting.EpayKey)
	values := url.Values{}
	for key, value := range signed {
		values.Set(key, value)
	}
	return values
}

func postEpayNotify(t *testing.T, router *gin.Engine, values url.Values) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/user/epay/notify", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func getEpayTestUserQuota(t *testing.T) int {
	t.Helper()
	var user model.User
	require.NoError(t, model.DB.Select("quota").Where("id = ?", 901).First(&user).Error)
	return user.Quota
}

func TestEpayNotifyCompletesAlipayOrderAtomicallyAndIdempotently(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/user/epay/notify", EpayNotify)
	createEpayNotifyOrder(t, "USR901NOalipay", "alipay")

	values := signedEpayNotifyForm("USR901NOalipay", "alipay")
	w := postEpayNotify(t, router, values)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "success", w.Body.String())
	require.Equal(t, int(2*common.QuotaPerUnit), getEpayTestUserQuota(t))

	w = postEpayNotify(t, router, values)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "success", w.Body.String())
	require.Equal(t, int(2*common.QuotaPerUnit), getEpayTestUserQuota(t))
}

func TestEpayNotifyCompletesWechatPayOrder(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/user/epay/notify", EpayNotify)
	createEpayNotifyOrder(t, "USR901NOwxpay", "wxpay")

	w := postEpayNotify(t, router, signedEpayNotifyForm("USR901NOwxpay", "wxpay"))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "success", w.Body.String())
	require.Equal(t, int(2*common.QuotaPerUnit), getEpayTestUserQuota(t))
}

func TestEpayNotifyRejectsInvalidSignatureWithoutCreditingQuota(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/user/epay/notify", EpayNotify)
	createEpayNotifyOrder(t, "USR901NOfake", "alipay")

	values := signedEpayNotifyForm("USR901NOfake", "alipay")
	values.Set("sign", "invalid")
	w := postEpayNotify(t, router, values)
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "fail", w.Body.String())
	require.Equal(t, 0, getEpayTestUserQuota(t))
}

func TestEpayNotifyReturnsFailWhenSignedOrderCannotBeSettled(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/user/epay/notify", EpayNotify)

	w := postEpayNotify(t, router, signedEpayNotifyForm("USR901NOmissing", "alipay"))
	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "fail", w.Body.String())
}

func TestGetTopUpReconciliationSummaryReturnsGroupedRows(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, model.DB.Create(&model.TopUp{
		UserId:          901,
		Amount:          5,
		Money:           36.50,
		TradeNo:         "admin-reconcile-alipay",
		PaymentMethod:   "alipay",
		PaymentProvider: model.PaymentProviderEpay,
		CreateTime:      200,
		Status:          common.TopUpStatusSuccess,
	}).Error)
	require.NoError(t, model.DB.Create(&model.TopUp{
		UserId:          901,
		Amount:          7,
		Money:           51.10,
		TradeNo:         "admin-reconcile-wxpay",
		PaymentMethod:   "wxpay",
		PaymentProvider: model.PaymentProviderEpay,
		CreateTime:      210,
		Status:          common.TopUpStatusPending,
	}).Error)

	router := gin.New()
	router.GET("/api/user/topup/reconciliation", GetTopUpReconciliationSummary)
	req := httptest.NewRequest(http.MethodGet, "/api/user/topup/reconciliation?start_time=190&end_time=220&payment_provider=epay", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Success bool `json:"success"`
		Data    struct {
			StartTime int64                          `json:"start_time"`
			EndTime   int64                          `json:"end_time"`
			Items     []model.TopUpReconciliationRow `json:"items"`
		} `json:"data"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.True(t, body.Success)
	require.Equal(t, int64(190), body.Data.StartTime)
	require.Equal(t, int64(220), body.Data.EndTime)
	require.Len(t, body.Data.Items, 2)
	require.Equal(t, "alipay", body.Data.Items[0].PaymentMethod)
	require.Equal(t, int64(1), body.Data.Items[0].OrderCount)
	require.Equal(t, "wxpay", body.Data.Items[1].PaymentMethod)
	require.Equal(t, int64(1), body.Data.Items[1].OrderCount)
}

func TestGetTopUpReconciliationSummaryRejectsInvalidTimeRange(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/user/topup/reconciliation", GetTopUpReconciliationSummary)

	req := httptest.NewRequest(http.MethodGet, "/api/user/topup/reconciliation?start_time=220&end_time=190", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var body struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	require.NoError(t, common.Unmarshal(w.Body.Bytes(), &body))
	require.False(t, body.Success)
	require.Contains(t, body.Message, "end_time")
}

func TestExportTopUpReconciliationSummaryReturnsCsv(t *testing.T) {
	setupEpayNotifyTestDB(t)
	gin.SetMode(gin.TestMode)
	require.NoError(t, model.DB.Create(&model.TopUp{
		UserId:          901,
		Amount:          5,
		Money:           36.50,
		TradeNo:         "admin-reconcile-export",
		PaymentMethod:   "alipay",
		PaymentProvider: model.PaymentProviderEpay,
		CreateTime:      200,
		CompleteTime:    205,
		Status:          common.TopUpStatusSuccess,
	}).Error)

	router := gin.New()
	router.GET("/api/user/topup/reconciliation/export", ExportTopUpReconciliationSummary)
	req := httptest.NewRequest(http.MethodGet, "/api/user/topup/reconciliation/export?start_time=190&end_time=220", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Header().Get("Content-Type"), "text/csv")
	require.Contains(t, w.Header().Get("Content-Disposition"), "topup-reconciliation-190-220.csv")
	require.Contains(t, w.Body.String(), "payment_provider,payment_method,status,order_count")
	require.Contains(t, w.Body.String(), "epay,alipay,success,1")
}
