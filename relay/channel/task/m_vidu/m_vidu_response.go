package mvidu

import (
	"fmt"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type submitResponse struct {
	TaskID string `json:"task_id"`
	State  string `json:"state"`
}

type queryResponse struct {
	ID        string         `json:"id"`
	State     string         `json:"state"`
	ErrCode   string         `json:"err_code"`
	Creations []creationItem `json:"creations"`
}

type creationItem struct {
	ID       string `json:"id"`
	URL      string `json:"url,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
}

func BuildSubmitResponse(publicTaskID string) ([]byte, error) {
	return common.Marshal(submitResponse{
		TaskID: publicTaskID,
		State:  "created",
	})
}

func BuildQueryResponse(task *model.Task) ([]byte, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	resp := queryResponse{
		ID:      task.TaskID,
		State:   mapTaskState(task.Status),
		ErrCode: "",
	}
	if task.Status == model.TaskStatusFailure {
		resp.ErrCode = task.FailReason
	}
	if resultURL := task.GetResultURL(); resultURL != "" {
		resp.Creations = []creationItem{{
			ID:  task.TaskID,
			URL: resultURL,
		}}
	} else {
		resp.Creations = []creationItem{}
	}
	return common.Marshal(resp)
}

func mapTaskState(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusSubmitted:
		return "created"
	case model.TaskStatusQueued:
		return "queueing"
	case model.TaskStatusInProgress:
		return "processing"
	case model.TaskStatusSuccess:
		return "success"
	case model.TaskStatusFailure:
		return "failed"
	default:
		return "created"
	}
}
