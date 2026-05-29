// Kling 视频生成 API 集成测试套件
//
// 基于视频定价规则生成测试用例，覆盖 4 个 MKling 接口:
//   1. POST /m-kling/v1/videos/text2video
//   2. POST /m-kling/v1/videos/image2video
//   3. POST /m-kling/v1/videos/multi-image2video
//   4. POST /m-kling/v1/videos/omni-video
//
// 每个用例验证:
//   - HTTP 200
//   - JSON body: code==0, task_id 非空
//   - X-New-Api-Other-Ratios 头中的 video_ratio 是否符合预期
//
// 运行方式（项目根目录执行）:
//
//   # 全部 Kling 用例（mock 模式，不真实扣费）
//   go test -v -run TestKling -kv-token 'sk-xxx' -kv-mock=true ./tests/
//
//   # 单个接口
//   go test -v -run 'TestKling/text2video' -kv-token 'sk-xxx' -kv-mock=true ./tests/
//
//   # 单个模型（正则匹配）
//   go test -v -run 'TestKling/text2video/kling_3_0' -kv-token 'sk-xxx' -kv-mock=true ./tests/
//
//   # 单条定价规则
//   go test -v -run 'TestKling/text2video/720p_voice' -kv-token 'sk-xxx' -kv-mock=true ./tests/
//
//
//   # kv-mock 默认 true，会在 URL 末尾追加 ?mock=true，不走真实上游
//   go test -v -run TestKling -kv-token 'sk-xxx' ./tests/
//
//   # -kv-mock=false 发起真实上游请求（会真实扣费，谨慎使用）
//   go test -v -run TestKling -kv-token 'sk-xxx' -kv-mock=false ./tests/
//
// Kling 官方 API 参数 ↔ 定价字段映射:
//
//   pricing.resolution=720p         -> API mode="std"
//   pricing.resolution=1080p        -> API mode="pro"
//   pricing.resolution=1440p/2k     -> API resolution="1440p"
//   pricing.resolution=2160p/4k     -> API mode="4k"
//   pricing.audio_output=none         -> 不传 sound（默认 off）
//   pricing.audio_output=voice        -> API sound="on"
//   pricing.audio_output=voice_timbre -> API sound="on" + voice_list [{voice_id:"..."}]
//   pricing.reference_types=[video]     -> API video_list [{video_url:..., refer_type:"feature", keep_original_sound:"no"}]
//   pricing.reference_types=[image]     -> API image_list=[{image:url1},{image:url2},...]（multi-image/omni 接口）
//   pricing.reference_types=[image]     -> API image="url"（image2video 接口）
//   pricing.audio_output=voice_timbre   -> API sound="on" + voice_list [{voice_id:"v001"}]
//   pricing.duration                     -> API duration（整数，默认 5）

package tests

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"strings"
	"testing"
)

// =============================================================================
// 命令行参数
// =============================================================================
// 优先级: 命令行 -kv-token > 环境变量 KLING_TOKEN > 下方默认值
// 可在 kvDefaultToken 中设置开发环境默认值，避免每次传参

const kvDefaultToken = "sk-AaN5sG2MkVOFV5MkVMW1fGm31AvoLjTlod3TGQpj7LzEUZKp"    // ← 在这里设置默认 token（开发环境用）
const kvDefaultBaseURL = "http://localhost:3001" // ← 默认 API 地址

var kvToken = flag.String("kv-token", kvResolveToken(), "API token (sk-xxx 或 accessKey|secretKey)")
var kvBase = flag.String("kv-base-url", kvDefaultBaseURL, "API 地址")
var kvMock = flag.Bool("kv-mock", true, "启用 mock 模式（请求体添加 mock:true，避免真实扣费）")

func kvResolveToken() string {
	if t := os.Getenv("KLING_TOKEN"); t != "" {
		return t
	}
	return kvDefaultToken
}

// =============================================================================
// 测试用资源 URL
// =============================================================================

var (
	kvi1 = "https://img1.baidu.com/it/u=1453953792,4069179488&fm=253&fmt=auto&app=138&f=JPEG?w=800&h=500"
	kvi2 = "https://picx.zhimg.com/v2-202ccd64c07348762401c43f2febd65c_r.jpg?source=2c26e567"
	kvi3 = "https://c-ssl.dtstatic.com/uploads/item/201806/26/20180626143653_skurw.thumb.1000_0.jpg"
	kvv  = "https://example.com/ref-video.mp4"
)

// =============================================================================
// 计费预期
// =============================================================================

type kvBE struct {
	BP float64 // 模型基础价格（元/秒）
	MP float64 // 匹配规则价格（元/秒）
	D  int     // 视频时长（秒）
	M  float64 // 倍率系数
	DS string  // 规则中文说明
}

// 预期 video_ratio = (MP × D × M) / BP
func (b kvBE) er() float64 { return (b.MP * float64(b.D) * b.M) / b.BP }

// 预期扣费（元）= MP × D × M
func (b kvBE) ec() float64 { return b.MP * float64(b.D) * b.M }

// =============================================================================
// 测试用例
// =============================================================================

type kvtc struct {
	N string         // 测试名称
	P string         // 请求路径
	B map[string]any // 请求参数（Kling 官方格式）
	C int            // 预期 HTTP 状态码
	E kvBE           // 计费预期
}

// =============================================================================
// 核心 runner
// =============================================================================

func kvRun(t *testing.T, tc kvtc) {
	t.Helper()

	expectedVR := tc.E.er()
	expectedCost := tc.E.ec()

	ratioCal := ""
	if tc.E.BP > 0 {
		ratioCal = fmt.Sprintf("(%.2f × %d × %.2f) ÷ %.2f = %.4f",
			tc.E.MP, tc.E.D, tc.E.M, tc.E.BP, expectedVR)
	}

	// 计费预期信息（始终计算）
	billingInfo := map[string]any{}
	if tc.E.BP > 0 {
		billingInfo = map[string]any{
			"模型基础价格(base_price)":   tc.E.BP,
			"匹配规则价格(matched_price)": tc.E.MP,
			"视频时长(秒)":             tc.E.D,
			"倍率系数(markup)":        tc.E.M,
			"规则说明":                tc.E.DS,
			"video_ratio计算公式":     ratioCal,
			"预期倍率(video_ratio)":   math.Round(expectedVR*10000) / 10000,
			"预期扣费(元)":            math.Round(expectedCost*10000) / 10000,
		}
	}

	// 响应信息
	respInfo := map[string]any{
		"HTTP状态码":  "未请求",
		"业务code":   "未请求",
		"业务message": "未请求",
		"任务ID(task_id)": "未请求",
	}

	allOK := true
	var failMsgs []string

	if *kvToken != "" {
		// 发送HTTP请求
		resp := kvDoReq(t, "POST", tc.P, tc.B)

		var bodyResp struct {
			Code    float64 `json:"code"`
			Message string  `json:"message"`
			Data    struct {
				TaskId string `json:"task_id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(resp.Body, &bodyResp)

		rh := resp.Header.Get("X-New-Api-Other-Ratios")
		hasVR := false
		actualVR := 0.0
		if rh != "" {
			var m map[string]float64
			if json.Unmarshal([]byte(rh), &m) == nil {
				actualVR = m["video_ratio"]
				hasVR = true
			}
		}

		respInfo = map[string]any{
			"HTTP状态码":    resp.StatusCode,
			"业务code":     bodyResp.Code,
			"业务message":  bodyResp.Message,
			"任务ID(task_id)": bodyResp.Data.TaskId,
		}

		// 填充实际倍率
		if tc.E.BP > 0 && hasVR {
			billingInfo["实际倍率(video_ratio)"] = math.Round(actualVR*10000) / 10000
			billingInfo["倍率匹配"] = math.Abs(actualVR-expectedVR) < 0.01
		} else if tc.E.BP > 0 {
			billingInfo["实际倍率(video_ratio)"] = "未返回"
			billingInfo["倍率匹配"] = "N/A"
		}

		// 判定
		httpOK := resp.StatusCode == tc.C
		codeOK := bodyResp.Code == 0
		taskOK := bodyResp.Data.TaskId != ""
		ratioMatch := !hasVR || tc.E.BP <= 0 || math.Abs(actualVR-expectedVR) < 0.01

		allOK = httpOK && codeOK && taskOK
		if hasVR && tc.E.BP > 0 {
			allOK = allOK && ratioMatch
		}

		if !allOK {
			if !httpOK { failMsgs = append(failMsgs, fmt.Sprintf("HTTP状态码期望%d实际%d", tc.C, resp.StatusCode)) }
			if !codeOK { failMsgs = append(failMsgs, fmt.Sprintf("业务code期望0实际%.0f", bodyResp.Code)) }
			if !taskOK { failMsgs = append(failMsgs, "task_id为空") }
			if hasVR && tc.E.BP > 0 && !ratioMatch { failMsgs = append(failMsgs, fmt.Sprintf("倍率期望%.4f实际%.4f", expectedVR, actualVR)) }
		}

		// Go断言
		if !httpOK { t.Errorf("❌ HTTP状态码异常: 期望%d实际%d", tc.C, resp.StatusCode) }
		if !codeOK { t.Errorf("❌ 业务code异常: 期望0实际%.0f message=%s", bodyResp.Code, bodyResp.Message) }
		if !taskOK { t.Error("❌ task_id为空") }
		if hasVR && tc.E.BP > 0 && !ratioMatch { t.Errorf("❌ 倍率不匹配: 期望%.4f实际%.4f", expectedVR, actualVR) }
		if allOK { t.Log("✅ 测试通过") }
	}

	// 输出单一JSON
	out := map[string]any{
		"测试用例": tc.N,
		"请求接口": fmt.Sprintf("POST %s", tc.P),
		"请求参数(Kling官方格式)": tc.B,
		"响应结果": respInfo,
		"计费验证": billingInfo,
	}
	if len(failMsgs) > 0 {
		out["失败原因"] = strings.Join(failMsgs, "；")
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	t.Log("\n" + string(b))
}

// =============================================================================
// HTTP 请求辅助
// =============================================================================

type kvResp struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

func kvDoReq(t *testing.T, method, path string, body map[string]any) *kvResp {
	t.Helper()
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("序列化请求体失败: %v", err)
		}
		r = bytes.NewReader(b)
	}
	url := *kvBase + path
	if *kvMock {
		if strings.ContainsRune(url, '?') {
			url += "&mock=true"
		} else {
			url += "?mock=true"
		}
	}
	out := map[string]any{
		"请求接口": fmt.Sprintf("%s %s", method, url),
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	t.Log("\n" + string(b))
	req, err := http.NewRequest(method, url, r)
	if err != nil {
		t.Fatalf("创建请求失败: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if *kvToken != "" {
		req.Header.Set("Authorization", "Bearer "+*kvToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("请求失败: %v", err)
	}
	defer resp.Body.Close()
	bodyBytes, _ := io.ReadAll(resp.Body)
	return &kvResp{StatusCode: resp.StatusCode, Header: resp.Header, Body: bodyBytes}
}

// =============================================================================
// 辅助：分辨率+价格描述
// =============================================================================

type kvRP struct {
	label string // 显示名
	mode  string // API mode 参数
	res   string // API resolution 参数
	price float64
	desc  string // 中文说明
}

var kvStdResPrices = []kvRP{
	{"720p", "std", "", 0.4, "720p → 0.4元/秒"},
	{"1080p", "pro", "", 0.7, "1080p → 0.7元/秒"},
	{"2k", "", "1440p", 1.0, "2k → 1.0元/秒"},
	{"4k", "4k", "", 1.5, "4k → 1.5元/秒"},
}

var kv25pResPrices = []kvRP{
	{"720p", "std", "", 0.3, "720p → 0.3元/秒"},
	{"1080p", "pro", "", 0.5, "1080p → 0.5元/秒"},
	{"2k", "", "1440p", 0.75, "2k → 0.75元/秒"},
	{"4k", "4k", "", 1.12, "4k → 1.12元/秒"},
}

// 测试用例数据类型（kling-2.6 / kling-3.0 / kling-o1 共用）
type kv26Case struct {
	label string
	rp    kvRP
	sound string
	mp    float64
	desc  string
}

type kv30Case struct {
	label  string
	rp     kvRP
	sound  string
	timbre bool
	mp     float64
	desc   string
}

type kvO1Case struct {
	label string
	rp    kvRP
	vref  bool
	mp    float64
	desc  string
}

var kv26Cases = []kv26Case{
	{"720p_none", kvRP{"720p", "std", "", 0.3, ""}, "", 0.3, "720p+无声 → 0.3元/秒"},
	{"1080p_none", kvRP{"1080p", "pro", "", 0.5, ""}, "", 0.5, "1080p+无声 → 0.5元/秒"},
	{"1080p_voice", kvRP{"1080p", "pro", "", 1.0, ""}, "on", 1.0, "1080p+有声 → 1.0元/秒"},
	{"2k_none", kvRP{"2k", "", "1440p", 0.75, ""}, "", 0.75, "2k+无声 → 0.75元/秒"},
	{"2k_voice", kvRP{"2k", "", "1440p", 1.5, ""}, "on", 1.5, "2k+有声 → 1.5元/秒"},
	{"4k_none", kvRP{"4k", "4k", "", 1.12, ""}, "", 1.12, "4k+无声 → 1.12元/秒"},
	{"4k_voice", kvRP{"4k", "4k", "", 2.25, ""}, "on", 2.25, "4k+有声 → 2.25元/秒"},
}

var kv30Cases = []kv30Case{
	{"720p_none", kvRP{"720p", "std", "", 0.6, ""}, "", false, 0.6, "720p+无声 → 0.6元/秒"},
	{"720p_voice", kvRP{"720p", "std", "", 0.9, ""}, "on", false, 0.9, "720p+有声 → 0.9元/秒"},
	{"720p_voice_timbre", kvRP{"720p", "std", "", 1.1, ""}, "on", true, 1.1, "720p+有声+音色 → 1.1元/秒"},
	{"1080p_none", kvRP{"1080p", "pro", "", 0.8, ""}, "", false, 0.8, "1080p+无声 → 0.8元/秒"},
	{"1080p_voice", kvRP{"1080p", "pro", "", 1.2, ""}, "on", false, 1.2, "1080p+有声 → 1.2元/秒"},
	{"1080p_voice_timbre", kvRP{"1080p", "pro", "", 1.4, ""}, "on", true, 1.4, "1080p+有声+音色 → 1.4元/秒"},
	{"2k_none", kvRP{"2k", "", "1440p", 1.0, ""}, "", false, 1.0, "2k+无声 → 1.0元/秒"},
	{"2k_voice", kvRP{"2k", "", "1440p", 1.5, ""}, "on", false, 1.5, "2k+有声 → 1.5元/秒"},
	{"2k_voice_timbre", kvRP{"2k", "", "1440p", 1.8, ""}, "on", true, 1.8, "2k+有声+音色 → 1.8元/秒"},
	{"4k_none", kvRP{"4k", "4k", "", 3.0, ""}, "", false, 3.0, "4k+无声 → 3.0元/秒"},
	{"4k_voice", kvRP{"4k", "4k", "", 3.0, ""}, "on", false, 3.0, "4k+有声 → 3.0元/秒"},
	{"4k_voice_timbre", kvRP{"4k", "4k", "", 2.4, ""}, "on", true, 2.4, "4k+有声+音色 → 2.4元/秒"},
}

var kvO1Cases = []kvO1Case{
	{"720p_video_ref", kvRP{"720p", "std", "", 0.9, ""}, true, 0.9, "720p+视频参考 → 0.9元/秒"},
	{"720p", kvRP{"720p", "std", "", 0.6, ""}, false, 0.6, "720p → 0.6元/秒"},
	{"1080p_video_ref", kvRP{"1080p", "pro", "", 1.2, ""}, true, 1.2, "1080p+视频参考 → 1.2元/秒"},
	{"1080p", kvRP{"1080p", "pro", "", 0.8, ""}, false, 0.8, "1080p → 0.8元/秒"},
	{"2k_video_ref", kvRP{"2k", "", "1440p", 1.8, ""}, true, 1.8, "2k+视频参考 → 1.8元/秒"},
	{"2k", kvRP{"2k", "", "1440p", 1.2, ""}, false, 1.2, "2k → 1.2元/秒"},
	{"4k_video_ref", kvRP{"4k", "4k", "", 2.7, ""}, true, 2.7, "4k+视频参考 → 2.7元/秒"},
	{"4k", kvRP{"4k", "4k", "", 1.8, ""}, false, 1.8, "4k → 1.8元/秒"},
}

// =============================================================================
// 构建请求体
// =============================================================================

func kvBuildBody(model string, dur int, rp kvRP, sound string) map[string]any {
	body := map[string]any{
		"model_name": model,
		"prompt":     fmt.Sprintf("自动化测试 %s %s", model, rp.label),
		"duration":   dur,
	}
	if rp.mode != "" {
		body["mode"] = rp.mode
	}
	if rp.res != "" {
		body["resolution"] = rp.res
	}
	if sound == "on" {
		body["sound"] = "on"
	}
	return body
}

func kvBuildResVtc(model, path string, bp float64, dur int, rp kvRP, img string, sound string) kvtc {
	body := kvBuildBody(model, dur, rp, sound)
	if img != "" {
		body["image"] = img
	}
	// 模型名中的点替换为下划线，避免 go test -run 正则问题
	n := fmt.Sprintf("%s %s",
		strings.ReplaceAll(model, ".", "_"),
		strings.ReplaceAll(rp.label, ".", "_"))
	return kvtc{N: n, P: path, B: body, C: 200, E: kvBE{BP: bp, MP: rp.price, D: dur, M: 1, DS: rp.desc}}
}

// =============================================================================
// TestKling 主入口
// =============================================================================

func TestKling(t *testing.T) {
	if *kvToken == "" {
		t.Skip("SKIP: 请用 -kv-token 参数提供 API token")
	}
	t.Run("text2video", kvTestText2Video)
	t.Run("image2video", kvTestImage2Video)
	t.Run("multi_image2video", kvTestMultiImage2Video)
	t.Run("omni_video", kvTestOmniVideo)
}

// =============================================================================
// text2video
// =============================================================================
// POST /m-kling/v1/videos/text2video
// 覆盖模型: kling-1.6, 2.0, 2.1, 2.5-pro, 2.6, 3.0

func kvTestText2Video(t *testing.T) {
	const path = "/m-kling/v1/videos/text2video"
	const dur = 5

	// ── 仅分辨率定价模型 ──
	t.Run("kling-1.6", func(t *testing.T) {
		for _, rp := range kvStdResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-1.6", path, 0.4, dur, rp, "", ""))
			})
		}
	})
	t.Run("kling-2.0", func(t *testing.T) {
		for _, rp := range kvStdResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-2.0", path, 0.4, dur, rp, "", ""))
			})
		}
	})
	t.Run("kling-2.1", func(t *testing.T) {
		for _, rp := range kvStdResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-2.1", path, 0.4, dur, rp, "", ""))
			})
		}
	})
	t.Run("kling-2.5-pro", func(t *testing.T) {
		for _, rp := range kv25pResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-2.5-pro", path, 0.3, dur, rp, "", ""))
			})
		}
	})

	// ── kling-2.6（分辨率 + audio_output: none/voice）──
	t.Run("kling-2.6", func(t *testing.T) {
		for _, c := range kv26Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-2.6", dur, c.rp, c.sound)
				kvRun(t, kvtc{
					N: "kling-2.6 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.3, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})

	// ── kling-3.0（分辨率 + audio_output: none/voice/voice_timbre）──
	t.Run("kling-3.0", func(t *testing.T) {
		for _, c := range kv30Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-3.0", dur, c.rp, c.sound)
				if c.timbre {
					body["voice_list"] = []map[string]any{{"voice_id": "v001"}}
				}
				kvRun(t, kvtc{
					N: "kling-3.0 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.6, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})
}

// =============================================================================
// image2video
// =============================================================================
// POST /m-kling/v1/videos/image2video
// 覆盖模型: kling-1.6, 2.0, 2.1, 2.5-pro, 2.6, 3.0, kling-o1
// kling-o1 视频参考使用官方 video_list 参数:
//   video_list: [{video_url: url, refer_type: "feature", keep_original_sound: "no"}]

func kvTestImage2Video(t *testing.T) {
	const path = "/m-kling/v1/videos/image2video"
	const dur = 5

	// ── 仅分辨率定价模型 + image ──
	t.Run("kling-1.6", func(t *testing.T) {
		for _, rp := range kvStdResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-1.6", path, 0.4, dur, rp, kvi1, ""))
			})
		}
	})
	t.Run("kling-2.0", func(t *testing.T) {
		for _, rp := range kvStdResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-2.0", path, 0.4, dur, rp, kvi1, ""))
			})
		}
	})
	t.Run("kling-2.1", func(t *testing.T) {
		for _, rp := range kvStdResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-2.1", path, 0.4, dur, rp, kvi1, ""))
			})
		}
	})
	t.Run("kling-2.5-pro", func(t *testing.T) {
		for _, rp := range kv25pResPrices {
			t.Run(rp.label, func(t *testing.T) {
				kvRun(t, kvBuildResVtc("kling-2.5-pro", path, 0.3, dur, rp, kvi1, ""))
			})
		}
	})

	// ── kling-2.6 + image ──
	t.Run("kling-2.6", func(t *testing.T) {
		for _, c := range kv26Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-2.6", dur, c.rp, c.sound)
				body["image"] = kvi1
				kvRun(t, kvtc{
					N: "kling-2.6 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.3, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})

	// ── kling-3.0 + image ──
	t.Run("kling-3.0", func(t *testing.T) {
		for _, c := range kv30Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-3.0", dur, c.rp, c.sound)
				if c.timbre {
					body["voice_list"] = []map[string]any{{"voice_id": "v001"}}
				}
				body["image"] = kvi1
				kvRun(t, kvtc{
					N: "kling-3.0 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.6, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})

	// ── kling-o1（分辨率 + reference_types: video）──
	t.Run("kling-o1", func(t *testing.T) {
		for _, c := range kvO1Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-o1", dur, c.rp, "")
				if c.vref {
					body["video_list"] = []map[string]any{
						{"video_url": kvv, "refer_type": "feature", "keep_original_sound": "no"},
					}
				}
				body["image"] = kvi1
				kvRun(t, kvtc{
					N: "kling-o1 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.6, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})
}

// =============================================================================
// multi-image2video
// =============================================================================
// POST /m-kling/v1/videos/multi-image2video
// 覆盖模型: kling-1.6, 2.0, 2.1, 2.5-pro, 2.6, 3.0, kling-3.0-omni, kling-o1
// 全部使用 Kling 官方 image_list 参数

func kvTestMultiImage2Video(t *testing.T) {
	const path = "/m-kling/v1/videos/multi-image2video"
	const dur = 5
	imageList := []map[string]any{
		{"image": kvi1}, {"image": kvi2}, {"image": kvi3},
	}

	// 分辨率定价模型 + image_list
	for _, m := range []struct {
		model  string
		bp     float64
		prices []kvRP
	}{
		{"kling-1.6", 0.4, kvStdResPrices},
		{"kling-2.0", 0.4, kvStdResPrices},
		{"kling-2.1", 0.4, kvStdResPrices},
		{"kling-2.5-pro", 0.3, kv25pResPrices},
	} {
		m := m
		t.Run(m.model, func(t *testing.T) {
			for _, rp := range m.prices {
				t.Run(rp.label, func(t *testing.T) {
					body := kvBuildBody(m.model, dur, rp, "")
					body["image_list"] = imageList
					kvRun(t, kvtc{
						N: m.model + " " + rp.label, P: path, B: body, C: 200,
						E: kvBE{BP: m.bp, MP: rp.price, D: dur, M: 1, DS: rp.desc},
					})
				})
			}
		})
	}

	// kling-2.6 + image_list
	t.Run("kling-2.6", func(t *testing.T) {
		for _, c := range kv26Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-2.6", dur, c.rp, c.sound)
				body["image_list"] = imageList
				kvRun(t, kvtc{
					N: "kling-2.6 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.3, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})

	// kling-3.0 + image_list
	t.Run("kling-3.0", func(t *testing.T) {
		for _, c := range kv30Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-3.0", dur, c.rp, c.sound)
				if c.timbre {
					body["voice_list"] = []map[string]any{{"voice_id": "v001"}}
				}
				body["image_list"] = imageList
				kvRun(t, kvtc{
					N: "kling-3.0 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.6, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})

	// kling-o1 + image_list（含 video_list 参考）
	t.Run("kling-o1", func(t *testing.T) {
		for _, c := range kvO1Cases {
			t.Run(c.label, func(t *testing.T) {
				body := kvBuildBody("kling-o1", dur, c.rp, "")
				if c.vref {
					body["video_list"] = []map[string]any{
						{"video_url": kvv, "refer_type": "feature", "keep_original_sound": "no"},
					}
				}
				body["image_list"] = imageList
				kvRun(t, kvtc{
					N: "kling-o1 " + c.label, P: path, B: body, C: 200,
					E: kvBE{BP: 0.6, MP: c.mp, D: dur, M: 1, DS: c.desc},
				})
			})
		}
	})
}

// =============================================================================
// omni-video
// =============================================================================
// POST /m-kling/v1/videos/omni-video
// 覆盖模型: kling-3.0-omni（16 条定价规则）
// 视频参考使用官方 video_list 参数:
//   video_list: [{video_url: url, refer_type: "feature", keep_original_sound: "no"}]
// 音频参考（音色）使用官方 voice_list 参数:
//   voice_list: [{voice_id: "v001"}]

func kvTestOmniVideo(t *testing.T) {
	const path = "/m-kling/v1/videos/omni-video"
	const dur = 5

	imageList := []map[string]any{
		{"image": kvi1},
		{"image": kvi2},
	}

	// 四种分辨率
	type resDef struct {
		label string
		mode  string
		res   string
	}
	resolutions := []resDef{
		{"720p", "std", ""},
		{"1080p", "pro", ""},
		{"2k", "", "1440p"},
		{"4k", "4k", ""},
	}

	// 每种分辨率的 4 种组合
	type comboDef struct {
		label  string
		sound  string
		vref   bool // 带视频参考文件
		afile  bool // 带音频参考文件
		mp     float64
		desc   string
	}

	// 各分辨率的定价规则
	combosByRes := map[string][]comboDef{
		"720p": {
			{"vref_voice", "on", true, false, 1.1, "720p+vref+voice → 1.1元/秒"},
			{"vref_none", "", true, false, 0.9, "720p+vref+none → 0.9元/秒"},
			{"voice", "on", false, false, 0.8, "720p+voice → 0.8元/秒"},
			{"none", "", false, false, 0.6, "720p+none → 0.6元/秒"},
		},
		"1080p": {
			{"vref_voice", "on", true, false, 1.4, "1080p+vref+voice → 1.4元/秒"},
			{"vref_none", "", true, false, 1.2, "1080p+vref+none → 1.2元/秒"},
			{"voice", "on", false, false, 1.0, "1080p+voice → 1.0元/秒"},
			{"none", "", false, false, 0.8, "1080p+none → 0.8元/秒"},
		},
		"2k": {
			{"vref_voice", "on", true, false, 1.8, "2k+vref+voice → 1.8元/秒"},
			{"vref_none", "", true, false, 1.5, "2k+vref+none → 1.5元/秒"},
			{"voice", "on", false, false, 1.2, "2k+voice → 1.2元/秒"},
			{"none", "", false, false, 1.0, "2k+none → 1.0元/秒"},
		},
		"4k": {
			{"vref_voice", "on", true, false, 2.4, "4k+vref+voice → 2.4元/秒"},
			{"vref_none", "", true, false, 2.0, "4k+vref+none → 2.0元/秒"},
			{"voice", "on", false, false, 3.0, "4k+voice → 3.0元/秒"},
			{"none", "", false, false, 3.0, "4k+none → 3.0元/秒"},
		},
	}

	for _, res := range resolutions {
		t.Run(res.label, func(t *testing.T) {
			for _, co := range combosByRes[res.label] {
				t.Run(co.label, func(t *testing.T) {
					body := map[string]any{
						"model_name": "kling-3.0-omni",
						"prompt":     fmt.Sprintf("自动化测试 omni %s %s", res.label, co.label),
						"duration":   dur,
						"image_list": imageList,
					}
					if res.mode != "" {
						body["mode"] = res.mode
					}
					if res.res != "" {
						body["resolution"] = res.res
					}
					if co.sound == "on" {
						body["sound"] = "on"
					}

					// 视频参考: Kling 官方 video_list 格式
					if co.vref {
						body["video_list"] = []map[string]any{
							{"video_url": kvv, "refer_type": "feature", "keep_original_sound": "no"},
						}
					}
					// 音频参考（音色）: Kling 官方 voice_list 格式
					if co.afile {
						body["voice_list"] = []map[string]any{
							{"voice_id": "v001"},
						}
					}

					kvRun(t, kvtc{
						N: fmt.Sprintf("omni %s-%s", res.label, co.label),
						P: path, B: body, C: 200,
						E: kvBE{BP: 0.6, MP: co.mp, D: dur, M: 1, DS: co.desc},
					})
				})
			}
		})
	}
}

// =============================================================================
// Hailuo / MiniMax 测试
// =============================================================================
// POST /minimax/v1/video_generation — MiniMax 官方原生格式
// 覆盖模型: MiniMax-Hailuo-2.3, MiniMax-Hailuo-2.3-Fast
// 计费方式: per_call（按次计费，不乘时长）
//
// MiniMax 官方请求参数:
//   model, prompt, first_frame_image, duration, resolution
//
// 运行:
//   go test -v -run TestHailuo -kv-token 'xxx' -kv-mock=false ./tests/
//   注意: MiniMax 渠道不走 mock（?mock=true 无效），需要真实渠道配置

func TestHailuo(t *testing.T) {
	if *kvToken == "" {
		t.Skip("SKIP: 请用 -kv-token 参数提供 API token")
	}

	const path = "/minimax/v1/video_generation"

	// ── MiniMax-Hailuo-2.3 ──
	t.Run("MiniMax-Hailuo-2.3", func(t *testing.T) {
		// 768p + duration=10 → price=4（有图）
		t.Run("768p_duration10", func(t *testing.T) {
			kvRun(t, kvtc{
				N: "MiniMax-Hailuo-2.3 768p+duration10",
				P: path, C: 200,
				B: map[string]any{
					"model":             "MiniMax-Hailuo-2.3",
					"prompt":            "测试 MiniMax-Hailuo-2.3 768p 10s 图生视频",
					"first_frame_image": kvi1,
					"duration":          10,
					"resolution":        "768P",
				},
				E: kvBE{BP: 2, MP: 4, D: 1, M: 1, DS: "768p+duration10+image → 4元/次"},
			})
		})
		// 1080p + duration=6 → price=3.5（有图）
		t.Run("1080p_duration6", func(t *testing.T) {
			kvRun(t, kvtc{
				N: "MiniMax-Hailuo-2.3 1080p+duration6",
				P: path, C: 200,
				B: map[string]any{
					"model":             "MiniMax-Hailuo-2.3",
					"prompt":            "测试 MiniMax-Hailuo-2.3 1080p 6s 图生视频",
					"first_frame_image": kvi1,
					"duration":          6,
					"resolution":        "1080P",
				},
				E: kvBE{BP: 2, MP: 3.5, D: 1, M: 1, DS: "1080p+duration6+image → 3.5元/次"},
			})
		})
		// 720p + duration=5 → fallback price=2（无图，无规则匹配）
		t.Run("720p_duration5_fallback", func(t *testing.T) {
			kvRun(t, kvtc{
				N: "MiniMax-Hailuo-2.3 720p+duration5 fallback",
				P: path, C: 200,
				B: map[string]any{
					"model":    "MiniMax-Hailuo-2.3",
					"prompt":   "测试 MiniMax-Hailuo-2.3 720p 5s 文生视频 fallback",
					"duration": 5,
					"resolution": "720P",
				},
				E: kvBE{BP: 2, MP: 2, D: 1, M: 1, DS: "fallback → 2元/次"},
			})
		})
	})

	// ── MiniMax-Hailuo-2.3-Fast ──
	t.Run("MiniMax-Hailuo-2.3-Fast", func(t *testing.T) {
		// 1080p + duration=6 + image → price=2.31
		t.Run("1080p_duration6_image", func(t *testing.T) {
			kvRun(t, kvtc{
				N: "MiniMax-Hailuo-2.3-Fast 1080p+duration6+image",
				P: path, C: 200,
				B: map[string]any{
					"model":             "MiniMax-Hailuo-2.3-Fast",
					"prompt":            "测试 MiniMax-Hailuo-2.3-Fast 1080p 6s 图生视频",
					"first_frame_image": kvi1,
					"duration":          6,
					"resolution":        "1080P",
				},
				E: kvBE{BP: 1.35, MP: 2.31, D: 1, M: 1, DS: "1080p+duration6+image → 2.31元/次"},
			})
		})
		// 768p + duration=10 + image → price=2.25
		t.Run("768p_duration10_image", func(t *testing.T) {
			kvRun(t, kvtc{
				N: "MiniMax-Hailuo-2.3-Fast 768p+duration10+image",
				P: path, C: 200,
				B: map[string]any{
					"model":             "MiniMax-Hailuo-2.3-Fast",
					"prompt":            "测试 MiniMax-Hailuo-2.3-Fast 768p 10s 图生视频",
					"first_frame_image": kvi1,
					"duration":          10,
					"resolution":        "768P",
				},
				E: kvBE{BP: 1.35, MP: 2.25, D: 1, M: 1, DS: "768p+duration10+image → 2.25元/次"},
			})
		})
		// 720p + duration=5 → fallback price=1.35（无图）
		t.Run("720p_duration5_fallback", func(t *testing.T) {
			kvRun(t, kvtc{
				N: "MiniMax-Hailuo-2.3-Fast 720p+duration5 fallback",
				P: path, C: 200,
				B: map[string]any{
					"model":    "MiniMax-Hailuo-2.3-Fast",
					"prompt":   "测试 MiniMax-Hailuo-2.3-Fast 720p 5s 文生视频 fallback",
					"duration": 5,
					"resolution": "720P",
				},
				E: kvBE{BP: 1.35, MP: 1.35, D: 1, M: 1, DS: "fallback → 1.35元/次"},
			})
		})
	})
}
