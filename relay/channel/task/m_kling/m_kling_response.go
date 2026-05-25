package mkling

import (
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type submitResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID string `json:"task_id"`
	} `json:"data"`
}

type queryResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID        string `json:"task_id"`
		TaskStatus    string `json:"task_status"`
		TaskStatusMsg string `json:"task_status_msg"`
		TaskResult    struct {
			Videos []videoItem `json:"videos"`
		} `json:"task_result"`
		CreatedAt int64 `json:"created_at"`
		UpdatedAt int64 `json:"updated_at"`
	} `json:"data"`
}

type videoItem struct {
	ID  string `json:"id,omitempty"`
	URL string `json:"url,omitempty"`
}

func BuildSubmitResponse(publicTaskID string) ([]byte, error) {
	resp := submitResponse{
		Code:    0,
		Message: "success",
	}
	resp.Data.TaskID = publicTaskID
	return common.Marshal(resp)
}

func BuildQueryResponse(task *model.Task) ([]byte, error) {
	if task == nil {
		return nil, fmt.Errorf("task is nil")
	}
	resp := queryResponse{
		Code:    0,
		Message: "success",
	}
	resp.Data.TaskID = task.TaskID
	resp.Data.TaskStatus = mapTaskStatus(task.Status)
	resp.Data.TaskStatusMsg = task.FailReason
	resp.Data.CreatedAt = time.Now().Unix()
	resp.Data.UpdatedAt = time.Now().Unix()
	if resultURL := task.GetResultURL(); resultURL != "" {
		resp.Data.TaskResult.Videos = []videoItem{{
			ID:  task.TaskID,
			URL: resultURL,
		}}
	} else {
		resp.Data.TaskResult.Videos = []videoItem{}
	}
	return common.Marshal(resp)
}

func mapTaskStatus(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusSubmitted, model.TaskStatusQueued:
		return "submitted"
	case model.TaskStatusInProgress:
		return "processing"
	case model.TaskStatusSuccess:
		return "succeed"
	case model.TaskStatusFailure:
		return "failed"
	default:
		return "submitted"
	}
}
