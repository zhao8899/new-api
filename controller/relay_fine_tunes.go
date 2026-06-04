package controller

import (
	"errors"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type createRelayFineTuneRequest struct {
	Model          string `json:"model"`
	TrainingFile   string `json:"training_file"`
	ValidationFile string `json:"validation_file,omitempty"`
}

func ListRelayFineTunes(c *gin.Context) {
	if !requireRelayFilePermission(c, common.FileDownloadPermission) {
		return
	}
	items, err := model.ListOpenAIFineTunesByUserID(c.GetInt("id"))
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to list fine-tunes", "new_api_error", "query_data_error")
		return
	}
	data := make([]model.OpenAIFineTuneDTO, 0, len(items))
	for _, item := range items {
		data = append(data, item.ToDTO())
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func CreateRelayFineTune(c *gin.Context) {
	if !requireRelayFilePermission(c, common.FileUploadPermission) {
		return
	}
	var req createRelayFineTuneRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		relayFileError(c, http.StatusBadRequest, "invalid request body", "invalid_request_error", "invalid_request")
		return
	}
	req.TrainingFile = strings.TrimSpace(req.TrainingFile)
	req.ValidationFile = strings.TrimSpace(req.ValidationFile)
	req.Model = strings.TrimSpace(req.Model)
	if req.TrainingFile == "" {
		relayFileError(c, http.StatusBadRequest, "training_file is required", "invalid_request_error", "training_file_required")
		return
	}
	if req.Model == "" {
		relayFileError(c, http.StatusBadRequest, "model is required", "invalid_request_error", "model_required")
		return
	}
	if _, err := model.GetOpenAIFileByIDAndUserID(req.TrainingFile, c.GetInt("id")); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			relayFileError(c, http.StatusNotFound, "training file not found", "invalid_request_error", "training_file_not_found")
			return
		}
		relayFileError(c, http.StatusInternalServerError, "failed to query training file", "new_api_error", "query_data_error")
		return
	}
	if req.ValidationFile != "" {
		if _, err := model.GetOpenAIFileByIDAndUserID(req.ValidationFile, c.GetInt("id")); err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				relayFileError(c, http.StatusNotFound, "validation file not found", "invalid_request_error", "validation_file_not_found")
				return
			}
			relayFileError(c, http.StatusInternalServerError, "failed to query validation file", "new_api_error", "query_data_error")
			return
		}
	}
	item, err := model.CreateOpenAIFineTune(c.GetInt("id"), c.GetInt("token_id"), req.TrainingFile, req.ValidationFile, req.Model)
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to create fine-tune", "new_api_error", "update_data_error")
		return
	}
	c.JSON(http.StatusOK, item.ToDTO())
}

func RetrieveRelayFineTune(c *gin.Context) {
	item, ok := getRelayFineTuneForCurrentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, item.ToDTO())
}

func CancelRelayFineTune(c *gin.Context) {
	if !requireRelayFilePermission(c, common.FileUploadPermission) {
		return
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		relayFileError(c, http.StatusBadRequest, "fine-tune id is required", "invalid_request_error", "fine_tune_id_required")
		return
	}
	item, err := model.CancelOpenAIFineTuneByIDAndUserID(id, c.GetInt("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			relayFileError(c, http.StatusNotFound, "fine-tune not found", "invalid_request_error", "fine_tune_not_found")
			return
		}
		relayFileError(c, http.StatusInternalServerError, "failed to cancel fine-tune", "new_api_error", "update_data_error")
		return
	}
	c.JSON(http.StatusOK, item.ToDTO())
}

func ListRelayFineTuneEvents(c *gin.Context) {
	item, ok := getRelayFineTuneForCurrentUser(c)
	if !ok {
		return
	}
	events, err := model.ListOpenAIFineTuneEventsByFineTuneIDAndUserID(item.ID, c.GetInt("id"))
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to list fine-tune events", "new_api_error", "query_data_error")
		return
	}
	data := make([]model.OpenAIFineTuneEventDTO, 0, len(events))
	for _, event := range events {
		data = append(data, event.ToDTO())
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func getRelayFineTuneForCurrentUser(c *gin.Context) (*model.OpenAIFineTune, bool) {
	if !requireRelayFilePermission(c, common.FileDownloadPermission) {
		return nil, false
	}
	id := strings.TrimSpace(c.Param("id"))
	if id == "" {
		relayFileError(c, http.StatusBadRequest, "fine-tune id is required", "invalid_request_error", "fine_tune_id_required")
		return nil, false
	}
	item, err := model.GetOpenAIFineTuneByIDAndUserID(id, c.GetInt("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			relayFileError(c, http.StatusNotFound, "fine-tune not found", "invalid_request_error", "fine_tune_not_found")
			return nil, false
		}
		relayFileError(c, http.StatusInternalServerError, "failed to query fine-tune", "new_api_error", "query_data_error")
		return nil, false
	}
	return item, true
}
