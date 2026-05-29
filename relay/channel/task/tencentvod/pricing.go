package tencentvod

import (
	"strings"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper/videopricing"
	"github.com/gin-gonic/gin"
)

type TencentVODVideoPricingAdaptor struct {
	*videopricing.BaseVideoPricingAdaptor
}

func (a *TencentVODVideoPricingAdaptor) ParseParams(c *gin.Context, info *relaycommon.RelayInfo) (videopricing.VideoPricingParams, error) {
	params, err := a.BaseVideoPricingAdaptor.ParseParams(c, info)
	if err != nil {
		return videopricing.VideoPricingParams{}, err
	}

	v, exists := c.Get("task_request")
	if !exists {
		return params, nil
	}
	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return params, nil
	}

	meta, _ := parseTaskMeta(req.Metadata)
	if meta == nil {
		return params, nil
	}

	if containsVideo(meta.FileInfos) {
		params.ReferenceTypes = append(params.ReferenceTypes, videopricing.ReferenceVideo)
	}
	// Kling 官方 video_list 参数
	if hasNonEmptyList(req.Metadata, "video_list") {
		params.ReferenceTypes = append(params.ReferenceTypes, videopricing.ReferenceVideo)
	}
	if meta.LastFrameFileID != "" || meta.LastFrameURL != "" || hasFirstFrame(meta.FileInfos) {
		params.ReferenceTypes = append(params.ReferenceTypes, videopricing.ReferenceImage)
	}
	// Kling 官方 image_list 参数
	if hasNonEmptyList(req.Metadata, "image_list") {
		params.ReferenceTypes = append(params.ReferenceTypes, videopricing.ReferenceImage)
	}
	// Kling 官方 image_tail 参数（首尾帧尾帧）
	if v, ok := req.Metadata["image_tail"]; ok {
		if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
			params.ReferenceTypes = append(params.ReferenceTypes, videopricing.ReferenceImage)
		}
	}

	if meta.OutputConfig != nil && meta.OutputConfig.Resolution != "" {
		params.Resolution = strings.ToLower(meta.OutputConfig.Resolution)
	} else if normalized := normalizeResolution(params.Resolution); normalized != "" {
		params.Resolution = normalized
	}

	params.Duration = resolveDuration(&req)

	hasVoice := hasVoiceID(meta.FileInfos) || hasNonEmptyList(req.Metadata, "voice_list")
	if meta.OutputConfig != nil && meta.OutputConfig.AudioGeneration == "Enabled" {
		if hasVoice {
			params.AudioOutput = "voice_timbre"
		} else {
			params.AudioOutput = "voice"
		}
	} else if hasVoice {
		params.AudioOutput = "voice_timbre"
	} else {
		params.AudioOutput = "none"
	}

	return params, nil
}

// hasNonEmptyList 检查 metadata 中指定 key 是否为非空数组
func hasNonEmptyList(metadata map[string]any, key string) bool {
	if metadata == nil {
		return false
	}
	raw, ok := metadata[key]
	if !ok || raw == nil {
		return false
	}
	// 支持 []any 和 []string 两种类型
	switch v := raw.(type) {
	case []any:
		return len(v) > 0
	case []string:
		return len(v) > 0
	}
	// 尝试 JSON 解析判断
	data, err := common.Marshal(raw)
	if err != nil {
		return false
	}
	var items []any
	if err := common.Unmarshal(data, &items); err != nil {
		return false
	}
	return len(items) > 0
}

func hasVoiceID(files []AigcVideoTaskInputFileInfo) bool {
	for _, f := range files {
		if f.VoiceID != "" {
			return true
		}
	}
	return false
}

func (a *TencentVODVideoPricingAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	params, err := a.ParseParams(c, info)
	if err != nil {
		return nil
	}

	ratio, err := videopricing.ComputeVideoRatio(info.OriginModelName, params)
	if err != nil {
		return nil
	}

	return map[string]float64{
		"video_ratio": ratio,
	}
}
