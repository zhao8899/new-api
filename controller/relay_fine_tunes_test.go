package controller

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type relayFineTuneListResponse struct {
	Object string                    `json:"object"`
	Data   []model.OpenAIFineTuneDTO `json:"data"`
}

type relayFineTuneEventListResponse struct {
	Object string                         `json:"object"`
	Data   []model.OpenAIFineTuneEventDTO `json:"data"`
}

func setupRelayFineTuneTestDB(t *testing.T) {
	t.Helper()

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.OpenAIFile{}, &model.OpenAIFineTune{}, &model.OpenAIFineTuneEvent{}))
}

func TestRelayFineTuneLifecycle(t *testing.T) {
	setupRelayFineTuneTestDB(t)
	gin.SetMode(gin.TestMode)

	trainingFile, err := model.CreateOpenAIFile(1001, 2002, "training.jsonl", "fine-tune", []byte("{\"messages\":[]}\n"))
	require.NoError(t, err)

	createRecorder := httptest.NewRecorder()
	createCtx, _ := gin.CreateTestContext(createRecorder)
	createCtx.Request = httptest.NewRequest(http.MethodPost, "/v1/fine-tunes", bytes.NewBufferString(fmt.Sprintf(`{"training_file":"%s","model":"gpt-4o-mini"}`, trainingFile.Id)))
	createCtx.Request.Header.Set("Content-Type", "application/json")
	createCtx.Set("id", 1001)
	createCtx.Set("token_id", 2002)

	CreateRelayFineTune(createCtx)

	require.Equal(t, http.StatusOK, createRecorder.Code)
	var created model.OpenAIFineTuneDTO
	require.NoError(t, common.Unmarshal(createRecorder.Body.Bytes(), &created))
	require.True(t, created.ID != "")
	require.Equal(t, "fine-tune", created.Object)
	require.Equal(t, trainingFile.Id, created.TrainingFile)
	require.Equal(t, "gpt-4o-mini", created.Model)
	require.Equal(t, "pending", created.Status)

	listRecorder := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRecorder)
	listCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/fine-tunes", nil)
	listCtx.Set("id", 1001)

	ListRelayFineTunes(listCtx)

	require.Equal(t, http.StatusOK, listRecorder.Code)
	var listResp relayFineTuneListResponse
	require.NoError(t, common.Unmarshal(listRecorder.Body.Bytes(), &listResp))
	require.Equal(t, "list", listResp.Object)
	require.Len(t, listResp.Data, 1)
	require.Equal(t, created.ID, listResp.Data[0].ID)

	getRecorder := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRecorder)
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/fine-tunes/"+created.ID, nil)
	getCtx.Params = gin.Params{{Key: "id", Value: created.ID}}
	getCtx.Set("id", 1001)

	RetrieveRelayFineTune(getCtx)

	require.Equal(t, http.StatusOK, getRecorder.Code)
	var retrieved model.OpenAIFineTuneDTO
	require.NoError(t, common.Unmarshal(getRecorder.Body.Bytes(), &retrieved))
	require.Equal(t, created.ID, retrieved.ID)

	eventRecorder := httptest.NewRecorder()
	eventCtx, _ := gin.CreateTestContext(eventRecorder)
	eventCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/fine-tunes/"+created.ID+"/events", nil)
	eventCtx.Params = gin.Params{{Key: "id", Value: created.ID}}
	eventCtx.Set("id", 1001)

	ListRelayFineTuneEvents(eventCtx)

	require.Equal(t, http.StatusOK, eventRecorder.Code)
	var eventResp relayFineTuneEventListResponse
	require.NoError(t, common.Unmarshal(eventRecorder.Body.Bytes(), &eventResp))
	require.Equal(t, "list", eventResp.Object)
	require.Len(t, eventResp.Data, 1)
	require.Equal(t, created.ID, eventResp.Data[0].FineTuneID)
	require.Equal(t, "Fine-tune job created", eventResp.Data[0].Message)
	require.LessOrEqual(t, eventResp.Data[0].CreatedAt, common.GetTimestamp())
	require.GreaterOrEqual(t, eventResp.Data[0].CreatedAt, created.CreatedAt)

	cancelRecorder := httptest.NewRecorder()
	cancelCtx, _ := gin.CreateTestContext(cancelRecorder)
	cancelCtx.Request = httptest.NewRequest(http.MethodPost, "/v1/fine-tunes/"+created.ID+"/cancel", nil)
	cancelCtx.Params = gin.Params{{Key: "id", Value: created.ID}}
	cancelCtx.Set("id", 1001)

	CancelRelayFineTune(cancelCtx)

	require.Equal(t, http.StatusOK, cancelRecorder.Code)
	var cancelled model.OpenAIFineTuneDTO
	require.NoError(t, common.Unmarshal(cancelRecorder.Body.Bytes(), &cancelled))
	require.Equal(t, created.ID, cancelled.ID)
	require.Equal(t, "cancelled", cancelled.Status)

	eventAfterCancelRecorder := httptest.NewRecorder()
	eventAfterCancelCtx, _ := gin.CreateTestContext(eventAfterCancelRecorder)
	eventAfterCancelCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/fine-tunes/"+created.ID+"/events", nil)
	eventAfterCancelCtx.Params = gin.Params{{Key: "id", Value: created.ID}}
	eventAfterCancelCtx.Set("id", 1001)

	ListRelayFineTuneEvents(eventAfterCancelCtx)

	require.Equal(t, http.StatusOK, eventAfterCancelRecorder.Code)
	var eventAfterCancelResp relayFineTuneEventListResponse
	require.NoError(t, common.Unmarshal(eventAfterCancelRecorder.Body.Bytes(), &eventAfterCancelResp))
	require.Len(t, eventAfterCancelResp.Data, 2)
	require.Equal(t, "Fine-tune job cancelled", eventAfterCancelResp.Data[0].Message)
}

func TestRelayFineTuneRequiresTrainingFileOwnership(t *testing.T) {
	setupRelayFineTuneTestDB(t)
	gin.SetMode(gin.TestMode)

	trainingFile, err := model.CreateOpenAIFile(1001, 2002, "training.jsonl", "fine-tune", []byte("payload"))
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/v1/fine-tunes", bytes.NewBufferString(fmt.Sprintf(`{"training_file":"%s","model":"gpt-4o-mini"}`, trainingFile.Id)))
	ctx.Request.Header.Set("Content-Type", "application/json")
	ctx.Set("id", 9999)

	CreateRelayFineTune(ctx)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "training_file_not_found")
}
