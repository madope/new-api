package videopricing

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/QuantumNous/new-api/common"
)

const (
	VideoPricingConfigKey = "video_pricing_config"
)

type ReferenceType string

const (
	ReferenceImage ReferenceType = "image"
	ReferenceVideo ReferenceType = "video"
	ReferenceAudio ReferenceType = "audio"
)

type BillingMode string

const (
	BillingModePerCall    BillingMode = "per_call"
	BillingModePerToken   BillingMode = "per_token"
	BillingModePerSecond  BillingMode = "per_second"
)

type PricingCondition struct {
	Resolution     []string        `json:"resolution"`
	Duration       []int           `json:"duration"`
	ReferenceTypes []ReferenceType `json:"reference_types"`
	AudioOutput    string          `json:"audio_output,omitempty"` // 音频输出条件：""（不限）、"none"（无声）、"voice"（有声）、"voice_timbre"（有声+有音色）
}

type PricingRule struct {
	Conditions         PricingCondition `json:"conditions"`
	Price              float64         `json:"price"`              // 按次计费时的价格
	InputPricePerMToken float64        `json:"input_price_per_m_token"`   // 按token计费时输入单价（每M token）
	OutputPricePerMToken float64       `json:"output_price_per_m_token"`  // 按token计费时输出单价（每M token）
}

type ModelPricing struct {
	BillingMode           BillingMode    `json:"billing_mode"`
	BasePrice             float64        `json:"base_price"`              // 基础价格（元/M token）
	Markup                float64        `json:"markup"`
	DefaultResolution     string         `json:"default_resolution"`      // 默认分辨率，空时回落 "768p"
	DefaultDuration       int            `json:"default_duration"`        // 默认时长（秒），0 时回落 6
	DefaultReferenceTypes []ReferenceType `json:"default_reference_types"` // 默认参考类型，仅用于展示
	PricingRules          []PricingRule  `json:"pricing_rules"`
}

type VideoPricingConfig struct {
	Models map[string]ModelPricing `json:"models"`
}

var (
	configCache     VideoPricingConfig
	configCacheLock sync.RWMutex
	lastCacheTime   int64
)

type VideoPricingParams struct {
	Channel        string
	Model          string
	Resolution     string
	Duration       int
	ReferenceTypes []ReferenceType
	AudioOutput    string // 音频输出类型："none"、"voice"、"voice_timbre"
	InputTokens    int
	OutputTokens   int
}

func LoadConfig() error {
	configCacheLock.Lock()
	defer configCacheLock.Unlock()

	common.OptionMapRWMutex.RLock()
	configStr, ok := common.OptionMap[VideoPricingConfigKey]
	common.OptionMapRWMutex.RUnlock()

	if !ok || configStr == "" {
		return fmt.Errorf("video pricing config not found")
	}

	var config VideoPricingConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return fmt.Errorf("failed to unmarshal video pricing config: %w", err)
	}

	configCache = config
	lastCacheTime = common.GetTimestamp()

	return nil
}

func GetConfig() VideoPricingConfig {
	// 每次都从 OptionMap 读取最新配置，确保更新后立即生效
	configCacheLock.Lock()
	defer configCacheLock.Unlock()

	common.OptionMapRWMutex.RLock()
	configStr, ok := common.OptionMap[VideoPricingConfigKey]
	common.OptionMapRWMutex.RUnlock()

	if !ok || configStr == "" {
		return configCache // 返回缓存的配置（如果有）
	}

	var config VideoPricingConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		common.SysLog(fmt.Sprintf("Failed to unmarshal video pricing config: %v", err))
		return configCache // 解析失败时返回缓存
	}

	configCache = config
	lastCacheTime = common.GetTimestamp()
	return configCache
}

func matchCondition(rule PricingRule, params VideoPricingParams) bool {
	if len(rule.Conditions.Resolution) > 0 {
		found := false
		resolution := strings.ToLower(params.Resolution)
		for _, r := range rule.Conditions.Resolution {
			if strings.ToLower(r) == resolution {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.Conditions.Duration) > 0 {
		found := false
		for _, d := range rule.Conditions.Duration {
			if d == params.Duration {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(rule.Conditions.ReferenceTypes) > 0 {
		for _, allowedType := range rule.Conditions.ReferenceTypes {
			found := false
			for _, refType := range params.ReferenceTypes {
				if refType == allowedType {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}

	if rule.Conditions.AudioOutput != "" && rule.Conditions.AudioOutput != params.AudioOutput {
		return false
	}

	return true
}

func Estimate(params VideoPricingParams) (float64, error) {
	config := GetConfig()
	modelPricing, ok := config.Models[params.Model]
	common.SysLog(fmt.Sprintf("params: %v", params))
	common.SysLog(fmt.Sprintf("config: %v", config))

	if !ok {
		return 0, fmt.Errorf("model %s not found in pricing config", params.Model)
	}

	var matchedRule *PricingRule
	for i := range modelPricing.PricingRules {
		if matchCondition(modelPricing.PricingRules[i], params) {
			matchedRule = &modelPricing.PricingRules[i]
			break
		}
	}

	if matchedRule == nil {
		return 0, fmt.Errorf("no pricing rule matched for %s", params.Model)
	}

	common.SysLog(fmt.Sprintf("matchedRule: resolution=%v duration=%v reference_types=%v price=%.4f input_price=%.4f output_price=%.4f",
		matchedRule.Conditions.Resolution,
		matchedRule.Conditions.Duration,
		matchedRule.Conditions.ReferenceTypes,
		matchedRule.Price,
		matchedRule.InputPricePerMToken,
		matchedRule.OutputPricePerMToken))

	var calculatedPrice float64
	switch modelPricing.BillingMode {
	case BillingModePerCall:
		calculatedPrice = matchedRule.Price
	case BillingModePerSecond:
		calculatedPrice = matchedRule.Price * float64(params.Duration)
	case BillingModePerToken:
		calculatedPrice = (float64(params.InputTokens)/1000000*matchedRule.InputPricePerMToken +
			float64(params.OutputTokens)/1000000*matchedRule.OutputPricePerMToken)
	default:
		return 0, fmt.Errorf("unknown billing mode: %s", modelPricing.BillingMode)
	}

	markup := modelPricing.Markup
	if markup <= 0 {
		markup = 1.0
	}

	finalPrice := calculatedPrice * markup

	if modelPricing.BasePrice <= 0 {
		return 0, fmt.Errorf("base price cannot be zero or negative")
	}

	var ratio float64
	switch modelPricing.BillingMode {
	case BillingModePerCall:
		ratio = finalPrice / modelPricing.BasePrice
	case BillingModePerSecond:
		ratio = finalPrice / modelPricing.BasePrice
	case BillingModePerToken:
		totalTokens := params.InputTokens + params.OutputTokens
		if totalTokens == 0 {
			return 0, fmt.Errorf("no tokens")
		}
		ratio = finalPrice * 1000000 / (modelPricing.BasePrice * float64(totalTokens))
	}
	common.SysLog(fmt.Sprintf("最终ratio: %f", ratio))
	// os.Exit(0)
	return ratio, nil
}

func EstimatePrice(params VideoPricingParams) (float64, error) {
	config := GetConfig()
	modelPricing, ok := config.Models[params.Model]
	if !ok {
		return 0, fmt.Errorf("model %s not found in pricing config", params.Model)
	}

	matchedRule := findMatchedRule(modelPricing.PricingRules, params)
	if matchedRule == nil {
		if modelPricing.BillingMode == BillingModePerCall || modelPricing.BillingMode == BillingModePerSecond {
			common.SysLog(fmt.Sprintf("EstimatePrice: no rule matched for %s, using base_price as fallback", params.Model))
			markup := modelPricing.Markup
			if markup <= 0 {
				markup = 1.0
			}
			return modelPricing.BasePrice * markup, nil
		}
		return 0, fmt.Errorf("no pricing rule matched for %s", params.Model)
	}

	var calculatedPrice float64
	switch modelPricing.BillingMode {
	case BillingModePerCall:
		calculatedPrice = matchedRule.Price
	case BillingModePerSecond:
		calculatedPrice = matchedRule.Price * float64(params.Duration)
	case BillingModePerToken:
		calculatedPrice = float64(params.InputTokens)/1000000*matchedRule.InputPricePerMToken +
			float64(params.OutputTokens)/1000000*matchedRule.OutputPricePerMToken
	default:
		return 0, fmt.Errorf("unknown billing mode: %s", modelPricing.BillingMode)
	}

	markup := modelPricing.Markup
	if markup <= 0 {
		markup = 1.0
	}

	finalPrice := calculatedPrice * markup
	common.SysLog(fmt.Sprintf("EstimatePrice: model=%s, calculatedPrice=%.4f, markup=%.2f, finalPrice=%.4f",
		params.Model, calculatedPrice, markup, finalPrice))

	return finalPrice, nil
}

func findMatchedRule(rules []PricingRule, params VideoPricingParams) *PricingRule {
	for i := range rules {
		if matchCondition(rules[i], params) {
			return &rules[i]
		}
	}
	return nil
}

func GetModelPricing(model string) (ModelPricing, bool) {
	config := GetConfig()

	modelPricing, ok := config.Models[model]
	if !ok {
		return ModelPricing{}, false
	}

	return modelPricing, true
}

func GetDefaultBasePrice(model string) float64 {
	modelPricing, ok := GetModelPricing(model)
	if !ok {
		return 2.0
	}
	if modelPricing.BasePrice > 0 {
		return modelPricing.BasePrice
	}
	return 2.0
}

func Init() {
	LoadConfig()
}

func GetReferenceTypesFromRequest(inputData map[string]interface{}) []ReferenceType {
	var types []ReferenceType

	if inputData == nil {
		return types
	}

	if _, ok := inputData["image"]; ok {
		types = append(types, ReferenceImage)
	}
	if _, ok := inputData["video"]; ok {
		types = append(types, ReferenceVideo)
	}
	if _, ok := inputData["audio"]; ok {
		types = append(types, ReferenceAudio)
	}

	return types
}

func CalculateTokens(text string, imageCount, videoCount, audioCount int) (int, int) {
	inputTokens := len(text) / 4
	inputTokens += imageCount * 768
	inputTokens += videoCount * 1000
	inputTokens += audioCount * 500

	outputTokens := 0

	return inputTokens, outputTokens
}