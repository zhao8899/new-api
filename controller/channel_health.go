package controller

import (
	"errors"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type channelHealthModeUpdateRequest struct {
	Mode string `json:"mode"`
}

func GetChannelHealthRecords(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	channelID, _ := strconv.Atoi(c.Query("channel_id"))
	query := model.ChannelHealthQuery{
		ChannelID:     channelID,
		Provider:      c.Query("provider"),
		ModelName:     c.Query("model"),
		CircuitState:  c.Query("circuit_state"),
		LastErrorType: c.Query("last_error_type"),
		StartIdx:      pageInfo.GetStartIdx(),
		Limit:         pageInfo.GetPageSize(),
	}
	items, err := model.ListChannelHealth(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	total, err := model.CountChannelHealth(query)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetChannelHealthMode(c *gin.Context) {
	common.ApiSuccess(c, gin.H{
		"mode": model.ChannelHealthCircuitMode(),
	})
}

func UpdateChannelHealthMode(c *gin.Context) {
	var req channelHealthModeUpdateRequest
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		common.ApiError(c, err)
		return
	}
	if !model.IsValidChannelHealthCircuitMode(req.Mode) {
		common.ApiError(c, errors.New("invalid channel health circuit mode"))
		return
	}
	mode := model.NormalizeChannelHealthCircuitMode(req.Mode)
	if err := model.UpdateOption(model.ChannelHealthCircuitModeOption, mode); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"mode": mode,
	})
}
