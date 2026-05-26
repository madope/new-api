package videopricing

import (
	"fmt"
	"strings"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/gin-gonic/gin"
)

type VideoPricingAdaptor interface {
	GetChannel() string
	ParseParams(c *gin.Context, info *relaycommon.RelayInfo) (VideoPricingParams, error)
	EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64
}

type BaseVideoPricingAdaptor struct {
	ChannelType int
}

func (a *BaseVideoPricingAdaptor) GetChannel() string {
	switch a.ChannelType {
	case 1:
		return "openai"
	case 2:
		return "minimax"
	case 3:
		return "doubao"
	case 4:
		return "suno"
	default:
		return fmt.Sprintf("channel_%d", a.ChannelType)
	}
}

func (a *BaseVideoPricingAdaptor) ParseParams(c *gin.Context, info *relaycommon.RelayInfo) (VideoPricingParams, error) {
	v, exists := c.Get("task_request")
	if !exists {
		return VideoPricingParams{}, fmt.Errorf("task_request not found in context")
	}

	req, ok := v.(relaycommon.TaskSubmitReq)
	if !ok {
		return VideoPricingParams{}, fmt.Errorf("task_request is not TaskSubmitReq")
	}

	modelPricing, _ := a.GetModelPricing(info.OriginModelName)
	resolution := a.parseResolution(req, modelPricing.DefaultResolution)
	duration := a.parseDuration(req, modelPricing.DefaultDuration)
	referenceTypes := a.parseReferenceTypes(req)
	inputTokens, outputTokens := a.parseTokens(req)

	return VideoPricingParams{
		Channel:        a.GetChannel(),
		Model:          info.OriginModelName,
		Resolution:     resolution,
		Duration:       duration,
		ReferenceTypes: referenceTypes,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
	}, nil
}

func (a *BaseVideoPricingAdaptor) parseResolution(req relaycommon.TaskSubmitReq, defaultResolution string) string {
	if defaultResolution == "" {
		defaultResolution = "768p"
	}
	// 优先从 Metadata 获取 resolution（主要参数）
	if req.Metadata != nil {
		if res, ok := req.Metadata["resolution"].(string); ok && res != "" {
			return strings.ToLower(res)
		}
	}
	// 其次从 size 字段获取（兼容别名）
	if req.Size != "" {
		return strings.ToLower(req.Size)
	}
	// 最后从 Metadata 获取 size（备用）
	if req.Metadata != nil {
		if res, ok := req.Metadata["size"].(string); ok && res != "" {
			return strings.ToLower(res)
		}
	}
	return defaultResolution
}

func (a *BaseVideoPricingAdaptor) parseDuration(req relaycommon.TaskSubmitReq, defaultDuration int) int {
	if defaultDuration <= 0 {
		defaultDuration = 6
	}
	if req.Duration > 0 {
		return req.Duration
	}
	if req.Metadata != nil {
		if dur, ok := req.Metadata["duration"].(float64); ok {
			return int(dur)
		}
	}
	return defaultDuration
}

func (a *BaseVideoPricingAdaptor) parseReferenceTypes(req relaycommon.TaskSubmitReq) []ReferenceType {
	var types []ReferenceType

	if req.HasImage() {
		types = append(types, ReferenceImage)
	}

	return types
}

func (a *BaseVideoPricingAdaptor) parseTokens(req relaycommon.TaskSubmitReq) (int, int) {
	inputTokens := 0
	outputTokens := 0

	if req.Prompt != "" {
		inputTokens = len(req.Prompt) / 4
	}

	imageCount := 0
	videoCount := 0
	audioCount := 0

	if req.HasImage() {
		imageCount++
	}

	inputTokens += imageCount * 768
	inputTokens += videoCount * 1000
	inputTokens += audioCount * 500

	return inputTokens, outputTokens
}

func (a *BaseVideoPricingAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	params, err := a.ParseParams(c, info)
	if err != nil {
		return nil
	}

	ratio, err := ComputeVideoRatio(info.OriginModelName, params)
	if err != nil {
		return nil
	}

	return map[string]float64{
		"video_ratio": ratio,
	}
}

func (a *BaseVideoPricingAdaptor) GetModelPricing(modelName string) (ModelPricing, bool) {
	return GetModelPricing(modelName)
}

func (a *BaseVideoPricingAdaptor) GetDefaultBasePrice(modelName string) float64 {
	return GetDefaultBasePrice(modelName)
}

type CustomVideoPricingAdaptor struct {
	BaseVideoPricingAdaptor
	ChannelName string
}

func (a *CustomVideoPricingAdaptor) GetChannel() string {
	return a.ChannelName
}

func NewCustomVideoPricingAdaptor(channelName string) *CustomVideoPricingAdaptor {
	return &CustomVideoPricingAdaptor{
		ChannelName: channelName,
	}
}

type VideoPricingMiddleware struct {
	Adaptor VideoPricingAdaptor
}

func (m *VideoPricingMiddleware) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return m.Adaptor.EstimateBilling(c, info)
}

type VideoGenerationRequest struct {
	Model          string                 `json:"model"`
	Prompt         string                 `json:"prompt"`
	GenerateAudio  bool                   `json:"generate_audio,omitempty"`
	Ratio          string                 `json:"ratio,omitempty"`
	Watermark      bool                   `json:"watermark,omitempty"`
	Duration       int                    `json:"duration,omitempty"`
	Resolution     string                 `json:"resolution,omitempty"`
	Size           string                 `json:"size,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

func ParseVideoGenerationRequest(c *gin.Context) (VideoGenerationRequest, error) {
	var req VideoGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return VideoGenerationRequest{}, err
	}

	if req.Resolution != "" && req.Size == "" {
		req.Size = req.Resolution
	}

	return req, nil
}

func ExtractPricingParams(c *gin.Context, info *relaycommon.RelayInfo) (VideoPricingParams, error) {
	videoReq, err := ParseVideoGenerationRequest(c)
	if err != nil {
		return VideoPricingParams{}, err
	}

	modelPricing, _ := GetModelPricing(info.OriginModelName)

	resolution := videoReq.Size
	if resolution == "" {
		resolution = modelPricing.DefaultResolution
		if resolution == "" {
			resolution = "768p"
		}
	}

	duration := videoReq.Duration
	if duration <= 0 {
		duration = modelPricing.DefaultDuration
		if duration <= 0 {
			duration = 6
		}
	}

	return VideoPricingParams{
		Channel:        fmt.Sprintf("channel_%d", info.ChannelType),
		Model:          info.OriginModelName,
		Resolution:     strings.ToLower(resolution),
		Duration:       duration,
		ReferenceTypes: []ReferenceType{},
		InputTokens:    0,
		OutputTokens:   0,
	}, nil
}

func GetReferenceTypesFromVideoRequest(req VideoGenerationRequest) []ReferenceType {
	var types []ReferenceType

	if req.Metadata != nil {
		if _, ok := req.Metadata["image"].(string); ok {
			types = append(types, ReferenceImage)
		}
		if _, ok := req.Metadata["video"].(string); ok {
			types = append(types, ReferenceVideo)
		}
		if _, ok := req.Metadata["audio"].(string); ok {
			types = append(types, ReferenceAudio)
		}
	}

	return types
}
