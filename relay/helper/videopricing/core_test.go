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
				ReferenceTypes: []ReferenceType{ReferenceImage},
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