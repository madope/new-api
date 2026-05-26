# 通用视频定价框架 (VideoPricing Framework)

## 概述

本框架提供了一个**通用、可扩展**的视频模型定价解决方案，支持多维度定价和灵活的计费模式。

### 核心特性

| 特性 | 说明 |
|------|------|
| **多维度定价** | 支持按分辨率、时长、参考类型等维度定价 |
| **多种计费模式** | 支持按次计费 (per_call) 和按 token 计费 (per_token) |
| **价格上浮控制** | 支持全局上浮比例配置 |
| **渠道隔离** | 不同渠道可以有独立的定价策略 |
| **配置驱动** | 定价规则存储在数据库，无需重新部署 |
| **易于扩展** | 新增渠道只需添加配置，无需修改代码 |

---

## 架构设计

```
┌─────────────────────────────────────────────────────────────────┐
│                      VideoPricing Framework                    │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────┐    ┌─────────────────────┐           │
│  │   Pricing Config    │    │   Channel Adaptor   │           │
│  │  (数据库 options)   │    │  (各渠道实现)       │           │
│  └──────────┬──────────┘    └──────────┬──────────┘           │
│             │                          │                       │
│             ▼                          ▼                       │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │              VideoPricingCore (核心定价引擎)              │  │
│  │  - 参数解析: 分辨率/时长/参考类型/Token数                   │  │
│  │  - 条件匹配: 根据定价规则匹配价格                          │  │
│  │  - 价格计算: 按次/按Token两种模式                         │  │
│  │  - 倍率计算: 上浮比例 + 分组比率                          │  │
│  └──────────────────────────────────────────────────────────┘  │
│                              │                                 │
│                              ▼                                 │
│  ┌──────────────────────────────────────────────────────────┐  │
│  │                   OtherRatios 输出                       │  │
│  │  返回 {video_ratio: ratio} 供计费系统使用               │  │
│  └──────────────────────────────────────────────────────────┘  │
└───────────────────────────────────────────────────────────────┘
```

---

## 快速开始

### 1. 初始化配置

在数据库 `options` 表中添加配置：

```sql
INSERT INTO options (key, value) VALUES ('video_pricing_config', '{
  "channels": {
    "minimax": {
      "models": {
        "MiniMax-Hailuo-02": {
          "billing_mode": "per_call",
          "base_price": 2.00,
          "markup": 1.15,
          "pricing_rules": [
            {
              "conditions": {
                "resolution": ["512p"],
                "duration": [6],
                "reference_types": ["image"]
              },
              "price": 0.60
            },
            {
              "conditions": {
                "resolution": ["768p"],
                "duration": [6, 10],
                "reference_types": ["image", "video"]
              },
              "price": 2.00
            }
          ]
        }
      }
    }
  }
}');
```

### 2. 在渠道 Adaptor 中集成

```go
func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
    adaptor := videopricing.NewCustomVideoPricingAdaptor("minimax")
    return adaptor.EstimateBilling(c, info)
}
```

---

## 定价维度

| 维度 | 类型 | 说明 | 示例 |
|------|------|------|------|
| **channel** | string | 渠道标识 | minimax, doubao, suno |
| **model** | string | 模型名称 | MiniMax-Hailuo-02 |
| **resolution** | string | 分辨率 | 512p, 768p, 1080p, 4k |
| **duration** | int | 时长(秒) | 6, 10, 15 |
| **reference_types** | []string | 参考类型 | image, video, audio |
| **input_tokens** | int | 输入 token 数 | 用于按 token 计费 |
| **output_tokens** | int | 输出 token 数 | 用于按 token 计费 |
| **billing_mode** | string | 计费模式 | per_call, per_token |

---

## 配置结构详解

### 完整配置示例

```json
{
  "channels": {
    "minimax": {
      "models": {
        "MiniMax-Hailuo-02": {
          "billing_mode": "per_call",
          "base_price": 2.00,
          "markup": 1.15,
          "pricing_rules": [
            {
              "conditions": {
                "resolution": ["512p"],
                "duration": [6],
                "reference_types": ["image"]
              },
              "price": 0.60
            },
            {
              "conditions": {
                "resolution": ["768p"],
                "duration": [6, 10],
                "reference_types": ["image", "video"]
              },
              "price": 2.00
            }
          ]
        },
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
    },
    "doubao": {
      "models": {
        "Doubao-Video-Pro": {
          "billing_mode": "per_call",
          "base_price": 3.00,
          "markup": 1.0,
          "pricing_rules": [
            {
              "conditions": {
                "resolution": ["720p", "1080p"],
                "duration": [5, 10, 15],
                "reference_types": ["image", "video", "audio"]
              },
              "price": 3.00
            }
          ]
        }
      }
    }
  }
}
```

### 配置字段说明

| 字段 | 类型 | 说明 |
|------|------|------|
| `channels` | object | 渠道配置集合 |
| `channels[channel_name]` | object | 具体渠道配置 |
| `models` | object | 模型配置集合 |
| `models[model_name].billing_mode` | string | 计费模式：`per_call` 或 `per_token` |
| `models[model_name].base_price` | float | 基础价格（用于计算倍率） |
| `models[model_name].markup` | float | 价格上浮比例（如 1.15 表示上浮 15%） |
| `models[model_name].pricing_rules` | array | 定价规则列表 |
| `pricing_rules[].conditions` | object | 匹配条件 |
| `pricing_rules[].conditions.resolution` | array | 分辨率列表 |
| `pricing_rules[].conditions.duration` | array | 时长列表（秒） |
| `pricing_rules[].conditions.reference_types` | array | 参考类型列表 |
| `pricing_rules[].price` | float | 按次计费时的价格 |
| `pricing_rules[].input_price_per_token` | float | 按 token 计费时输入单价 |
| `pricing_rules[].output_price_per_token` | float | 按 token 计费时输出单价 |

---

## 计费计算逻辑

### 按次计费 (per_call)

```
最终价格 = 规则价格 × 上浮比例 (markup)
倍率 = 最终价格 / 基础价格 (base_price)
```

**示例**：
- 规则价格：2.00 元
- 上浮比例：1.15
- 基础价格：2.00 元

```
最终价格 = 2.00 × 1.15 = 2.30 元
倍率 = 2.30 / 2.00 = 1.15
```

### 按 Token 计费 (per_token)

```
最终价格 = (输入 Token 数 × 输入单价 + 输出 Token 数 × 输出单价) × 上浮比例 (markup)
倍率 = 最终价格 / 基础价格 (base_price)
```

**示例**：
- 输入 Token：1000
- 输出 Token：500
- 输入单价：0.0005 元/Token
- 输出单价：0.002 元/Token
- 上浮比例：1.0
- 基础价格：0.001 元

```
最终价格 = (1000 × 0.0005 + 500 × 0.002) × 1.0 = 1.50 元
倍率 = 1.50 / 0.001 = 1500
```

---

## API 接口

### 核心函数

| 函数 | 说明 | 参数 | 返回值 |
|------|------|------|--------|
| `Estimate(params)` | 计算定价倍率 | `VideoPricingParams` | `(ratio float64, error)` |
| `LoadConfig()` | 重新加载配置 | 无 | `error` |
| `GetModelPricing(channel, model)` | 获取模型定价配置 | channel, model | `(ModelPricing, bool)` |
| `GetDefaultBasePrice(channel, model)` | 获取默认基础价格 | channel, model | `float64` |
| `ValidatePricingConfig(config)` | 验证配置格式 | config string | `error` |

### 适配层接口

| 接口 | 说明 |
|------|------|
| `VideoPricingAdaptor` | 视频定价适配接口 |
| `GetChannel()` | 获取渠道名称 |
| `ParseParams(c, info)` | 解析请求参数 |
| `EstimateBilling(c, info)` | 估算计费 |

---

## 扩展新渠道

### 步骤 1：添加配置

```sql
UPDATE options SET value = '{
  "channels": {
    "new_channel": {
      "models": {
        "New-Video-Model": {
          "billing_mode": "per_call",
          "base_price": 5.00,
          "markup": 1.0,
          "pricing_rules": [
            {
              "conditions": {
                "resolution": ["1080p"],
                "duration": [10],
                "reference_types": ["image"]
              },
              "price": 5.00
            }
          ]
        }
      }
    }
  }
}' WHERE key = 'video_pricing_config';
```

### 步骤 2：在渠道 Adaptor 中集成

```go
func (a *NewChannelAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
    adaptor := videopricing.NewCustomVideoPricingAdaptor("new_channel")
    return adaptor.EstimateBilling(c, info)
}
```

---

## MiniMax 完整配置示例

```sql
INSERT INTO options (key, value) VALUES ('video_pricing_config', '{
  "channels": {
    "minimax": {
      "models": {
        "MiniMax-Hailuo-2.3-Fast": {
          "billing_mode": "per_call",
          "base_price": 1.35,
          "markup": 1.15,
          "pricing_rules": [
            {
              "conditions": {"resolution": ["768p"], "duration": [6], "reference_types": ["image"]},
              "price": 1.35
            },
            {
              "conditions": {"resolution": ["768p"], "duration": [10], "reference_types": ["image"]},
              "price": 2.25
            },
            {
              "conditions": {"resolution": ["1080p"], "duration": [6], "reference_types": ["image"]},
              "price": 2.31
            }
          ]
        },
        "MiniMax-Hailuo-2.3": {
          "billing_mode": "per_call",
          "base_price": 2.00,
          "markup": 1.15,
          "pricing_rules": [
            {
              "conditions": {"resolution": ["768p"], "duration": [6], "reference_types": ["image", "video"]},
              "price": 2.00
            },
            {
              "conditions": {"resolution": ["768p"], "duration": [10], "reference_types": ["image", "video"]},
              "price": 4.00
            },
            {
              "conditions": {"resolution": ["1080p"], "duration": [6], "reference_types": ["image", "video"]},
              "price": 3.50
            }
          ]
        },
        "MiniMax-Hailuo-02": {
          "billing_mode": "per_call",
          "base_price": 2.00,
          "markup": 1.15,
          "pricing_rules": [
            {
              "conditions": {"resolution": ["512p"], "duration": [6], "reference_types": ["image"]},
              "price": 0.60
            },
            {
              "conditions": {"resolution": ["512p"], "duration": [10], "reference_types": ["image"]},
              "price": 1.00
            },
            {
              "conditions": {"resolution": ["768p"], "duration": [6], "reference_types": ["image", "video"]},
              "price": 2.00
            },
            {
              "conditions": {"resolution": ["768p"], "duration": [10], "reference_types": ["image", "video"]},
              "price": 4.00
            },
            {
              "conditions": {"resolution": ["1080p"], "duration": [6], "reference_types": ["image", "video"]},
              "price": 3.50
            }
          ]
        }
      }
    }
  }
}');
```

---

## 模型基础价格配置

在管理后台配置 `model_price`：

| 模型 | model_price (元) | 说明 |
|------|------------------|------|
| MiniMax-Hailuo-2.3-Fast | 1.35 | 768P 6s 的价格 |
| MiniMax-Hailuo-2.3 | 2.00 | 768P 6s 的价格 |
| MiniMax-Hailuo-02 | 2.00 | 768P 6s 的价格 |

---

## 测试

运行测试：

```bash
cd /Users/meng/Documents/project/mmodelapi
go test ./relay/helper/videopricing -v
```

---

## 文件结构

```
relay/helper/videopricing/
├── core.go          # 核心定价引擎
├── adaptor.go       # 渠道适配层
├── config.go        # 配置工具函数
└── core_test.go     # 单元测试
```

---

## 扩展建议

### 1. 添加更多定价维度

可以扩展 `VideoPricingParams` 结构体添加新的维度：

```go
type VideoPricingParams struct {
    Channel        string
    Model          string
    Resolution     string
    Duration       int
    ReferenceTypes []ReferenceType
    InputTokens    int
    OutputTokens   int
    AspectRatio    string    // 新增：宽高比
    Quality        string    // 新增：画质等级
    Style          string    // 新增：风格类型
}
```

### 2. 添加时段定价

可以在配置中添加时段规则：

```json
{
  "pricing_rules": [
    {
      "conditions": {
        "resolution": ["768p"],
        "duration": [6],
        "time_range": ["09:00-18:00"]
      },
      "price": 2.00
    },
    {
      "conditions": {
        "resolution": ["768p"],
        "duration": [6],
        "time_range": ["18:00-24:00"]
      },
      "price": 2.50
    }
  ]
}
```

### 3. 添加用户等级定价

可以在配置中添加用户等级规则：

```json
{
  "pricing_rules": [
    {
      "conditions": {
        "resolution": ["768p"],
        "duration": [6],
        "user_level": ["vip"]
      },
      "price": 1.80
    },
    {
      "conditions": {
        "resolution": ["768p"],
        "duration": [6],
        "user_level": ["normal"]
      },
      "price": 2.00
    }
  ]
}
```

---

## 总结

本框架提供了一个**通用、可扩展**的视频模型定价解决方案：

| 特性 | 实现方式 |
|------|---------|
| **多维度定价** | 支持分辨率、时长、参考类型 |
| **多种计费模式** | 按次 (per_call) / 按 Token (per_token) |
| **价格上浮控制** | 通过 markup 配置项 |
| **渠道隔离** | 每个渠道独立配置 |
| **配置驱动** | 存储在数据库，无需重新部署 |
| **易于扩展** | 新增渠道只需添加配置 |

---