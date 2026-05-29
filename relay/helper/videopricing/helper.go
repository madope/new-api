package videopricing

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
)

func ComputeVideoRatio(modelName string, params VideoPricingParams) (float64, error) {
	modelPricing, ok := GetModelPricing(modelName)
	if !ok {
		return 0, fmt.Errorf("model %s not found in video pricing config", modelName)
	}

	switch modelPricing.BillingMode {
	case BillingModePerCall:
		// 按次计费：计算预估价格并求比率
		estimatePrice, err := EstimatePrice(params)
		if err != nil {
			return 0, fmt.Errorf("estimate price failed: %w", err)
		}

		modelPrice, success := ratio_setting.GetModelPrice(modelName, true)
		if !success {
			modelPrice = modelPricing.BasePrice
		}

		if modelPrice <= 0 {
			return 0, fmt.Errorf("invalid base price: %f", modelPrice)
		}

		ratio := estimatePrice / modelPrice
		common.SysLog(fmt.Sprintf("ComputeVideoRatio (per_call): model=%s, estimatePrice=%.4f, modelPrice=%.4f, ratio=%.4f",
			modelName, estimatePrice, modelPrice, ratio))
		return ratio, nil

	case BillingModePerSecond:
		estimatePrice, err := EstimatePrice(params)
		if err != nil {
			return 0, fmt.Errorf("estimate price failed: %w", err)
		}

		modelPrice, success := ratio_setting.GetModelPrice(modelName, true)
		if !success {
			modelPrice = modelPricing.BasePrice
		}

		if modelPrice <= 0 {
			return 0, fmt.Errorf("invalid base price: %f", modelPrice)
		}

		ratio := estimatePrice / modelPrice
		common.SysLog(fmt.Sprintf("ComputeVideoRatio (per_second): model=%s, estimatePrice=%.4f, modelPrice=%.4f, ratio=%.4f",
			modelName, estimatePrice, modelPrice, ratio))
		return ratio, nil

	case BillingModePerToken:
		// 按Token计费：复用new-api原有机制
		// 从QuotaPerUnit实时计算系统基准价格
		// QuotaPerUnit = $0.002 / 1K tokens = 500,000 quota
		// 1M tokens = $0.002 × 1000 = $2 = 500,000,000 quota
		// 系统基准价格 ($/M token) = (QuotaPerUnit / 500000) × 2
		systemBasePrice := (common.QuotaPerUnit / common.QuotaPerUnit) * 2.0

		if modelPricing.BasePrice <= 0 {
			return 0, fmt.Errorf("invalid base price: %f", modelPricing.BasePrice)
		}

		ratio := (modelPricing.BasePrice / systemBasePrice) * 0.5
		common.SysLog(fmt.Sprintf("ComputeVideoRatio (per_token): model=%s, base_price=$%.2f/M token, system_base_price=$%.2f/M token, ratio=%.4f",
			modelName, modelPricing.BasePrice, systemBasePrice, ratio))
		return ratio, nil

	default:
		return 0, fmt.Errorf("unknown billing mode: %s", modelPricing.BillingMode)
	}
}
