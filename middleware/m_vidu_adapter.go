package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

const (
	mViduRequestTypeText2Video      = "text2video"
	mViduRequestTypeImg2Video       = "img2video"
	mViduRequestTypeStartEnd2Video  = "start-end2video"
	mViduRequestTypeReference2Video = "reference2video"
)

func MViduRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyMViduCompat, true)

		requestType := detectMViduRequestType(c.Request.URL.Path)
		if requestType == "" {
			c.Next()
			return
		}

		var originalReq map[string]any
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		unifiedReq, action, err := convertMViduRequestToUnified(originalReq, requestType)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, err.Error())
			return
		}

		jsonData, err := common.Marshal(unifiedReq)
		if err != nil {
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "Failed to marshal request body")
			return
		}
		if err := common.ReplaceRequestBody(c, jsonData); err != nil {
			abortWithOpenAiMessage(c, http.StatusInternalServerError, "Failed to replace request body")
			return
		}

		c.Request.URL.Path = "/v1/video/generations"
		c.Set("action", action)
		c.Next()
	}
}

func detectMViduRequestType(path string) string {
	switch {
	case strings.HasSuffix(path, "/text2video"):
		return mViduRequestTypeText2Video
	case strings.HasSuffix(path, "/img2video"):
		return mViduRequestTypeImg2Video
	case strings.HasSuffix(path, "/start-end2video"):
		return mViduRequestTypeStartEnd2Video
	case strings.HasSuffix(path, "/reference2video"):
		return mViduRequestTypeReference2Video
	default:
		return ""
	}
}

func convertMViduRequestToUnified(originalReq map[string]any, requestType string) (map[string]any, string, error) {
	model, _ := originalReq["model"].(string)
	if strings.TrimSpace(model) == "" {
		return nil, "", fmt.Errorf("model is required")
	}

	unifiedReq := map[string]any{
		"model": model,
	}

	metadata := make(map[string]any, len(originalReq))
	for key, value := range originalReq {
		switch key {
		case "model":
			continue
		case "prompt":
			if prompt, ok := value.(string); ok && strings.TrimSpace(prompt) != "" {
				unifiedReq["prompt"] = prompt
				continue
			}
		case "duration":
			unifiedReq["duration"] = value
			continue
		case "images":
			unifiedReq["images"] = value
			metadata[key] = value
			continue
		}
		metadata[key] = value
	}

	if len(metadata) > 0 {
		unifiedReq["metadata"] = metadata
	}

	return unifiedReq, mapMViduAction(requestType), nil
}

func mapMViduAction(requestType string) string {
	switch requestType {
	case mViduRequestTypeText2Video:
		return constant.TaskActionTextGenerate
	case mViduRequestTypeStartEnd2Video:
		return constant.TaskActionFirstTailGenerate
	case mViduRequestTypeReference2Video:
		return constant.TaskActionReferenceGenerate
	default:
		return constant.TaskActionGenerate
	}
}
