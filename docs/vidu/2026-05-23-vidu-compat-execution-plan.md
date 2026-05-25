# m-vidu Compat API Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `new-api` 增加 `/m-vidu/...` 用户侧兼容接口，请求侧参考 `volces` 的轻量改写模式，仅抽取通用字段并将 Vidu 特有字段原样保留到 `metadata`，响应侧返回 Vidu 官方核心结构。

**Architecture:** 兼容层只做三件事：读取原始请求 `map[string]any`、抽取少量统一字段写回 `/v1/video/generations` 请求体、在提交/查询出口将结果改写为 Vidu 核心响应。不上强类型 Vidu DTO，不派生 `file_infos`、`last_frame_url` 等 provider-specific 字段，尽量让上游 adaptor 直接读取原始 `metadata`。

**Tech Stack:** Go, Gin, `RelayTask`, `RelayTaskFetch`, `common/json.go`, existing task adaptor framework

---

## File Structure

### New files

- `middleware/m_vidu_adapter.go`
  - 用户侧 `/m-vidu/...` 请求改写入口，参考 `middleware/volces_seedance_adapter.go`
- `middleware/m_vidu_adapter_test.go`
  - 4 个路由的请求改写测试与未知字段透传测试
- `relay/channel/task/m_vidu/m_vidu_response.go`
  - Vidu 核心提交响应、查询响应构造
- `relay/channel/task/m_vidu/m_vidu_test.go`
  - 核心响应结构测试

### Modified files

- `router/video-router.go`
  - 新增 `/m-vidu/ent/v2/...` 路由
- `middleware/distributor.go`
  - 识别 `/m-vidu/ent/v2/...` 并设置 video relay mode
- `relay/relay_task.go`
  - `/m-vidu/ent/v2/tasks/:task_id/creations` 返回 Vidu 核心查询结构
- `controller/swag_video.go`
  - 补 `/m-vidu/...` swagger 占位

### Existing references

- `middleware/volces_seedance_adapter.go`
- `middleware/kling_adapter.go`
- `relay/common/relay_info.go`
- `constant/task.go`
- `relay/relay_task.go`
- `dto/openai_video.go`

---

## Task 1: 建立 `m_vidu` 请求改写层

**Files:**
- Create: `middleware/m_vidu_adapter.go`
- Test: `middleware/m_vidu_adapter_test.go`

- [ ] **Step 1: 写失败测试，验证 `text2video` 会被改写到统一视频请求**

```go
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
		strings.NewReader(`{"model":"viduq2","prompt":"a cat running","duration":4,"future_flag":"beta"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Contains(t, w.Body.String(), `"model":"viduq2"`)
	require.Contains(t, w.Body.String(), `"prompt":"a cat running"`)
	require.Contains(t, w.Body.String(), `"duration":4`)
	require.Contains(t, w.Body.String(), `"future_flag":"beta"`)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./middleware -run TestMViduRequestConvertText2Video -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 实现最小版本 `MViduRequestConvert` 和 `convertMViduRequestToUnified`**

```go
package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
)

func MViduRequestConvert() func(c *gin.Context) {
	return func(c *gin.Context) {
		common.SetContextKey(c, constant.ContextKeyVolcesCompat, false)

		var originalReq map[string]any
		if err := common.UnmarshalBodyReusable(c, &originalReq); err != nil {
			abortWithOpenAiMessage(c, 400, "Invalid request body")
			return
		}

		requestType := detectMViduRequestType(c.Request.URL.Path)
		if requestType == "" {
			c.Next()
			return
		}

		unifiedReq, action, err := convertMViduRequestToUnified(originalReq, requestType)
		if err != nil {
			abortWithOpenAiMessage(c, 400, err.Error())
			return
		}

		jsonData, err := common.Marshal(unifiedReq)
		if err != nil {
			abortWithOpenAiMessage(c, 500, "Failed to marshal request body")
			return
		}
		if err := common.ReplaceRequestBody(c, jsonData); err != nil {
			abortWithOpenAiMessage(c, 500, "Failed to replace request body")
			return
		}

		c.Request.URL.Path = "/v1/video/generations"
		c.Set("action", action)
		c.Next()
	}
}

func detectMViduRequestType(path string) string {
	switch {
	case strings.HasSuffix(path, "/text2video"):
		return "text2video"
	case strings.HasSuffix(path, "/img2video"):
		return "img2video"
	case strings.HasSuffix(path, "/start-end2video"):
		return "start-end2video"
	case strings.HasSuffix(path, "/reference2video"):
		return "reference2video"
	default:
		return ""
	}
}

func convertMViduRequestToUnified(originalReq map[string]any, requestType string) (map[string]any, string, error) {
	model, _ := originalReq["model"].(string)
	if strings.TrimSpace(model) == "" {
		return nil, "", fmt.Errorf("model is required")
	}

	metadata := make(map[string]any, len(originalReq))
	for k, v := range originalReq {
		metadata[k] = v
	}

	unifiedReq := map[string]any{
		"model": model,
	}

	if prompt, _ := originalReq["prompt"].(string); strings.TrimSpace(prompt) != "" {
		unifiedReq["prompt"] = prompt
		delete(metadata, "prompt")
	}

	if images, ok := originalReq["images"].([]any); ok && len(images) > 0 {
		unifiedReq["images"] = images
		delete(metadata, "images")
	}

	if duration, ok := originalReq["duration"]; ok {
		unifiedReq["duration"] = duration
		delete(metadata, "duration")
	}

	delete(metadata, "model")
	if len(metadata) > 0 {
		unifiedReq["metadata"] = metadata
	}

	return unifiedReq, mapMViduAction(requestType), nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./middleware -run TestMViduRequestConvertText2Video -count=1
```

Expected:

```text
ok
```

- [ ] **Step 5: 扩展测试，验证 `start-end2video`、`reference2video`、未知字段透传**

```go
func TestMViduRequestConvertOtherRoutes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(MViduRequestConvert())
	router.POST("/m-vidu/ent/v2/start-end2video", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Data(http.StatusOK, "application/json", body)
	})
	router.POST("/m-vidu/ent/v2/reference2video", func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		c.Data(http.StatusOK, "application/json", body)
	})

	startEndReq := httptest.NewRequest(
		http.MethodPost,
		"/m-vidu/ent/v2/start-end2video",
		strings.NewReader(`{"model":"viduq2","images":["https://example.com/a.png","https://example.com/b.png"],"camera":"pan"}`),
	)
	startEndReq.Header.Set("Content-Type", "application/json")
	startEndW := httptest.NewRecorder()
	router.ServeHTTP(startEndW, startEndReq)
	require.Equal(t, http.StatusOK, startEndW.Code)
	require.Contains(t, startEndW.Body.String(), `"images":["https://example.com/a.png","https://example.com/b.png"]`)
	require.Contains(t, startEndW.Body.String(), `"camera":"pan"`)

	referenceReq := httptest.NewRequest(
		http.MethodPost,
		"/m-vidu/ent/v2/reference2video",
		strings.NewReader(`{"model":"viduq2","images":["https://example.com/ref.png"],"videos":["https://example.com/ref.mp4"],"subjects":[{"name":"hero"}]}`),
	)
	referenceReq.Header.Set("Content-Type", "application/json")
	referenceW := httptest.NewRecorder()
	router.ServeHTTP(referenceW, referenceReq)
	require.Equal(t, http.StatusOK, referenceW.Code)
	require.Contains(t, referenceW.Body.String(), `"videos":["https://example.com/ref.mp4"]`)
	require.Contains(t, referenceW.Body.String(), `"subjects":[{"name":"hero"}]`)
}
```

- [ ] **Step 6: 跑 middleware 全部测试**

Run:

```bash
go test ./middleware -count=1
```

Expected:

```text
ok
```

- [ ] **Step 7: Commit**

```bash
git add middleware/m_vidu_adapter.go middleware/m_vidu_adapter_test.go
git commit -m "feat: add m-vidu request conversion middleware"
```

---

## Task 2: 接入路由与分发

**Files:**
- Modify: `router/video-router.go`
- Modify: `middleware/distributor.go`

- [ ] **Step 1: 写失败测试，验证 `/m-vidu/...` 会走 video submit relay mode**

```go
func TestGetModelRequestForMViduSubmit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodPost, "/m-vidu/ent/v2/text2video", strings.NewReader(`{"model":"viduq2"}`))
	c.Request.Header.Set("Content-Type", "application/json")

	modelReq, shouldSelectChannel, taskErr := getModelRequest(c)
	require.NoError(t, taskErr)
	require.True(t, shouldSelectChannel)
	require.Equal(t, "viduq2", modelReq.Model)
	require.Equal(t, relayconstant.RelayModeVideoSubmit, c.GetInt("relay_mode"))
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./middleware -run TestGetModelRequestForMViduSubmit -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 在 `router/video-router.go` 增加 `/m-vidu/ent/v2/...` 路由**

```go
mViduRouter := router.Group("/m-vidu/ent/v2")
mViduRouter.Use(middleware.RouteTag("relay"))
mViduRouter.Use(middleware.MViduRequestConvert(), middleware.TokenAuth(), middleware.Distribute())
{
	mViduRouter.POST("/text2video", controller.RelayTask)
	mViduRouter.POST("/img2video", controller.RelayTask)
	mViduRouter.POST("/start-end2video", controller.RelayTask)
	mViduRouter.POST("/reference2video", controller.RelayTask)
	mViduRouter.GET("/tasks/:task_id/creations", controller.RelayTaskFetch)
}
```

- [ ] **Step 4: 在 `middleware/distributor.go` 增加 `/m-vidu/ent/v2/...` 识别**

```go
} else if strings.Contains(c.Request.URL.Path, "/m-vidu/ent/v2/") {
	relayMode := relayconstant.RelayModeUnknown
	if c.Request.Method == http.MethodPost {
		req, err := getModelFromRequest(c)
		if err != nil {
			return nil, false, err
		}
		modelRequest.Model = req.Model
		relayMode = relayconstant.RelayModeVideoSubmit
	} else if c.Request.Method == http.MethodGet {
		relayMode = relayconstant.RelayModeVideoFetchByID
		shouldSelectChannel = false
	}
	c.Set("relay_mode", relayMode)
```

- [ ] **Step 5: 运行测试确认通过**

Run:

```bash
go test ./middleware -run TestGetModelRequestForMViduSubmit -count=1
```

Expected:

```text
ok
```

- [ ] **Step 6: 运行轻量回归**

Run:

```bash
go test ./middleware ./router -run TestDoesNotExist -count=1
```

Expected:

```text
ok
```

- [ ] **Step 7: Commit**

```bash
git add router/video-router.go middleware/distributor.go
git commit -m "feat: add m-vidu routes and distributor wiring"
```

---

## Task 3: 建立 Vidu 核心响应构造

**Files:**
- Create: `relay/channel/task/m_vidu/m_vidu_response.go`
- Test: `relay/channel/task/m_vidu/m_vidu_test.go`

- [ ] **Step 1: 写失败测试，验证提交响应和查询响应的核心字段**

```go
func TestBuildMViduResponses(t *testing.T) {
	submitBody, err := BuildSubmitResponse("task_123", "created")
	require.NoError(t, err)
	require.Contains(t, string(submitBody), `"task_id":"task_123"`)
	require.Contains(t, string(submitBody), `"state":"created"`)

	task := &model.Task{
		TaskID:   "task_123",
		Status:   model.TaskStatusSuccess,
		Progress: "100%",
	}
	task.PrivateData.ResultURL = "https://example.com/video.mp4"

	queryBody, err := BuildQueryResponse(task)
	require.NoError(t, err)
	require.Contains(t, string(queryBody), `"id":"task_123"`)
	require.Contains(t, string(queryBody), `"state":"success"`)
	require.Contains(t, string(queryBody), `"creations"`)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./relay/channel/task/m_vidu -run TestBuildMViduResponses -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 写最小响应构造实现**

```go
package mvidu

import (
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

type SubmitResponse struct {
	TaskID string `json:"task_id"`
	State  string `json:"state"`
}

type Creation struct {
	URL string `json:"url,omitempty"`
}

type QueryResponse struct {
	ID        string     `json:"id"`
	State     string     `json:"state"`
	ErrCode   string     `json:"err_code,omitempty"`
	Creations []Creation `json:"creations,omitempty"`
}

func BuildSubmitResponse(taskID, state string) ([]byte, error) {
	return common.Marshal(SubmitResponse{
		TaskID: taskID,
		State:  state,
	})
}

func BuildQueryResponse(task *model.Task) ([]byte, error) {
	resp := QueryResponse{
		ID:    task.TaskID,
		State: mapTaskStatus(task.Status),
	}
	if task.Status == model.TaskStatusFailure && task.FailReason != "" {
		resp.ErrCode = task.FailReason
	}
	if url := task.GetResultURL(); url != "" {
		resp.Creations = []Creation{{URL: url}}
	}
	return common.Marshal(resp)
}

func mapTaskStatus(status model.TaskStatus) string {
	switch status {
	case model.TaskStatusSubmitted, model.TaskStatusQueued:
		return "created"
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
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./relay/channel/task/m_vidu -run TestBuildMViduResponses -count=1
```

Expected:

```text
ok
```

- [ ] **Step 5: Commit**

```bash
git add relay/channel/task/m_vidu/m_vidu_response.go relay/channel/task/m_vidu/m_vidu_test.go
git commit -m "feat: add m-vidu core response builders"
```

---

## Task 4: 查询响应接入 `RelayTaskFetch`

**Files:**
- Modify: `relay/relay_task.go`

- [ ] **Step 1: 写失败测试或最小手工验证目标**

目标：

- `/m-vidu/ent/v2/tasks/:task_id/creations`
- 不再返回通用 `TaskResponse[TaskDto]`
- 返回 `BuildQueryResponse` 产物

- [ ] **Step 2: 在 `videoFetchByIDRespBodyBuilder` 增加路径分支**

```go
if strings.HasPrefix(c.Request.RequestURI, "/m-vidu/ent/v2/tasks/") {
	respBody, err = taskmvidu.BuildQueryResponse(originTask)
	if err != nil {
		taskResp = service.TaskErrorWrapper(err, "build_m_vidu_response_failed", http.StatusInternalServerError)
	}
	return
}
```

- [ ] **Step 3: 运行轻量回归**

Run:

```bash
go test ./relay ./controller -run TestDoesNotExist -count=1
```

Expected:

```text
ok
```

- [ ] **Step 4: Commit**

```bash
git add relay/relay_task.go
git commit -m "feat: add m-vidu task fetch response branch"
```

---

## Task 5: Swagger 占位

**Files:**
- Modify: `controller/swag_video.go`

- [ ] **Step 1: 增加 `/m-vidu/...` 的 swagger 函数占位**

```go
// MViduText2VideoGenerations
// @Summary m-vidu 文生视频
// @Tags Video
// @Accept json
// @Produce json
// @Router /m-vidu/ent/v2/text2video [post]
func MViduText2VideoGenerations(c *gin.Context) {}

// MViduImg2VideoGenerations
// @Summary m-vidu 图生视频
// @Tags Video
// @Accept json
// @Produce json
// @Router /m-vidu/ent/v2/img2video [post]
func MViduImg2VideoGenerations(c *gin.Context) {}

// MViduStartEnd2VideoGenerations
// @Summary m-vidu 首尾帧生视频
// @Tags Video
// @Accept json
// @Produce json
// @Router /m-vidu/ent/v2/start-end2video [post]
func MViduStartEnd2VideoGenerations(c *gin.Context) {}

// MViduReference2VideoGenerations
// @Summary m-vidu 参考图生视频
// @Tags Video
// @Accept json
// @Produce json
// @Router /m-vidu/ent/v2/reference2video [post]
func MViduReference2VideoGenerations(c *gin.Context) {}

// MViduTaskCreations
// @Summary m-vidu 任务查询
// @Tags Video
// @Accept json
// @Produce json
// @Param task_id path string true "Task ID"
// @Router /m-vidu/ent/v2/tasks/{task_id}/creations [get]
func MViduTaskCreations(c *gin.Context) {}
```

- [ ] **Step 2: 运行轻量回归**

Run:

```bash
go test ./controller -run TestDoesNotExist -count=1
```

Expected:

```text
ok
```

- [ ] **Step 3: Commit**

```bash
git add controller/swag_video.go
git commit -m "docs: add m-vidu swagger placeholders"
```

---

## Task 6: 端到端回归

**Files:**
- Verify only

- [ ] **Step 1: 跑包级测试**

Run:

```bash
go test ./middleware ./relay/channel/task/m_vidu -count=1
```

Expected:

```text
ok
```

- [ ] **Step 2: 跑集成级轻量回归**

Run:

```bash
go test ./middleware ./router ./relay ./controller -run TestDoesNotExist -count=1
```

Expected:

```text
ok
```

- [ ] **Step 3: 手工 curl 验证请求改写**

```bash
curl -X POST "http://localhost:3000/m-vidu/ent/v2/text2video" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_NEWAPI_TOKEN" \
  -d '{
    "model": "viduq2",
    "prompt": "a cat running",
    "duration": 4,
    "future_flag": "beta"
  }'
```

Expected:

- 接口进入现有视频任务链路
- `future_flag` 没有在 compat 层被丢掉

- [ ] **Step 4: Commit**

```bash
git status
git add middleware/m_vidu_adapter.go middleware/m_vidu_adapter_test.go \
  relay/channel/task/m_vidu/m_vidu_response.go relay/channel/task/m_vidu/m_vidu_test.go \
  router/video-router.go middleware/distributor.go relay/relay_task.go controller/swag_video.go
git commit -m "feat: add lightweight m-vidu compat api"
```

---

## Self-Review

### Spec coverage

- `/m-vidu/...` 路由：Task 2
- 参考 `volces` 的轻量兼容：Task 1
- 只抽通用字段：Task 1
- 特有字段直接进 `metadata`：Task 1
- 不派生 `file_infos` / `last_frame_url`：Task 1
- 核心提交/查询响应：Task 3 + Task 4
- swagger：Task 5

### Placeholder scan

- 无 `TBD` / `TODO`
- 所有代码任务都给出了实际代码
- 所有验证都给了命令和预期

### Type consistency

- `MViduRequestConvert`、`convertMViduRequestToUnified`、`BuildSubmitResponse`、`BuildQueryResponse` 在各任务里命名一致

Plan complete and saved to `docs/vidu/2026-05-23-vidu-compat-execution-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
