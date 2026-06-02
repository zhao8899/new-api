package controller

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type relayFileListResponse struct {
	Object string                  `json:"object"`
	Data   []model.OpenAIFileBrief `json:"data"`
}

type relayFileDeleteResponse struct {
	Id      string `json:"id"`
	Object  string `json:"object"`
	Deleted bool   `json:"deleted"`
}

func setupRelayFilesTestDB(t *testing.T) {
	t.Helper()

	db := setupModelListControllerTestDB(t)
	require.NoError(t, db.AutoMigrate(&model.OpenAIFile{}))
}

func newRelayFileUploadRequest(t *testing.T, filename, purpose string, content []byte) *http.Request {
	t.Helper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)
	_, err = part.Write(content)
	require.NoError(t, err)

	err = writer.WriteField("purpose", purpose)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPost, "/v1/files", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestRelayFilesLifecycle(t *testing.T) {
	setupRelayFilesTestDB(t)
	gin.SetMode(gin.TestMode)

	uploadRecorder := httptest.NewRecorder()
	uploadCtx, _ := gin.CreateTestContext(uploadRecorder)
	uploadCtx.Request = newRelayFileUploadRequest(t, "training.jsonl", "fine-tune", []byte("{\"messages\":[]}\n"))
	uploadCtx.Set("id", 1001)
	uploadCtx.Set("token_id", 2002)

	CreateRelayFile(uploadCtx)

	require.Equal(t, http.StatusOK, uploadRecorder.Code)

	var uploaded model.OpenAIFileBrief
	require.NoError(t, common.Unmarshal(uploadRecorder.Body.Bytes(), &uploaded))
	require.True(t, strings.HasPrefix(uploaded.Id, "file-"))
	require.Equal(t, "file", uploaded.Object)
	require.Equal(t, int64(len("{\"messages\":[]}\n")), uploaded.Bytes)
	require.Equal(t, "training.jsonl", uploaded.Filename)
	require.Equal(t, "fine-tune", uploaded.Purpose)
	require.NotZero(t, uploaded.CreatedAt)

	listRecorder := httptest.NewRecorder()
	listCtx, _ := gin.CreateTestContext(listRecorder)
	listCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/files", nil)
	listCtx.Set("id", 1001)

	ListRelayFiles(listCtx)

	require.Equal(t, http.StatusOK, listRecorder.Code)
	var listResp relayFileListResponse
	require.NoError(t, common.Unmarshal(listRecorder.Body.Bytes(), &listResp))
	require.Equal(t, "list", listResp.Object)
	require.Len(t, listResp.Data, 1)
	require.Equal(t, uploaded.Id, listResp.Data[0].Id)

	getRecorder := httptest.NewRecorder()
	getCtx, _ := gin.CreateTestContext(getRecorder)
	getCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/files/"+uploaded.Id, nil)
	getCtx.Params = gin.Params{{Key: "id", Value: uploaded.Id}}
	getCtx.Set("id", 1001)

	RetrieveRelayFile(getCtx)

	require.Equal(t, http.StatusOK, getRecorder.Code)
	var retrieved model.OpenAIFileBrief
	require.NoError(t, common.Unmarshal(getRecorder.Body.Bytes(), &retrieved))
	require.Equal(t, uploaded.Id, retrieved.Id)

	contentRecorder := httptest.NewRecorder()
	contentCtx, _ := gin.CreateTestContext(contentRecorder)
	contentCtx.Request = httptest.NewRequest(http.MethodGet, "/v1/files/"+uploaded.Id+"/content", nil)
	contentCtx.Params = gin.Params{{Key: "id", Value: uploaded.Id}}
	contentCtx.Set("id", 1001)

	DownloadRelayFileContent(contentCtx)

	require.Equal(t, http.StatusOK, contentRecorder.Code)
	require.Equal(t, "{\"messages\":[]}\n", contentRecorder.Body.String())
	require.Equal(t, "attachment; filename=\"training.jsonl\"", contentRecorder.Header().Get("Content-Disposition"))

	deleteRecorder := httptest.NewRecorder()
	deleteCtx, _ := gin.CreateTestContext(deleteRecorder)
	deleteCtx.Request = httptest.NewRequest(http.MethodDelete, "/v1/files/"+uploaded.Id, nil)
	deleteCtx.Params = gin.Params{{Key: "id", Value: uploaded.Id}}
	deleteCtx.Set("id", 1001)

	DeleteRelayFile(deleteCtx)

	require.Equal(t, http.StatusOK, deleteRecorder.Code)
	var deleted relayFileDeleteResponse
	require.NoError(t, common.Unmarshal(deleteRecorder.Body.Bytes(), &deleted))
	require.Equal(t, uploaded.Id, deleted.Id)
	require.Equal(t, "file", deleted.Object)
	require.True(t, deleted.Deleted)
}

func TestRelayFileRequiresOwnership(t *testing.T) {
	setupRelayFilesTestDB(t)
	gin.SetMode(gin.TestMode)

	created, err := model.CreateOpenAIFile(1001, 2002, "secret.jsonl", "fine-tune", []byte("payload"))
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/v1/files/%s", created.Id), nil)
	ctx.Params = gin.Params{{Key: "id", Value: created.Id}}
	ctx.Set("id", 9999)

	RetrieveRelayFile(ctx)

	require.Equal(t, http.StatusNotFound, recorder.Code)
	require.Contains(t, recorder.Body.String(), "file_not_found")
}

func TestRelayFileRejectsOversizedUpload(t *testing.T) {
	setupRelayFilesTestDB(t)
	gin.SetMode(gin.TestMode)
	oldLimit := constant.MaxFileDownloadMB
	constant.MaxFileDownloadMB = 1
	defer func() {
		constant.MaxFileDownloadMB = oldLimit
	}()

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = newRelayFileUploadRequest(t, "large.jsonl", "fine-tune", bytes.Repeat([]byte("a"), (1<<20)+1))
	ctx.Set("id", 1001)
	ctx.Set("token_id", 2002)

	CreateRelayFile(ctx)

	require.Equal(t, http.StatusRequestEntityTooLarge, recorder.Code)
	require.Contains(t, recorder.Body.String(), "file_too_large")
}
