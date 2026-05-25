package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	relayconstant "github.com/QuantumNous/new-api/relay/constant"
	"github.com/gin-gonic/gin"
)

func TestConvertMKlingRequestToUnifiedActionMapping(t *testing.T) {
	cases := []struct {
		name        string
		requestType string
		raw         map[string]any
		wantAction  string
	}{
		{
			name:        "text2video maps to textGenerate",
			requestType: mKlingRequestTypeText2Video,
			raw: map[string]any{
				"model_name": "kling-v2-master",
				"prompt":     "a fox running in the snow",
			},
			wantAction: constant.TaskActionTextGenerate,
		},
		{
			name:        "image2video maps to firstTailGenerate",
			requestType: mKlingRequestTypeImage2Video,
			raw: map[string]any{
				"model_name": "kling-v2-master",
				"image":      "https://example.com/first.png",
				"image_tail": "https://example.com/last.png",
			},
			wantAction: constant.TaskActionFirstTailGenerate,
		},
		{
			name:        "multi image maps to referenceGenerate",
			requestType: mKlingRequestTypeMultiImage2Video,
			raw: map[string]any{
				"model_name": "kling-v2-master",
				"images": []any{
					"https://example.com/frame-1.png",
					"https://example.com/frame-2.png",
					"https://example.com/frame-3.png",
				},
			},
			wantAction: constant.TaskActionReferenceGenerate,
		},
		{
			name:        "omni maps to referenceGenerate",
			requestType: mKlingRequestTypeOmniVideo,
			raw: map[string]any{
				"model_name": "kling-v2-master",
				"prompt":     "make it cinematic",
				"images": []any{
					"https://example.com/ref-1.png",
					"https://example.com/ref-2.png",
				},
			},
			wantAction: constant.TaskActionReferenceGenerate,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			unified, action, err := convertMKlingRequestToUnified(tc.raw, tc.requestType)
			if err != nil {
				t.Fatalf("convertMKlingRequestToUnified returned error: %v", err)
			}
			if action != tc.wantAction {
				t.Fatalf("unexpected action: %s", action)
			}
			if got := unified["model"]; got != "kling-v2-master" {
				t.Fatalf("unexpected model: %#v", got)
			}
			metadata, ok := unified["metadata"].(map[string]any)
			if !ok {
				t.Fatalf("metadata missing or invalid: %#v", unified["metadata"])
			}
			if _, ok := metadata["model_name"]; !ok {
				t.Fatalf("original model_name should remain in metadata")
			}
		})
	}
}

func TestConvertMKlingRequestToUnifiedPreservesSpecialFields(t *testing.T) {
	raw := map[string]any{
		"model_name":       "kling-v2-master",
		"prompt":           "a spaceship landing",
		"image":            "https://example.com/first.png",
		"image_tail":       "https://example.com/last.png",
		"callback_url":     "https://example.com/callback",
		"external_task_id": "client-task-1",
		"camera_control": map[string]any{
			"type": "simple",
		},
	}

	unified, _, err := convertMKlingRequestToUnified(raw, mKlingRequestTypeImage2Video)
	if err != nil {
		t.Fatalf("convertMKlingRequestToUnified returned error: %v", err)
	}

	if got := unified["image"]; got != "https://example.com/first.png" {
		t.Fatalf("unexpected image: %#v", got)
	}
	metadata := unified["metadata"].(map[string]any)
	if got := metadata["image_tail"]; got != "https://example.com/last.png" {
		t.Fatalf("unexpected image_tail: %#v", got)
	}
	if got := metadata["callback_url"]; got != "https://example.com/callback" {
		t.Fatalf("unexpected callback_url: %#v", got)
	}
	if got := metadata["external_task_id"]; got != "client-task-1" {
		t.Fatalf("unexpected external_task_id: %#v", got)
	}
}

func TestMKlingRequestConvertSetsCompatContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(
		http.MethodPost,
		"/m-kling/v1/videos/text2video",
		strings.NewReader(`{"model_name":"kling-v2-master","prompt":"test prompt"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	MKlingRequestConvert()(c)
	if c.IsAborted() {
		t.Fatalf("request should not be aborted")
	}

	modelRequest, shouldSelectChannel, err := getModelRequest(c)
	if err != nil {
		t.Fatalf("getModelRequest returned error: %v", err)
	}
	if !shouldSelectChannel {
		t.Fatalf("shouldSelectChannel should be true")
	}
	if modelRequest.Model != "kling-v2-master" {
		t.Fatalf("unexpected model: %s", modelRequest.Model)
	}
	if c.GetString("action") != constant.TaskActionTextGenerate {
		t.Fatalf("unexpected action: %s", c.GetString("action"))
	}
	if c.GetInt("relay_mode") != relayconstant.RelayModeVideoSubmit {
		t.Fatalf("unexpected relay_mode: %d", c.GetInt("relay_mode"))
	}
	if !common.GetContextKeyBool(c, constant.ContextKeyMKlingCompat) {
		t.Fatalf("m_kling compat flag missing")
	}
}

func TestMKlingRequestConvertRewritesBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MKlingRequestConvert())
	router.POST("/m-kling/v1/videos/multi-image2video", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Data(http.StatusOK, "application/json", body)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/m-kling/v1/videos/multi-image2video",
		strings.NewReader(`{"model_name":"kling-v2-master","images":["https://example.com/1.png","https://example.com/2.png"]}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}
	var payload map[string]any
	if err := common.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal rewritten body: %v", err)
	}
	if _, ok := payload["images"]; !ok {
		t.Fatalf("images should remain in unified request")
	}
	if metadata, ok := payload["metadata"].(map[string]any); !ok || metadata["model_name"] != "kling-v2-master" {
		t.Fatalf("metadata should preserve official model_name")
	}
}

func TestMKlingRequestConvertSupportsOfficialPaths(t *testing.T) {
	cases := []string{
		"/m-kling/v1/videos/text2video",
		"/m-kling/v1/videos/image2video",
		"/m-kling/v1/videos/multi-image2video",
		"/m-kling/v1/videos/omni-video",
	}

	for _, path := range cases {
		gin.SetMode(gin.TestMode)
		router := gin.New()
		router.Use(MKlingRequestConvert())
		router.POST(path, func(c *gin.Context) {
			c.Status(http.StatusNoContent)
		})

		req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"model_name":"kling-v2-master","prompt":"x"}`))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("unexpected status for %s: %d", path, w.Code)
		}
	}
}
