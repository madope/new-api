package videopricing

import (
	"math"
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestEstimatePerCall(t *testing.T) {
	configStr, _ := GenerateMiniMaxConfig()
	common.OptionMap = map[string]string{VideoPricingConfigKey: configStr}
	LoadConfig()

	tests := []struct {
		name        string
		params      VideoPricingParams
		expected    float64
		expectError bool
	}{
		{
			name: "MiniMax-Hailuo-02 512p 6s image",
			params: VideoPricingParams{
				Channel:        "minimax",
				Model:          "MiniMax-Hailuo-02",
				Resolution:     "512p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expected:    (0.60 * 1.15) / 2.00,
			expectError: false,
		},
		{
			name: "MiniMax-Hailuo-02 768p 10s image",
			params: VideoPricingParams{
				Channel:        "minimax",
				Model:          "MiniMax-Hailuo-02",
				Resolution:     "768p",
				Duration:       10,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expected:    (4.00 * 1.15) / 2.00,
			expectError: false,
		},
		{
			name: "MiniMax-Hailuo-2.3-Fast 768p 6s image",
			params: VideoPricingParams{
				Channel:        "minimax",
				Model:          "MiniMax-Hailuo-2.3-Fast",
				Resolution:     "768p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expected:    (1.35 * 1.15) / 1.35,
			expectError: false,
		},
		{
			name: "MiniMax-Hailuo-2.3-Fast 1080p 6s image",
			params: VideoPricingParams{
				Channel:        "minimax",
				Model:          "MiniMax-Hailuo-2.3-Fast",
				Resolution:     "1080p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expected:    (2.31 * 1.15) / 1.35,
			expectError: false,
		},
		{
			name: "Unknown model",
			params: VideoPricingParams{
				Channel:        "minimax",
				Model:          "Unknown-Model",
				Resolution:     "768p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio, err := Estimate(tt.params)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if math.Abs(ratio-tt.expected) > 0.0001 {
					t.Errorf("Expected ratio %.4f, got %.4f", tt.expected, ratio)
				}
			}
		})
	}
}

func TestEstimatePerToken(t *testing.T) {
	configStr := `{
		"models": {
			"MiniMax-Token-Model": {
				"billing_mode": "per_token",
				"base_price": 0.001,
				"markup": 1.0,
				"pricing_rules": [
					{
						"conditions": {
							"resolution": ["768p"]
						},
						"input_price_per_token": 0.0005,
						"output_price_per_token": 0.002
					}
				]
			}
		}
	}`

	common.OptionMap = map[string]string{VideoPricingConfigKey: configStr}
	LoadConfig()

	params := VideoPricingParams{
		Channel:        "minimax",
		Model:          "MiniMax-Token-Model",
		Resolution:     "768p",
		Duration:       6,
		ReferenceTypes: []ReferenceType{},
		InputTokens:    1000,
		OutputTokens:   500,
	}

	ratio, err := Estimate(params)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	expectedPrice := (1000*0.0005 + 500*0.002) * 1.0
	expectedRatio := expectedPrice / 0.001

	if ratio != expectedRatio {
		t.Errorf("Expected ratio %.4f, got %.4f", expectedRatio, ratio)
	}
}

func TestMatchCondition(t *testing.T) {
	rule := PricingRule{
		Conditions: PricingCondition{
			Resolution:     []string{"768p", "1080p"},
			Duration:       []int{6, 10},
			ReferenceTypes: []ReferenceType{ReferenceImage, ReferenceVideo},
		},
		Price: 2.00,
	}

	tests := []struct {
		name     string
		params   VideoPricingParams
		expected bool
	}{
		{
			name: "All conditions match",
			params: VideoPricingParams{
				Resolution:     "768p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceImage, ReferenceVideo},
			},
			expected: true,
		},
		{
			name: "Resolution mismatch",
			params: VideoPricingParams{
				Resolution:     "512p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expected: false,
		},
		{
			name: "Duration mismatch",
			params: VideoPricingParams{
				Resolution:     "768p",
				Duration:       15,
				ReferenceTypes: []ReferenceType{ReferenceImage},
			},
			expected: false,
		},
		{
			name: "Reference type mismatch",
			params: VideoPricingParams{
				Resolution:     "768p",
				Duration:       6,
				ReferenceTypes: []ReferenceType{ReferenceAudio},
			},
			expected: false,
		},
		{
			name: "No conditions",
			params: VideoPricingParams{
				Resolution:     "512p",
				Duration:       3,
				ReferenceTypes: []ReferenceType{},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchCondition(rule, tt.params)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetReferenceTypesFromRequest(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected []ReferenceType
	}{
		{
			name:     "Empty input",
			input:    nil,
			expected: []ReferenceType{},
		},
		{
			name: "Image only",
			input: map[string]interface{}{
				"image": "base64data",
			},
			expected: []ReferenceType{ReferenceImage},
		},
		{
			name: "Multiple references",
			input: map[string]interface{}{
				"image": "base64data",
				"video": "video_url",
				"audio": "audio_url",
			},
			expected: []ReferenceType{ReferenceImage, ReferenceVideo, ReferenceAudio},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetReferenceTypesFromRequest(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d types, got %d", len(tt.expected), len(result))
				return
			}
			for i, rt := range result {
				if rt != tt.expected[i] {
					t.Errorf("Expected %v, got %v", tt.expected[i], rt)
				}
			}
		})
	}
}

func TestValidatePricingConfig(t *testing.T) {
	tests := []struct {
		name        string
		configStr   string
		expectError bool
	}{
		{
			name:        "Valid config",
			configStr:   `{"models":{"test":{"billing_mode":"per_call","base_price":2.0,"markup":1.0,"pricing_rules":[{"conditions":{"resolution":["768p"]},"price":2.0}]}}}`,
			expectError: false,
		},
		{
			name:        "Empty models",
			configStr:   `{"models":{}}`,
			expectError: true,
		},
		{
			name:        "Invalid base price",
			configStr:   `{"models":{"test":{"billing_mode":"per_call","base_price":0,"markup":1.0,"pricing_rules":[{"conditions":{},"price":2.0}]}}}`,
			expectError: true,
		},
		{
			name:        "No pricing rules",
			configStr:   `{"models":{"test":{"billing_mode":"per_call","base_price":2.0,"markup":1.0,"pricing_rules":[]}}}`,
			expectError: true,
		},
		{
			name:        "Invalid JSON",
			configStr:   `invalid json`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePricingConfig(tt.configStr)
			if tt.expectError && err == nil {
				t.Error("Expected error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

const klingPerSecondConfig = `{
	"models": {
		"kling-3.0": {
			"billing_mode": "per_second",
			"base_price": 1.0,
			"markup": 1.0,
			"default_resolution": "720p",
			"default_duration": 5,
			"pricing_rules": [
				{"conditions":{"resolution":["720p"],"duration":[5],"reference_types":["text"]},"price":0.15},
				{"conditions":{"resolution":["1080p"],"duration":[5],"reference_types":["text"]},"price":0.21},
				{"conditions":{"resolution":["720p"],"duration":[10],"reference_types":["text"]},"price":0.12},
				{"conditions":{"duration":[5,10],"reference_types":["audio"],"audio_output":"none"},"price":0.18},
				{"conditions":{"duration":[5,10],"reference_types":["audio"],"audio_output":"voice"},"price":0.23},
				{"conditions":{"duration":[5,10],"reference_types":["audio"],"audio_output":"voice_timbre"},"price":0.28}
			]
		}
	}
}`

func TestEstimatePerSecond(t *testing.T) {
	common.OptionMap = map[string]string{VideoPricingConfigKey: klingPerSecondConfig}
	LoadConfig()

	tests := []struct {
		name        string
		params      VideoPricingParams
		expected    float64
		expectError bool
	}{
		{
			name: "文生视频 720p 5s",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "720p",
				Duration:       5,
				ReferenceTypes: []ReferenceType{"text"},
			},
			// (0.15 * 5 * 1.0) / 1.0 = 0.75
			expected: (0.15 * 5 * 1.0) / 1.0,
		},
		{
			name: "文生视频 1080p 5s",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "1080p",
				Duration:       5,
				ReferenceTypes: []ReferenceType{"text"},
			},
			expected: (0.21 * 5 * 1.0) / 1.0,
		},
		{
			name: "文生视频 720p 10s",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "720p",
				Duration:       10,
				ReferenceTypes: []ReferenceType{"text"},
			},
			expected: (0.12 * 10 * 1.0) / 1.0,
		},
		{
			name: "音频生视频 无声 5s",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "720p",
				Duration:       5,
				ReferenceTypes: []ReferenceType{"audio"},
				AudioOutput:    "none",
			},
			expected: (0.18 * 5 * 1.0) / 1.0,
		},
		{
			name: "音频生视频 有声 5s",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "720p",
				Duration:       5,
				ReferenceTypes: []ReferenceType{"audio"},
				AudioOutput:    "voice",
			},
			expected: (0.23 * 5 * 1.0) / 1.0,
		},
		{
			name: "音频生视频 有声+有音色 5s",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "720p",
				Duration:       5,
				ReferenceTypes: []ReferenceType{"audio"},
				AudioOutput:    "voice_timbre",
			},
			expected: (0.28 * 5 * 1.0) / 1.0,
		},
		{
			name: "无匹配规则返回错误",
			params: VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "4k",
				Duration:       5,
				ReferenceTypes: []ReferenceType{"text"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ratio, err := Estimate(tt.params)
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if math.Abs(ratio-tt.expected) > 0.0001 {
					t.Errorf("Expected ratio %.4f, got %.4f", tt.expected, ratio)
				}
			}
		})
	}
}

func TestMatchConditionAudioOutput(t *testing.T) {
	ruleWithVoice := PricingRule{
		Conditions: PricingCondition{
			ReferenceTypes: []ReferenceType{"text"},
			AudioOutput:    "voice",
		},
		Price: 2.0,
	}
	ruleWithoutAudio := PricingRule{
		Conditions: PricingCondition{
			ReferenceTypes: []ReferenceType{"text"},
			AudioOutput:    "",
		},
		Price: 2.0,
	}

	tests := []struct {
		name     string
		rule     PricingRule
		params   VideoPricingParams
		expected bool
	}{
		{
			name: "规则有voice,参数有voice — 匹配",
			rule: ruleWithVoice,
			params: VideoPricingParams{
				ReferenceTypes: []ReferenceType{"text"},
				AudioOutput:    "voice",
			},
			expected: true,
		},
		{
			name: "规则有voice,参数有none — 不匹配",
			rule: ruleWithVoice,
			params: VideoPricingParams{
				ReferenceTypes: []ReferenceType{"text"},
				AudioOutput:    "none",
			},
			expected: false,
		},
		{
			name: "规则有voice,参数有voice_timbre — 不匹配",
			rule: ruleWithVoice,
			params: VideoPricingParams{
				ReferenceTypes: []ReferenceType{"text"},
				AudioOutput:    "voice_timbre",
			},
			expected: false,
		},
		{
			name: "规则有none,参数有voice — 不匹配",
			rule: PricingRule{
				Conditions: PricingCondition{
					ReferenceTypes: []ReferenceType{"text"},
					AudioOutput:    "none",
				},
				Price: 2.0,
			},
			params: VideoPricingParams{
				ReferenceTypes: []ReferenceType{"text"},
				AudioOutput:    "voice",
			},
			expected: false,
		},
		{
			name: "规则无audio条件(空),参数有值 — 匹配",
			rule: ruleWithoutAudio,
			params: VideoPricingParams{
				ReferenceTypes: []ReferenceType{"text"},
				AudioOutput:    "none",
			},
			expected: true,
		},
		{
			name: "规则无audio条件(空),参数空 — 匹配",
			rule: ruleWithoutAudio,
			params: VideoPricingParams{
				ReferenceTypes: []ReferenceType{"text"},
				AudioOutput:    "",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchCondition(tt.rule, tt.params)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGenerateMiniMaxConfig(t *testing.T) {
	configStr, err := GenerateMiniMaxConfig()
	if err != nil {
		t.Errorf("Failed to generate config: %v", err)
	}

	err = ValidatePricingConfig(configStr)
	if err != nil {
		t.Errorf("Generated config is invalid: %v", err)
	}

	common.OptionMap = map[string]string{VideoPricingConfigKey: configStr}
	LoadConfig()

	params := VideoPricingParams{
		Channel:        "minimax",
		Model:          "MiniMax-Hailuo-02",
		Resolution:     "768p",
		Duration:       6,
		ReferenceTypes: []ReferenceType{ReferenceImage},
	}

	ratio, err := Estimate(params)
	if err != nil {
		t.Errorf("Failed to estimate: %v", err)
	}

	if ratio <= 0 {
		t.Errorf("Invalid ratio: %v", ratio)
	}
}