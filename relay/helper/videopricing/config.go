package videopricing

import (
	"encoding/json"
	"fmt"
	"strings"
)

func BuildPricingConfig(models []ModelPricingConfig) (string, error) {
	config := VideoPricingConfig{
		Models: make(map[string]ModelPricing),
	}

	for _, modelConfig := range models {
		modelPricing := ModelPricing{
			BillingMode:           BillingMode(modelConfig.BillingMode),
			BasePrice:             modelConfig.BasePrice,
			Markup:                modelConfig.Markup,
			DefaultResolution:     modelConfig.DefaultResolution,
			DefaultDuration:       modelConfig.DefaultDuration,
			DefaultReferenceTypes: toReferenceTypes(modelConfig.DefaultReferenceTypes),
			PricingRules:          make([]PricingRule, 0),
		}

		for _, ruleConfig := range modelConfig.PricingRules {
			referenceTypes := make([]ReferenceType, 0)
			for _, rt := range ruleConfig.ReferenceTypes {
				referenceTypes = append(referenceTypes, ReferenceType(rt))
			}

			rule := PricingRule{
				Conditions: PricingCondition{
					Resolution:     ruleConfig.Resolution,
					Duration:       ruleConfig.Duration,
					ReferenceTypes: referenceTypes,
				},
				Price:              ruleConfig.Price,
				InputPricePerMToken: ruleConfig.InputPricePerMToken,
				OutputPricePerMToken: ruleConfig.OutputPricePerMToken,
			}

			modelPricing.PricingRules = append(modelPricing.PricingRules, rule)
		}

		config.Models[modelConfig.ModelName] = modelPricing
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}

	return string(data), nil
}

type PricingRuleConfig struct {
	Resolution          []string  `json:"resolution"`
	Duration            []int     `json:"duration"`
	ReferenceTypes      []string  `json:"reference_types"`
	Price               float64   `json:"price"`
	InputPricePerMToken float64   `json:"input_price_per_m_token"`
	OutputPricePerMToken float64  `json:"output_price_per_m_token"`
}

type ModelPricingConfig struct {
	ModelName             string             `json:"model_name"`
	BillingMode           string             `json:"billing_mode"`
	BasePrice             float64            `json:"base_price"`              // 基础价格（元/M token）
	Markup                float64            `json:"markup"`
	DefaultResolution     string             `json:"default_resolution"`      // 默认分辨率，空时回落 "768p"
	DefaultDuration       int                `json:"default_duration"`        // 默认时长（秒），0 时回落 6
	DefaultReferenceTypes []string           `json:"default_reference_types"` // 默认参考类型，仅用于展示
	PricingRules          []PricingRuleConfig `json:"pricing_rules"`
}

func GenerateMiniMaxConfig() (string, error) {
	models := []ModelPricingConfig{
			{
				ModelName:   "MiniMax-Hailuo-2.3-Fast",
				BillingMode: "per_call",
				BasePrice:   1.35,
				Markup:      1.15,
				PricingRules: []PricingRuleConfig{
					{
						Resolution:     []string{"768p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image"},
						Price:          1.35,
					},
					{
						Resolution:     []string{"768p"},
						Duration:       []int{10},
						ReferenceTypes: []string{"image"},
						Price:          2.25,
					},
					{
						Resolution:     []string{"1080p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image"},
						Price:          2.31,
					},
				},
			},
			{
				ModelName:   "MiniMax-Hailuo-2.3",
				BillingMode: "per_call",
				BasePrice:   2.00,
				Markup:      1.15,
				PricingRules: []PricingRuleConfig{
					{
						Resolution:     []string{"768p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image", "video"},
						Price:          2.00,
					},
					{
						Resolution:     []string{"768p"},
						Duration:       []int{10},
						ReferenceTypes: []string{"image", "video"},
						Price:          4.00,
					},
					{
						Resolution:     []string{"1080p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image", "video"},
						Price:          3.50,
					},
				},
			},
			{
				ModelName:   "MiniMax-Hailuo-02",
				BillingMode: "per_call",
				BasePrice:   2.00,
				Markup:      1.15,
				PricingRules: []PricingRuleConfig{
					{
						Resolution:     []string{"512p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image"},
						Price:          0.60,
					},
					{
						Resolution:     []string{"512p"},
						Duration:       []int{10},
						ReferenceTypes: []string{"image"},
						Price:          1.00,
					},
					{
						Resolution:     []string{"768p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image", "video"},
						Price:          2.00,
					},
					{
						Resolution:     []string{"768p"},
						Duration:       []int{10},
						ReferenceTypes: []string{"image", "video"},
						Price:          4.00,
					},
					{
						Resolution:     []string{"1080p"},
						Duration:       []int{6},
						ReferenceTypes: []string{"image", "video"},
						Price:          3.50,
					},
				},
			},
	}

	return BuildPricingConfig(models)
}

func GenerateDoubaoConfig() (string, error) {
	models := []ModelPricingConfig{
		{
			ModelName:   "Doubao-Video-Pro",
			BillingMode: "per_call",
			BasePrice:   3.00,
			Markup:      1.0,
			PricingRules: []PricingRuleConfig{
				{
					Resolution:     []string{"720p", "1080p"},
					Duration:       []int{5, 10, 15},
					ReferenceTypes: []string{"image", "video", "audio"},
					Price:          3.00,
				},
			},
		},
	}

	return BuildPricingConfig(models)
}

func ValidatePricingConfig(configStr string) error {
	var config VideoPricingConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if len(config.Models) == 0 {
		return fmt.Errorf("no models defined")
	}

	for modelName, modelPricing := range config.Models {
		if modelPricing.BasePrice <= 0 {
			return fmt.Errorf("model %s has invalid base price", modelName)
		}

		if len(modelPricing.PricingRules) == 0 {
			return fmt.Errorf("model %s has no pricing rules", modelName)
		}

		for i, rule := range modelPricing.PricingRules {
			if modelPricing.BillingMode == BillingModePerCall && rule.Price <= 0 {
				return fmt.Errorf("model %s rule %d has invalid price", modelName, i)
			}
			if modelPricing.BillingMode == BillingModePerToken && (rule.InputPricePerMToken <= 0 || rule.OutputPricePerMToken <= 0) {
				return fmt.Errorf("model %s rule %d has invalid token price", modelName, i)
			}
		}
	}

	return nil
}

func GetConfigSummary(configStr string) (string, error) {
	var config VideoPricingConfig
	if err := json.Unmarshal([]byte(configStr), &config); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	var summary strings.Builder
	summary.WriteString("Video Pricing Config Summary:\n")
	summary.WriteString("=============================\n\n")

	summary.WriteString(fmt.Sprintf("Models: %d\n", len(config.Models)))

	for modelName, modelPricing := range config.Models {
		summary.WriteString(fmt.Sprintf("  - %s\n", modelName))
		summary.WriteString(fmt.Sprintf("      Billing Mode: %s\n", modelPricing.BillingMode))
		summary.WriteString(fmt.Sprintf("      Base Price: %.2f\n", modelPricing.BasePrice))
		summary.WriteString(fmt.Sprintf("      Markup: %.2f\n", modelPricing.Markup))
		summary.WriteString(fmt.Sprintf("      Rules: %d\n", len(modelPricing.PricingRules)))
	}

	summary.WriteString("\n")

	return summary.String(), nil
}

func toReferenceTypes(types []string) []ReferenceType {
	result := make([]ReferenceType, 0, len(types))
	for _, t := range types {
		result = append(result, ReferenceType(t))
	}
	return result
}