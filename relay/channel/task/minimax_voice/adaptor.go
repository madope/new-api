package minimax_voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

type TaskAdaptor struct {
	ChannelType int
	apiKey      string
	baseURL     string
}

type baseResp struct {
	StatusCode int    `json:"status_code"`
	StatusMsg  string `json:"status_msg"`
}

type submitResponse struct {
	TaskID     json.Number `json:"task_id"`
	TaskToken  string      `json:"task_token"`
	FileID     json.Number `json:"file_id"`
	UsageChars int         `json:"usage_characters,omitempty"`
	BaseResp   baseResp    `json:"base_resp"`
}

type queryResponse struct {
	Status   string      `json:"status"`
	TaskID   json.Number `json:"task_id"`
	FileID   json.Number `json:"file_id"`
	BaseResp baseResp    `json:"base_resp"`
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.apiKey = info.ApiKey
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	rawBody, err := common.GetBodyStorage(c)
	if err != nil {
		return service.TaskErrorWrapper(err, "read_body_failed", http.StatusInternalServerError)
	}

	bodyBytes, err := io.ReadAll(rawBody)
	if err != nil {
		return service.TaskErrorWrapper(err, "read_body_failed", http.StatusInternalServerError)
	}

	info.Action = "voice_generate"

	c.Set("raw_request_body", bodyBytes)

	logger.LogInfo(c, fmt.Sprintf("MiniMax async voice request body: %s", string(bodyBytes)))
	return nil
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s%s", a.baseURL, SubmitEndpoint), nil
}

func (a *TaskAdaptor) BuildRequestHeader(c *gin.Context, req *http.Request, info *relaycommon.RelayInfo) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	v, exists := c.Get("raw_request_body")
	if !exists {
		return nil, fmt.Errorf("request body not found in context")
	}

	rawBody, ok := v.([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid request body type in context")
	}

	return bytes.NewReader(rawBody), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
		return
	}
	_ = resp.Body.Close()

	logger.LogInfo(c, fmt.Sprintf("MiniMax async voice response body: %s", string(responseBody)))

	var submitResp submitResponse
	if err := common.Unmarshal(responseBody, &submitResp); err != nil {
		taskErr = service.TaskErrorWrapper(fmt.Errorf("unmarshal response failed: %w, body: %s", err, responseBody), "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}

	if submitResp.BaseResp.StatusCode != 0 {
		taskErr = service.TaskErrorWrapper(
			fmt.Errorf("minimax voice api error: %s", submitResp.BaseResp.StatusMsg),
			strconv.Itoa(submitResp.BaseResp.StatusCode),
			http.StatusBadRequest,
		)
		return
	}

	// Replace upstream task_id with system public task_id
	upstreamTaskID := submitResp.TaskID.String()
	var nativeResp map[string]any
	if err := common.Unmarshal(responseBody, &nativeResp); err != nil {
		taskErr = service.TaskErrorWrapper(err, "unmarshal_response_failed", http.StatusInternalServerError)
		return
	}
	nativeResp["task_id"] = info.PublicTaskID
	modifiedBody, err := common.Marshal(nativeResp)
	if err != nil {
		taskErr = service.TaskErrorWrapper(err, "marshal_response_failed", http.StatusInternalServerError)
		return
	}

	c.Data(http.StatusOK, "application/json", modifiedBody)
	return upstreamTaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseUrl, key string, body map[string]any, proxy string) (*http.Response, error) {
	taskID, ok := body["task_id"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid task_id")
	}

	uri := fmt.Sprintf("%s%s?task_id=%s", baseUrl, QueryEndpoint, taskID)

	req, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+key)

	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var queryResp queryResponse
	if err := common.Unmarshal(respBody, &queryResp); err != nil {
		return nil, fmt.Errorf("unmarshal task result failed: %w, body: %s", err, respBody)
	}

	taskResult := relaycommon.TaskInfo{}

	if queryResp.BaseResp.StatusCode != 0 {
		taskResult.Code = queryResp.BaseResp.StatusCode
		taskResult.Reason = queryResp.BaseResp.StatusMsg
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		return &taskResult, nil
	}

	switch queryResp.Status {
	case StatusPreparing, StatusQueueing:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	case StatusProcessing:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "50%"
	case StatusSuccess:
		taskResult.Status = model.TaskStatusSuccess
		taskResult.Progress = "100%"
	case StatusFailed:
		taskResult.Status = model.TaskStatusFailure
		taskResult.Progress = "100%"
		if taskResult.Reason == "" {
			taskResult.Reason = "task failed"
		}
	default:
		taskResult.Status = model.TaskStatusInProgress
		taskResult.Progress = "30%"
	}

	return &taskResult, nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	return nil
}

func (a *TaskAdaptor) AdjustBillingOnSubmit(info *relaycommon.RelayInfo, taskData []byte) map[string]float64 {
	return nil
}

func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	return 0
}

func (a *TaskAdaptor) GetModelList() []string {
	return nil
}

func (a *TaskAdaptor) GetChannelName() string {
	return "minimax_voice"
}
