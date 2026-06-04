package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

func GetModelRegistries(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := model.ModelRegistryQuery{
		ExternalModel: c.Query("model"),
		Provider:      c.Query("provider"),
		Protocol:      c.Query("protocol"),
		Enabled:       parseOptionalBool(c.Query("enabled")),
		StartIdx:      pageInfo.GetStartIdx(),
		Limit:         pageInfo.GetPageSize(),
	}
	items, err := model.ListModelRegistries(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, err := model.CountModelRegistries(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetModelRegistry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	registry, found, err := model.GetModelRegistryByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !found {
		common.ApiError(c, errors.New("model registry not found"))
		return
	}
	common.ApiSuccess(c, registry)
}

func CreateModelRegistry(c *gin.Context) {
	var registry model.ModelRegistry
	if err := c.ShouldBindJSON(&registry); err != nil {
		common.ApiError(c, err)
		return
	}
	if registry.ExternalModel == "" || registry.Provider == "" || registry.Protocol == "" {
		common.ApiError(c, errors.New("external_model, provider, and protocol are required"))
		return
	}
	if err := registry.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &registry)
}

func UpdateModelRegistry(c *gin.Context) {
	var registry model.ModelRegistry
	if err := c.ShouldBindJSON(&registry); err != nil {
		common.ApiError(c, err)
		return
	}
	if registry.Id <= 0 {
		common.ApiError(c, errors.New("model registry id is required"))
		return
	}
	if registry.ExternalModel == "" || registry.Provider == "" || registry.Protocol == "" {
		common.ApiError(c, errors.New("external_model, provider, and protocol are required"))
		return
	}
	if err := registry.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &registry)
}

func DeleteModelRegistry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteModelRegistry(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func GetProviderRegistries(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	query := model.ProviderRegistryQuery{
		Provider:     c.Query("provider"),
		Protocol:     c.Query("protocol"),
		HealthStatus: c.Query("health_status"),
		Enabled:      parseOptionalBool(c.Query("enabled")),
		StartIdx:     pageInfo.GetStartIdx(),
		Limit:        pageInfo.GetPageSize(),
	}
	items, err := model.ListProviderRegistries(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, err := model.CountProviderRegistries(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetProviderRegistry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	registry, found, err := model.GetProviderRegistryByID(id)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if !found {
		common.ApiError(c, errors.New("provider registry not found"))
		return
	}
	common.ApiSuccess(c, registry)
}

func CreateProviderRegistry(c *gin.Context) {
	var registry model.ProviderRegistry
	if err := c.ShouldBindJSON(&registry); err != nil {
		common.ApiError(c, err)
		return
	}
	if registry.Provider == "" || registry.Protocol == "" {
		common.ApiError(c, errors.New("provider and protocol are required"))
		return
	}
	if err := registry.Insert(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &registry)
}

func UpdateProviderRegistry(c *gin.Context) {
	var registry model.ProviderRegistry
	if err := c.ShouldBindJSON(&registry); err != nil {
		common.ApiError(c, err)
		return
	}
	if registry.Id <= 0 {
		common.ApiError(c, errors.New("provider registry id is required"))
		return
	}
	if registry.Provider == "" || registry.Protocol == "" {
		common.ApiError(c, errors.New("provider and protocol are required"))
		return
	}
	if err := registry.Update(); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, &registry)
}

func DeleteProviderRegistry(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if err := model.DeleteProviderRegistry(id); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func parseOptionalBool(raw string) *bool {
	if raw == "" {
		return nil
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return nil
	}
	return &value
}
