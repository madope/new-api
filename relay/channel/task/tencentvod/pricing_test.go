package tencentvod

import (
	"testing"

	"net/http/httptest"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/relay/helper/videopricing"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newPricingTestCtx(req relaycommon.TaskSubmitReq) (*gin.Context, *relaycommon.RelayInfo) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Set("task_request", req)
	info := &relaycommon.RelayInfo{
		OriginModelName: req.Model,
	}
	return c, info
}

func TestTencentVODParseParams(t *testing.T) {
	tests := []struct {
		name    string
		req     relaycommon.TaskSubmitReq
		want    videopricing.VideoPricingParams
		wantErr bool
	}{
		{
			name: "纯文生视频 - kling官方参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Mode:   "std",
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "720p",
				Duration:       5,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "none",
			},
		},
		{
			name: "文生视频 - 指定pro模式",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Mode:   "pro",
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "1080p",
				Duration:       5,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "none",
			},
		},
		{
			name: "文生视频 - 指定4k模式",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Mode:   "4k",
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "4k",
				Duration:       5,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "none",
			},
		},
		{
			name: "图生视频 - image参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Image:  "https://example.com/img.jpg",
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{videopricing.ReferenceImage},
				AudioOutput:    "none",
			},
		},
		{
			name: "图生视频 - image_list参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"image_list": []string{"https://example.com/img1.jpg", "https://example.com/img2.jpg"},
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{videopricing.ReferenceImage},
				AudioOutput:    "none",
			},
		},
		{
			name: "图生视频 - image_tail参数(尾帧)",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"image":       "https://example.com/first.jpg",
					"image_tail":  "https://example.com/last.jpg",
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{videopricing.ReferenceImage},
				AudioOutput:    "none",
			},
		},
		{
			name: "视频参考 - video_list参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"video_list": []string{"https://example.com/ref.mp4"},
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{videopricing.ReferenceVideo},
				AudioOutput:    "none",
			},
		},
		{
			name: "视频+图片参考 - video_list + image_list",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"video_list": []string{"https://example.com/ref.mp4"},
					"image_list": []string{"https://example.com/first.png"},
				},
			},
			want: videopricing.VideoPricingParams{
				Model:      "kling-3.0",
				Resolution: "768p",
				Duration:   4,
				ReferenceTypes: []videopricing.ReferenceType{
					videopricing.ReferenceVideo,
					videopricing.ReferenceImage,
				},
				AudioOutput: "none",
			},
		},
		{
			name: "有音频输出 - sound参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"sound": "on",
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "voice",
			},
		},
		{
			name: "有音频输出 - audio参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"audio": true,
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "voice",
			},
		},
		{
			name: "有音频输出+音色 - voice_list参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"sound":      "on",
					"voice_list": []string{"voice_1"},
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "voice_timbre",
			},
		},
		{
			name: "指定分辨率 - resolution参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"resolution": "1080p",
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "1080p",
				Duration:       4,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "none",
			},
		},
		{
			name: "指定时长 - duration参数",
			req: relaycommon.TaskSubmitReq{
				Model:    "kling-3.0",
				Prompt:   "test prompt",
				Duration: 10,
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       10,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "none",
			},
		},
		{
			name: "指定时长 - output_config.Duration",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Metadata: map[string]interface{}{
					"output_config": map[string]interface{}{
						"Duration": 15,
					},
				},
			},
			want: videopricing.VideoPricingParams{
				Model:          "kling-3.0",
				Resolution:     "768p",
				Duration:       15,
				ReferenceTypes: []videopricing.ReferenceType{},
				AudioOutput:    "none",
			},
		},
		{
			name: "完全参数组合 - kling官方参数",
			req: relaycommon.TaskSubmitReq{
				Model:  "kling-3.0",
				Prompt: "test prompt",
				Mode:   "pro",
				Metadata: map[string]interface{}{
					"image_list":    []string{"https://example.com/img.jpg"},
					"video_list":    []string{"https://example.com/ref.mp4"},
					"sound":         "on",
					"voice_list":    []string{"voice_1"},
					"output_config": map[string]interface{}{"Duration": 20},
				},
			},
			want: videopricing.VideoPricingParams{
				Model:      "kling-3.0",
				Resolution: "1080p",
				Duration:   20,
				ReferenceTypes: []videopricing.ReferenceType{
					videopricing.ReferenceImage,
					videopricing.ReferenceVideo,
				},
				AudioOutput: "voice_timbre",
			},
		},
	}

	adaptor := &TencentVODVideoPricingAdaptor{
		BaseVideoPricingAdaptor: &videopricing.BaseVideoPricingAdaptor{},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, info := newPricingTestCtx(tt.req)
			got, err := adaptor.ParseParams(c, info)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want.Model, got.Model)
			assert.Equal(t, tt.want.Resolution, got.Resolution)
			assert.Equal(t, tt.want.Duration, got.Duration)
			assert.Equal(t, tt.want.AudioOutput, got.AudioOutput)
			assert.ElementsMatch(t, tt.want.ReferenceTypes, got.ReferenceTypes)
		})
	}
}