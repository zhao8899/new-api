package controller

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func requireRelayFilePermission(c *gin.Context, minRole int) bool {
	if minRole <= common.RoleGuestUser {
		return true
	}
	userId := c.GetInt("id")
	if userId <= 0 {
		relayFileError(c, http.StatusUnauthorized, "authentication required", "invalid_request_error", "authentication_required")
		return false
	}
	user, err := model.GetUserById(userId, false)
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to load user", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeQueryDataError))
		return false
	}
	if user.Role < minRole {
		relayFileError(c, http.StatusForbidden, "insufficient permissions", "invalid_request_error", "insufficient_permissions")
		return false
	}
	return true
}

func relayFileError(c *gin.Context, status int, message, errorType, code string) {
	c.JSON(status, gin.H{
		"error": types.OpenAIError{
			Message: message,
			Type:    errorType,
			Code:    code,
		},
	})
}

func ListRelayFiles(c *gin.Context) {
	if !requireRelayFilePermission(c, common.FileDownloadPermission) {
		return
	}
	files, err := model.ListOpenAIFilesByUserId(c.GetInt("id"))
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to list files", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeQueryDataError))
		return
	}
	data := make([]model.OpenAIFileBrief, 0, len(files))
	for _, file := range files {
		data = append(data, file.ToBrief())
	}
	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

func CreateRelayFile(c *gin.Context) {
	if !requireRelayFilePermission(c, common.FileUploadPermission) {
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		relayFileError(c, http.StatusBadRequest, "file is required", "invalid_request_error", "file_required")
		return
	}
	purpose := strings.TrimSpace(c.PostForm("purpose"))
	if purpose == "" {
		relayFileError(c, http.StatusBadRequest, "purpose is required", "invalid_request_error", "purpose_required")
		return
	}
	maxBytes := relayFileMaxUploadBytes()
	if fileHeader.Size > maxBytes {
		relayFileError(c, http.StatusRequestEntityTooLarge, "file size exceeds maximum allowed size", "invalid_request_error", "file_too_large")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		relayFileError(c, http.StatusBadRequest, "failed to open upload", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeBadRequestBody))
		return
	}
	defer file.Close()
	content, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		relayFileError(c, http.StatusBadRequest, "failed to read upload", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeReadRequestBodyFailed))
		return
	}
	if int64(len(content)) > maxBytes {
		relayFileError(c, http.StatusRequestEntityTooLarge, "file size exceeds maximum allowed size", "invalid_request_error", "file_too_large")
		return
	}
	created, err := model.CreateOpenAIFile(c.GetInt("id"), c.GetInt("token_id"), fileHeader.Filename, purpose, content)
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to create file", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeUpdateDataError))
		return
	}
	c.JSON(http.StatusOK, created.ToBrief())
}

func RetrieveRelayFile(c *gin.Context) {
	file, ok := getRelayFileForCurrentUser(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, file.ToBrief())
}

func DownloadRelayFileContent(c *gin.Context) {
	file, ok := getRelayFileForCurrentUser(c)
	if !ok {
		return
	}
	content, err := file.ContentBytes()
	if err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to read file content", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeReadResponseBodyFailed))
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", file.Filename))
	c.Data(http.StatusOK, "application/octet-stream", content)
}

func DeleteRelayFile(c *gin.Context) {
	if !requireRelayFilePermission(c, common.FileUploadPermission) {
		return
	}
	file, ok := getRelayFileForCurrentUser(c)
	if !ok {
		return
	}
	if err := model.DeleteOpenAIFileByIDAndUserID(file.Id, c.GetInt("id")); err != nil {
		relayFileError(c, http.StatusInternalServerError, "failed to delete file", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeUpdateDataError))
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"id":      file.Id,
		"object":  "file",
		"deleted": true,
	})
}

func getRelayFileForCurrentUser(c *gin.Context) (*model.OpenAIFile, bool) {
	if !requireRelayFilePermission(c, common.FileDownloadPermission) {
		return nil, false
	}
	fileId := strings.TrimSpace(c.Param("id"))
	if fileId == "" {
		relayFileError(c, http.StatusBadRequest, "file id is required", "invalid_request_error", "file_id_required")
		return nil, false
	}
	file, err := model.GetOpenAIFileByIDAndUserID(fileId, c.GetInt("id"))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			relayFileError(c, http.StatusNotFound, "file not found", "invalid_request_error", "file_not_found")
			return nil, false
		}
		relayFileError(c, http.StatusInternalServerError, "failed to query file", string(types.ErrorTypeNewAPIError), string(types.ErrorCodeQueryDataError))
		return nil, false
	}
	return file, true
}

func relayFileMaxUploadBytes() int64 {
	limitMB := constant.MaxFileDownloadMB
	if limitMB <= 0 {
		limitMB = 64
	}
	return int64(limitMB) << 20
}
