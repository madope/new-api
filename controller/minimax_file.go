package controller

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

func minimaxFileError(code int, msg string) gin.H {
	return gin.H{
		"base_resp": gin.H{
			"status_code": code,
			"status_msg":  msg,
		},
	}
}

func MiniMaxFileRetrieve(c *gin.Context) {
	fileID := c.Param("file_id")
	if fileID == "" {
		c.JSON(http.StatusBadRequest, minimaxFileError(2013, "file_id is required"))
		return
	}

	userGroup := common.GetContextKeyString(c, constant.ContextKeyUserGroup)
	if userGroup == "" {
		userGroup = "default"
	}

	channel, _, err := service.CacheGetRandomSatisfiedChannel(&service.RetryParam{
		Ctx:        c,
		ModelName:  "minimax-files",
		TokenGroup: userGroup,
		Retry:      common.GetPointer(0),
	})
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("MiniMax file retrieve - no available channel: %s", err.Error()))
		c.JSON(http.StatusServiceUnavailable, minimaxFileError(5001, "no available channel"))
		return
	}
	if channel == nil {
		c.JSON(http.StatusServiceUnavailable, minimaxFileError(5001, "no available channel"))
		return
	}

	baseURL := channel.GetBaseURL()
	if baseURL == "" {
		baseURL = "https://api.minimaxi.com"
	}

	key, _, apiErr := channel.GetNextEnabledKey()
	if apiErr != nil {
		c.JSON(http.StatusServiceUnavailable, minimaxFileError(5003, "no available key"))
		return
	}

	uri := fmt.Sprintf("%s/v1/files/retrieve?file_id=%s", baseURL, fileID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("MiniMax file retrieve - build request failed: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, minimaxFileError(5002, "build request failed"))
		return
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("MiniMax file retrieve - upstream request failed: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, minimaxFileError(5002, "upstream request failed"))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.LogError(c.Request.Context(), fmt.Sprintf("MiniMax file retrieve - read response failed: %s", err.Error()))
		c.JSON(http.StatusInternalServerError, minimaxFileError(5002, "read response failed"))
		return
	}

	logger.LogInfo(c.Request.Context(), fmt.Sprintf("MiniMax file retrieve - file_id=%s, status=%d, response: %s", fileID, resp.StatusCode, string(body)))

	c.Data(resp.StatusCode, "application/json", body)
}
