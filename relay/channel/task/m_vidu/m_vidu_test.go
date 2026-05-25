package mvidu

import (
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

func TestBuildSubmitResponse(t *testing.T) {
	body, err := BuildSubmitResponse("task_public_id")
	if err != nil {
		t.Fatalf("BuildSubmitResponse returned error: %v", err)
	}

	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got := payload["task_id"]; got != "task_public_id" {
		t.Fatalf("unexpected task_id: %#v", got)
	}
	if got := payload["state"]; got != "created" {
		t.Fatalf("unexpected state: %#v", got)
	}
}

func TestBuildQueryResponse(t *testing.T) {
	task := &model.Task{
		TaskID: "task_public_id",
		Status: model.TaskStatusSuccess,
		PrivateData: model.TaskPrivateData{
			ResultURL: "https://example.com/video.mp4",
		},
	}
	body, err := BuildQueryResponse(task)
	if err != nil {
		t.Fatalf("BuildQueryResponse returned error: %v", err)
	}

	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got := payload["id"]; got != "task_public_id" {
		t.Fatalf("unexpected id: %#v", got)
	}
	if got := payload["state"]; got != "success" {
		t.Fatalf("unexpected state: %#v", got)
	}
	creations, ok := payload["creations"].([]any)
	if !ok || len(creations) != 1 {
		t.Fatalf("unexpected creations: %#v", payload["creations"])
	}
	creation, ok := creations[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected creation item: %#v", creations[0])
	}
	if got := creation["url"]; got != "https://example.com/video.mp4" {
		t.Fatalf("unexpected url: %#v", got)
	}
}

func TestBuildQueryResponseFailure(t *testing.T) {
	task := &model.Task{
		TaskID:     "task_public_id",
		Status:     model.TaskStatusFailure,
		FailReason: "upstream_error",
	}
	body, err := BuildQueryResponse(task)
	if err != nil {
		t.Fatalf("BuildQueryResponse returned error: %v", err)
	}
	var payload map[string]any
	if err := common.Unmarshal(body, &payload); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	if got := payload["state"]; got != "failed" {
		t.Fatalf("unexpected state: %#v", got)
	}
	if got := payload["err_code"]; got != "upstream_error" {
		t.Fatalf("unexpected err_code: %#v", got)
	}
}
