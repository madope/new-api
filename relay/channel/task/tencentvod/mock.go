package tencentvod

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/QuantumNous/new-api/common"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

// MockConfig 配置结构
type MockConfig struct {
	Enabled  bool   // 是否启用 mock
	DelayMs  int    // 模拟延迟（毫秒）
	FailRate float64 // 失败率 (0.0-1.0)
}

// MockService 统一管理 mock 逻辑
type MockService struct {
	config MockConfig
}

// NewMockService 创建 mock 服务
func NewMockService(apiKey string) *MockService {
	return &MockService{
		config: MockConfig{
			Enabled: strings.HasPrefix(apiKey, "mock-"),
		},
	}
}

// IsEnabled 检查是否启用 mock（按优先级）
func (m *MockService) IsEnabled(c *gin.Context) bool {
	// 优先级1：URL 查询参数 ?mock=true
	if c.Query("mock") == "true" {
		common.SysLog(fmt.Sprintf("Mock mode enabled via URL parameter, path: %s", c.Request.URL.Path))
		return true
	}

	// 优先级2：请求体中的 metadata.mock
	if v, exists := c.Get("task_request"); exists {
		if req, ok := v.(relaycommon.TaskSubmitReq); ok {
			if req.Metadata != nil {
				if mockVal, ok := req.Metadata["mock"]; ok {
					switch m := mockVal.(type) {
					case bool:
						if m {
							common.SysLog(fmt.Sprintf("Mock mode enabled via metadata, path: %s", c.Request.URL.Path))
						}
						return m
					case string:
						enabled := strings.EqualFold(m, "true") || m == "1"
						if enabled {
							common.SysLog(fmt.Sprintf("Mock mode enabled via metadata, path: %s", c.Request.URL.Path))
						}
						return enabled
					}
				}
			}
		}
	}

	// 优先级3：apiKey 配置
	if m.config.Enabled {
		common.SysLog(fmt.Sprintf("Mock mode enabled via apiKey configuration, path: %s", c.Request.URL.Path))
	}
	return m.config.Enabled
}

// IsEnabledFromBody 检查 FetchTask 的 mock 状态
func (m *MockService) IsEnabledFromBody(body map[string]any) bool {
	if mockVal, ok := body["mock"]; ok {
		switch m := mockVal.(type) {
		case bool:
			return m
		case string:
			return strings.EqualFold(m, "true") || m == "1"
		}
	}
	return false
}

// CreateTaskResponse 创建模拟的任务提交响应
func (m *MockService) CreateTaskResponse() (*http.Response, error) {
	// 模拟延迟
	if m.config.DelayMs > 0 {
		time.Sleep(time.Duration(m.config.DelayMs) * time.Millisecond)
	}

	// 模拟失败
	if m.config.FailRate > 0 && rand.Float64() < m.config.FailRate {
		common.SysLog("Mock mode: simulating failure")
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(`{"Response":{"Error":{"Code":"MockError","Message":"Simulated failure"}}}`)),
			Header:     make(http.Header),
		}, nil
	}

	mockResp := CreateAigcVideoTaskResponse{
		Response: struct {
			TaskID   string `json:"TaskId"`
			RequestID string `json:"RequestId"`
		}{
			TaskID:   fmt.Sprintf("mock-task-%d", time.Now().UnixNano()),
			RequestID: fmt.Sprintf("mock-request-%d", time.Now().UnixNano()),
		},
	}

	data, err := common.Marshal(mockResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mock response: %w", err)
	}

	common.SysLog(fmt.Sprintf("Mock task created successfully, task_id: %s", mockResp.Response.TaskID))

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}, nil
}

// CreateFetchResponse 创建模拟的任务状态查询响应
func (m *MockService) CreateFetchResponse(body map[string]any) (*http.Response, error) {
	taskID, _ := body["task_id"].(string)

	// 获取模拟状态（支持多种场景）
	mockStatus := getMockStatus(body)

	// 模拟延迟
	if m.config.DelayMs > 0 {
		time.Sleep(time.Duration(m.config.DelayMs) * time.Millisecond)
	}

	var status, taskStatus string
	var progress int
	var output AigcVideoTaskOutput

	switch mockStatus {
	case "pending", "processing":
		status = "RUNNING"
		taskStatus = "PROCESSING"
		progress = 50
	case "failed":
		status = "FINISH"
		taskStatus = "FAIL"
		progress = 0
	default: // success
		status = "FINISH"
		taskStatus = "SUCCESS"
		progress = 100
		output = AigcVideoTaskOutput{
			FileInfos: []AigcVideoOutputFileInfo{
				{
					FileType: "mp4",
					FileURL:  "https://example.com/mock-video.mp4",
				},
			},
		}
	}

	mockResp := DescribeTaskDetailResponse{
		Response: DescribeTaskDetailResponseBody{
			RequestID: fmt.Sprintf("mock-request-%d", time.Now().UnixNano()),
			CreateTime:    time.Now().UTC().Format(time.RFC3339),
			FinishTime:    time.Now().UTC().Format(time.RFC3339),
			Status:        status,
			AigcVideoTask: &AigcVideoTask{
				TaskID:   taskID,
				Status:   taskStatus,
				Progress: progress,
				Output:   output,
				ErrCode:  0,
				Message:  "",
			},
		},
	}

	data, err := common.Marshal(mockResp)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal mock fetch response: %w", err)
	}

	common.SysLog(fmt.Sprintf("Mock task status fetched, task_id: %s, status: %s", taskID, taskStatus))

	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewReader(data)),
		Header:     make(http.Header),
	}, nil
}

// getMockStatus 获取模拟状态
func getMockStatus(body map[string]any) string {
	if status, ok := body["mock_status"]; ok {
		return fmt.Sprintf("%v", status)
	}
	if status, ok := body["status"]; ok {
		return fmt.Sprintf("%v", status)
	}
	return "success"
}

// AddMockHeaders 在响应中添加 mock 标识头
func (m *MockService) AddMockHeaders(c *gin.Context) {
	c.Header("X-Mock-Mode", "true")
	c.Header("X-Mock-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
}