package doubao

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
)

func TestBuildVolcesSubmitResponse(t *testing.T) {
	body, err := BuildVolcesSubmitResponse("task_public_id")
	if err != nil {
		t.Fatalf("BuildVolcesSubmitResponse returned error: %v", err)
	}

	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if len(payload) != 1 {
		t.Fatalf("unexpected payload keys: %#v", payload)
	}
	if got := payload["id"]; got != "task_public_id" {
		t.Fatalf("unexpected id: %#v", got)
	}
}

func TestBuildVolcesFetchResponse(t *testing.T) {
	raw := []byte(`{"id":"upstream_id","status":"succeeded","content":{"video_url":"https://example.com/video.mp4"}}`)
	body, err := BuildVolcesFetchResponse("task_public_id", raw)
	if err != nil {
		t.Fatalf("BuildVolcesFetchResponse returned error: %v", err)
	}

	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got := payload["id"]; got != "task_public_id" {
		t.Fatalf("unexpected id: %#v", got)
	}
	if got := payload["status"]; got != "succeeded" {
		t.Fatalf("unexpected status: %#v", got)
	}
	content, ok := payload["content"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected content: %#v", payload["content"])
	}
	if got := content["video_url"]; got != "https://example.com/video.mp4" {
		t.Fatalf("unexpected video_url: %#v", got)
	}
}
