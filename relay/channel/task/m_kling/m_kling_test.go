package mkling

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
	if got := payload["code"]; got != float64(0) {
		t.Fatalf("unexpected code: %#v", got)
	}
	data, ok := payload["data"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected data: %#v", payload["data"])
	}
	if got := data["task_id"]; got != "task_public_id" {
		t.Fatalf("unexpected task_id: %#v", got)
	}
}

func TestBuildQueryResponseSuccess(t *testing.T) {
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
	data := payload["data"].(map[string]any)
	if got := data["task_status"]; got != "succeed" {
		t.Fatalf("unexpected task_status: %#v", got)
	}
	taskResult := data["task_result"].(map[string]any)
	videos := taskResult["videos"].([]any)
	video := videos[0].(map[string]any)
	if got := video["url"]; got != "https://example.com/video.mp4" {
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
	data := payload["data"].(map[string]any)
	if got := data["task_status"]; got != "failed" {
		t.Fatalf("unexpected task_status: %#v", got)
	}
	if got := data["task_status_msg"]; got != "upstream_error" {
		t.Fatalf("unexpected task_status_msg: %#v", got)
	}
}

func TestMapTaskStatusQueued(t *testing.T) {
	if got := mapTaskStatus(model.TaskStatusQueued); got != "submitted" {
		t.Fatalf("unexpected queued mapping: %s", got)
	}
}
