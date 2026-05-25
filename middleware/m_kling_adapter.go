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
	mKlingRequestTypeText2Video       = "text2video"
	mKlingRequestTypeImage2Video      = "image2video"
	mKlingRequestTypeMultiImage2Video = "multi-image2video"
	mKlingRequestTypeOmniVideo        = "omni-video"
)

func MKlingRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyMKlingCompat, true)

		requestType := detectMKlingRequestType(c.Request.URL.Path)
		if requestType == "" {
			c.Next()
			return
		}

		var originalReq map[string]any
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			abortWithOpenAiMessage(c, http.StatusBadRequest, "Invalid request body")
			return
		}

		unifiedReq, action, err := convertMKlingRequestToUnified(originalReq, requestType)
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

func detectMKlingRequestType(path string) string {
	switch {
	case strings.HasSuffix(path, "/text2video"):
		return mKlingRequestTypeText2Video
	case strings.HasSuffix(path, "/image2video"):
		return mKlingRequestTypeImage2Video
	case strings.HasSuffix(path, "/multi-image2video"):
		return mKlingRequestTypeMultiImage2Video
	case strings.HasSuffix(path, "/omni-video"):
		return mKlingRequestTypeOmniVideo
	default:
		return ""
	}
}

func convertMKlingRequestToUnified(originalReq map[string]any, requestType string) (map[string]any, string, error) {
	model, _ := originalReq["model_name"].(string)
	if strings.TrimSpace(model) == "" {
		model, _ = originalReq["model"].(string)
	}
	if strings.TrimSpace(model) == "" {
		return nil, "", fmt.Errorf("model_name is required")
	}

	unifiedReq := map[string]any{
		"model": model,
	}

	metadata := make(map[string]any, len(originalReq))
	for key, value := range originalReq {
		switch key {
		case "model":
			continue
		case "model_name":
			metadata[key] = value
			continue
		case "prompt":
			if prompt, ok := value.(string); ok && strings.TrimSpace(prompt) != "" {
				unifiedReq["prompt"] = prompt
			}
			metadata[key] = value
			continue
		case "duration":
			unifiedReq["duration"] = value
			metadata[key] = value
			continue
		case "mode":
			unifiedReq["mode"] = value
			metadata[key] = value
			continue
		case "image":
			if image, ok := value.(string); ok && strings.TrimSpace(image) != "" {
				unifiedReq["image"] = image
			}
			metadata[key] = value
			continue
		case "images":
			unifiedReq["images"] = value
			metadata[key] = value
			continue
		default:
			metadata[key] = value
		}
	}

	if len(metadata) > 0 {
		unifiedReq["metadata"] = metadata
	}

	return unifiedReq, mapMKlingAction(requestType), nil
}

func mapMKlingAction(requestType string) string {
	switch requestType {
	case mKlingRequestTypeText2Video:
		return constant.TaskActionTextGenerate
	case mKlingRequestTypeImage2Video:
		return constant.TaskActionFirstTailGenerate
	case mKlingRequestTypeMultiImage2Video:
		return constant.TaskActionReferenceGenerate
	case mKlingRequestTypeOmniVideo:
		return constant.TaskActionReferenceGenerate
	default:
		return constant.TaskActionGenerate
	}
}
