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

func TestConvertMViduRequestToUnified(t *testing.T) {
	raw := map[string]any{
		"model":        "vidu-q2",
		"prompt":       "a cat running",
		"duration":     float64(4),
		"future_flag":  "beta",
		"callback_url": "https://example.com/callback",
	}

	unified, action, err := convertMViduRequestToUnified(raw, mViduRequestTypeText2Video)
	if err != nil {
		t.Fatalf("convertMViduRequestToUnified returned error: %v", err)
	}
	if action != constant.TaskActionTextGenerate {
		t.Fatalf("unexpected action: %s", action)
	}
	if got := unified["model"]; got != "vidu-q2" {
		t.Fatalf("unexpected model: %#v", got)
	}
	if got := unified["prompt"]; got != "a cat running" {
		t.Fatalf("unexpected prompt: %#v", got)
	}
	if got := unified["duration"]; got != float64(4) {
		t.Fatalf("unexpected duration: %#v", got)
	}
	metadata, ok := unified["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or invalid: %#v", unified["metadata"])
	}
	if _, exists := metadata["model"]; exists {
		t.Fatalf("model should not remain in metadata")
	}
	if _, exists := metadata["prompt"]; exists {
		t.Fatalf("prompt should not remain in metadata")
	}
	if got := metadata["future_flag"]; got != "beta" {
		t.Fatalf("unexpected future_flag: %#v", got)
	}
	if got := metadata["callback_url"]; got != "https://example.com/callback" {
		t.Fatalf("unexpected callback_url: %#v", got)
	}
}

func TestConvertMViduRequestToUnifiedKeepsImagesInMetadata(t *testing.T) {
	raw := map[string]any{
		"model":  "vidu-q2",
		"images": []any{"https://example.com/first.png", "https://example.com/last.png"},
		"seed":   float64(7),
	}

	unified, action, err := convertMViduRequestToUnified(raw, mViduRequestTypeStartEnd2Video)
	if err != nil {
		t.Fatalf("convertMViduRequestToUnified returned error: %v", err)
	}
	if action != constant.TaskActionFirstTailGenerate {
		t.Fatalf("unexpected action: %s", action)
	}
	images, ok := unified["images"].([]any)
	if !ok || len(images) != 2 {
		t.Fatalf("unexpected unified images: %#v", unified["images"])
	}
	metadata, ok := unified["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or invalid: %#v", unified["metadata"])
	}
	metaImages, ok := metadata["images"].([]any)
	if !ok || len(metaImages) != 2 {
		t.Fatalf("unexpected metadata images: %#v", metadata["images"])
	}
}

func TestMViduRequestConvertText2Video(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MViduRequestConvert())
	router.POST("/m-vidu/ent/v2/text2video", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Data(http.StatusOK, "application/json", body)
	})

	req := httptest.NewRequest(
		http.MethodPost,
		"/m-vidu/ent/v2/text2video",
		strings.NewReader(`{"model":"vidu-q2","prompt":"a cat running","duration":4,"future_flag":"beta"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", w.Code)
	}
	var payload map[string]any
	if err := common.Unmarshal(w.Body.Bytes(), &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got := payload["model"]; got != "vidu-q2" {
		t.Fatalf("unexpected model: %#v", got)
	}
	metadata, ok := payload["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("metadata missing or invalid: %#v", payload["metadata"])
	}
	if got := metadata["future_flag"]; got != "beta" {
		t.Fatalf("unexpected future_flag: %#v", got)
	}
}

func TestGetModelRequestForMViduSubmit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	req := httptest.NewRequest(
		http.MethodPost,
		"/m-vidu/ent/v2/text2video",
		strings.NewReader(`{"model":"vidu-q2","prompt":"a cat running"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	c.Request = req

	MViduRequestConvert()(c)
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
	if modelRequest.Model != "vidu-q2" {
		t.Fatalf("unexpected model: %s", modelRequest.Model)
	}
	if c.GetString("action") != constant.TaskActionTextGenerate {
		t.Fatalf("unexpected action: %s", c.GetString("action"))
	}
	if c.GetInt("relay_mode") != relayconstant.RelayModeVideoSubmit {
		t.Fatalf("unexpected relay_mode: %d", c.GetInt("relay_mode"))
	}
	if !common.GetContextKeyBool(c, constant.ContextKeyMViduCompat) {
		t.Fatalf("m_vidu compat flag missing")
	}
}
