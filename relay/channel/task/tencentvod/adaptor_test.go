package tencentvod

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseKey(t *testing.T) {
	cfg, err := ParseKey("sid|skey|12345")
	require.NoError(t, err)
	assert.Equal(t, "sid", cfg.SecretID)
	assert.Equal(t, "skey", cfg.SecretKey)
	assert.EqualValues(t, 12345, cfg.SubAppID)
}

func TestParseKeyInvalid(t *testing.T) {
	_, err := ParseKey("sid|skey")
	require.Error(t, err)
}

func TestResolveModelByRegistry(t *testing.T) {
	spec, err := ResolveModel("kling-3.0-omni")
	require.NoError(t, err)
	assert.Equal(t, "Kling", spec.ModelName)
	assert.Equal(t, "3.0-Omni", spec.ModelVersion)
	assert.Equal(t, "kling-3.0-omni", spec.Canonical)
}

func TestResolveModelByFallback(t *testing.T) {
	spec, err := ResolveModel("vidu-q3-pro")
	require.NoError(t, err)
	assert.Equal(t, "Vidu", spec.ModelName)
	assert.Equal(t, "q3-pro", spec.ModelVersion)
}

func TestResolveModelRejectsUnknownProvider(t *testing.T) {
	_, err := ResolveModel("unknown-1.0")
	require.Error(t, err)
}

func TestDetectBillingMode(t *testing.T) {
	tests := []struct {
		name string
		req  relaycommon.TaskSubmitReq
		want BillingMode
	}{
		{
			name: "scene mode",
			req:  relaycommon.TaskSubmitReq{Metadata: map[string]interface{}{"scene_type": "motion_control"}},
			want: BillingModeSceneMotionControl,
		},
		{
			name: "video edit",
			req: relaycommon.TaskSubmitReq{Metadata: map[string]interface{}{
				"file_infos": []interface{}{
					map[string]interface{}{"category": "Video", "reference_type": "Edit"},
				},
			}},
			want: BillingModeVideoEdit,
		},
		{
			name: "video reference",
			req: relaycommon.TaskSubmitReq{Metadata: map[string]interface{}{
				"file_infos": []interface{}{
					map[string]interface{}{"category": "Video"},
				},
			}},
			want: BillingModeReferenceVideo,
		},
		{
			name: "first last frame",
			req: relaycommon.TaskSubmitReq{
				Images: []string{"https://example.com/a.png"},
				Metadata: map[string]interface{}{
					"last_frame_url": "https://example.com/b.png",
				},
			},
			want: BillingModeFirstLastFrame,
		},
		{
			name: "reference image",
			req:  relaycommon.TaskSubmitReq{Images: []string{"https://example.com/a.png"}},
			want: BillingModeReferenceImage,
		},
		{
			name: "text to video",
			req:  relaycommon.TaskSubmitReq{Prompt: "hello"},
			want: BillingModeTextToVideo,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, DetectBillingMode(&tc.req))
		})
	}
}

func TestParseTaskResultFinish(t *testing.T) {
	body := []byte(`{
		"Response":{
			"AigcVideoTask":{
				"ErrCode":0,
				"Message":"",
				"Progress":100,
				"Status":"FINISH",
				"Output":{"FileInfos":[{"FileType":"mp4","FileUrl":"https://vod.example.com/video.mp4"}]},
				"TaskId":"upstream-task"
			}
		}
	}`)

	adaptor := &TaskAdaptor{}
	info, err := adaptor.ParseTaskResult(body)
	require.NoError(t, err)
	assert.Equal(t, model.TaskStatusSuccess, info.Status)
	assert.Equal(t, "100%", info.Progress)
	assert.Equal(t, "https://vod.example.com/video.mp4", info.Url)
}

func TestEstimateBillingUsesModeDurationResolution(t *testing.T) {
	adaptor := &TaskAdaptor{}
	info := &relaycommon.RelayInfo{OriginModelName: "kling-2.6"}

	req := relaycommon.TaskSubmitReq{
		Prompt:   "hello",
		Duration: 8,
		Size:     "1920x1080",
	}

	ratios := adaptor.estimateBillingFromRequest(&req, info)
	require.NotNil(t, ratios)
	assert.Equal(t, ModeRatioMap[BillingModeTextToVideo], ratios[BillingRatioMode])
	assert.Equal(t, 8.0/4.0, ratios[BillingRatioDuration])
	assert.Equal(t, 1080.0/720.0, ratios[BillingRatioResolution])
}

func TestInitUsesDefaultRegionOnly(t *testing.T) {
	adaptor := &TaskAdaptor{}
	adaptor.Init(&relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ApiKey:               "sid|skey|12345",
			ChannelOtherSettings: dto.ChannelOtherSettings{},
		},
	})

	assert.Equal(t, defaultRegion, adaptor.region)
}

func TestBuildRequestHeaderUsesDefaultRegionOnly(t *testing.T) {
	adaptor := &TaskAdaptor{
		keyConfig: KeyConfig{
			SecretID:  "sid",
			SecretKey: "skey",
			SubAppID:  12345,
		},
		region: defaultRegion,
	}
	req, err := http.NewRequest(http.MethodPost, "https://vod.tencentcloudapi.com/", strings.NewReader(`{}`))
	require.NoError(t, err)

	err = adaptor.BuildRequestHeader(nil, req, &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			ChannelOtherSettings: dto.ChannelOtherSettings{},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, defaultRegion, req.Header.Get("X-TC-Region"))
}

func TestConvertToRequestPayloadDoesNotFallbackInputRegion(t *testing.T) {
	adaptor := &TaskAdaptor{region: defaultRegion}
	req := &relaycommon.TaskSubmitReq{
		Model:  "kling-2.6",
		Prompt: "hello",
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "kling-2.6",
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	assert.Empty(t, payload.InputRegion)
}

func TestFetchTaskUsesDefaultRegionOnly(t *testing.T) {
	service.InitHttpClient()
	var receivedRegion string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRegion = r.Header.Get("X-TC-Region")
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"Response":{"AigcVideoTask":{"TaskId":"upstream-task","Status":"PROCESSING","Progress":50}}}`)
	}))
	defer server.Close()

	adaptor := &TaskAdaptor{}
	resp, err := adaptor.FetchTask(server.URL, "sid|skey|12345", map[string]any{
		"task_id": "upstream-task",
		"region":  "ap-shanghai",
	}, "")
	require.NoError(t, err)
	_ = resp.Body.Close()

	assert.Equal(t, defaultRegion, receivedRegion)
}

func TestValidateRequestAllowsReferenceVideoWithoutPrompt(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("POST", "/v1/videos", strings.NewReader(`{
		"model":"kling-2.6",
		"metadata":{
			"file_infos":[
				{"category":"Video","type":"Url","url":"https://example.com/input.mp4"}
			]
		}
	}`))
	c.Request.Header.Set("Content-Type", "application/json")

	info := &relaycommon.RelayInfo{}
	adaptor := &TaskAdaptor{}
	err := adaptor.ValidateRequestAndSetAction(c, info)
	require.Nil(t, err)
	assert.Equal(t, "videoReferenceGenerate", info.Action)
}

func TestConvertToOpenAIVideo(t *testing.T) {
	resp := DescribeTaskDetailResponse{
		Response: DescribeTaskDetailResponseBody{
			AigcVideoTask: &AigcVideoTask{
				Status:   "FINISH",
				Progress: 100,
				Output: AigcVideoTaskOutput{
					FileInfos: []AigcVideoOutputFileInfo{
						{FileType: "mp4", FileURL: "https://vod.example.com/video.mp4"},
					},
				},
			},
			FinishTime: "2026-05-22T09:40:11Z",
		},
	}
	data, err := common.Marshal(resp)
	require.NoError(t, err)

	task := &model.Task{
		TaskID:   "task_123",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
		Properties: model.Properties{
			OriginModelName: "kling-2.6",
		},
		Data: data,
	}

	adaptor := &TaskAdaptor{}
	out, err := adaptor.ConvertToOpenAIVideo(task)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"model":"kling-2.6"`)
	assert.Contains(t, string(out), "video.mp4")
}

func TestConvertToRequestPayloadMapsMViduCommonFields(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:    "vidu-q3",
		Prompt:   "cinematic scene",
		Duration: 8,
		Metadata: map[string]interface{}{
			"seed":         123,
			"payload":      "trace-1",
			"resolution":   "1080p",
			"aspect_ratio": "16:9",
			"audio":        true,
			"bgm":          false,
			"off_peak":     true,
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotNil(t, payload.OutputConfig)
	assert.Equal(t, 123, payload.Seed)
	assert.Equal(t, "trace-1", payload.SessionContext)
	assert.Equal(t, "1080p", payload.OutputConfig.Resolution)
	assert.Equal(t, "16:9", payload.OutputConfig.AspectRatio)
	assert.Equal(t, "Enabled", payload.OutputConfig.AudioGeneration)
	assert.Equal(t, "Disabled", payload.OutputConfig.EnableBGM)
	assert.Equal(t, "Enabled", payload.OutputConfig.OffPeak)
	assert.Equal(t, 8, payload.OutputConfig.Duration)
}

func TestConvertToRequestPayloadMapsMViduStartEndImages(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "smooth transition",
		Images: []string{
			"https://example.com/first.png",
			"https://example.com/last.png",
		},
		Metadata: map[string]interface{}{
			"images": []interface{}{
				"https://example.com/first.png",
				"https://example.com/last.png",
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionFirstTailGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.FileInfos, 1)
	assert.Equal(t, "FirstFrame", payload.FileInfos[0].Usage)
	assert.Equal(t, "https://example.com/first.png", payload.FileInfos[0].URL)
	assert.Equal(t, "https://example.com/last.png", payload.LastFrameURL)
}

func TestConvertToRequestPayloadMapsMViduReferenceVideos(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "cinematic motion",
		Metadata: map[string]interface{}{
			"videos": []interface{}{
				"https://example.com/ref.mp4",
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReferenceGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.FileInfos, 1)
	assert.Equal(t, "Video", payload.FileInfos[0].Category)
	assert.Equal(t, "https://example.com/ref.mp4", payload.FileInfos[0].URL)
}

func TestConvertToRequestPayloadMapsMViduReferenceImagesAndVideos(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "reference mix",
		Images: []string{
			"https://example.com/ref.png",
		},
		Metadata: map[string]interface{}{
			"images": []interface{}{
				"https://example.com/ref.png",
			},
			"videos": []interface{}{
				"https://example.com/ref.mp4",
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReferenceGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.FileInfos, 2)
	var imageFound, videoFound bool
	for _, item := range payload.FileInfos {
		switch item.Category {
		case "Image":
			imageFound = true
			assert.Equal(t, "https://example.com/ref.png", item.URL)
			assert.Equal(t, "Reference", item.Usage)
		case "Video":
			videoFound = true
			assert.Equal(t, "https://example.com/ref.mp4", item.URL)
		}
	}
	assert.True(t, imageFound)
	assert.True(t, videoFound)
}

func TestConvertToRequestPayloadMapsMViduSubjectsToFileInfosOnly(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "subject showcase",
		Metadata: map[string]interface{}{
			"subjects": []interface{}{
				map[string]interface{}{
					"name":      "hero",
					"server_id": "obj_123",
					"voice_id":  "voice_1",
					"images": []interface{}{
						"https://example.com/hero-1.png",
					},
					"videos": []interface{}{
						"https://example.com/hero-1.mp4",
					},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReferenceGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	assert.Empty(t, payload.SubjectInfos)
	require.Len(t, payload.FileInfos, 2)
	assert.Equal(t, "obj_123", payload.FileInfos[0].ObjectID)
	assert.Equal(t, "hero", payload.FileInfos[0].Text)
	assert.Equal(t, "voice_1", payload.FileInfos[0].VoiceID)
	assert.Equal(t, "Reference", payload.FileInfos[0].Usage)
	assert.Equal(t, "Video", payload.FileInfos[1].Category)
	assert.Equal(t, "hero", payload.FileInfos[1].Text)
}

func TestConvertToRequestPayloadIgnoresUnsupportedMViduFields(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "ignore unsupported fields",
		Metadata: map[string]interface{}{
			"style":              "anime",
			"movement_amplitude": "large",
			"watermark":          true,
			"wm_position":        "bottom-right",
			"callback_url":       "https://example.com/callback",
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	assert.Empty(t, payload.ExtInfo)
	assert.Empty(t, payload.SceneType)
	assert.Empty(t, payload.Procedure)
}
