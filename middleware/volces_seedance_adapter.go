package middleware

import (
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"

	"github.com/gin-gonic/gin"
)

func VolcesSeedanceRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyVolcesCompat, true)

		var originalReq map[string]any
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			abortWithOpenAiMessage(c, 400, "Invalid request body")
			return
		}

		unifiedReq, err := convertVolcesRequestToUnified(originalReq)
		if err != nil {
			abortWithOpenAiMessage(c, 400, err.Error())
			return
		}

		jsonData, err := common.Marshal(unifiedReq)
		if err != nil {
			abortWithOpenAiMessage(c, 500, "Failed to marshal request body")
			return
		}

		if err := common.ReplaceRequestBody(c, jsonData); err != nil {
			abortWithOpenAiMessage(c, 500, "Failed to replace request body")
			return
		}
		c.Request.URL.Path = "/v1/video/generations"
		c.Next()
	}
}

func convertVolcesRequestToUnified(originalReq map[string]any) (map[string]any, error) {
	model, _ := originalReq["model"].(string)
	if strings.TrimSpace(model) == "" {
		return nil, fmt.Errorf("model is required")
	}

	metadata := make(map[string]any, len(originalReq))
	for key, value := range originalReq {
		metadata[key] = value
	}

	var promptParts []string
	var nonTextContent []any
	if contentRaw, ok := originalReq["content"]; ok && contentRaw != nil {
		contentItems, ok := contentRaw.([]any)
		if !ok {
			return nil, fmt.Errorf("content must be an array")
		}
		for _, item := range contentItems {
			itemMap, ok := item.(map[string]any)
			if !ok {
				nonTextContent = append(nonTextContent, item)
				continue
			}
			if itemType, _ := itemMap["type"].(string); itemType == "text" {
				if text, _ := itemMap["text"].(string); strings.TrimSpace(text) != "" {
					promptParts = append(promptParts, text)
				}
				continue
			}
			nonTextContent = append(nonTextContent, itemMap)
		}
	}

	delete(metadata, "model")
	delete(metadata, "content")
	delete(metadata, "duration")

	if len(nonTextContent) > 0 {
		metadata["content"] = nonTextContent
	}

	unifiedReq := map[string]any{
		"model":  model,
		"prompt": strings.Join(promptParts, "\n"),
	}
	if durationValue, ok := originalReq["duration"]; ok {
		switch v := durationValue.(type) {
		case string:
			if strings.TrimSpace(v) != "" {
				unifiedReq["seconds"] = v
			}
		case float64:
			unifiedReq["seconds"] = fmt.Sprintf("%.0f", v)
		case int:
			unifiedReq["seconds"] = fmt.Sprintf("%d", v)
		case int64:
			unifiedReq["seconds"] = fmt.Sprintf("%d", v)
		}
	}
	if len(metadata) > 0 {
		unifiedReq["metadata"] = metadata
	}
	return unifiedReq, nil
}
