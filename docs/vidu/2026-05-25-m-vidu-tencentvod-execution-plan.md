# m-vidu TencentVODVideo Mapping Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 按 `docs/vidu/2026-05-25-m-vidu-tencentvod-parameter-mapping.md` 的约束，为 `TencentVODVideo` adaptor 实现 `m-vidu` 请求到腾讯 VOD 上游请求的参数映射，并补齐对应测试。

**Architecture:** 保持 `m-vidu` compat 层轻量，不新增 Vidu 专用 DTO，也不在 compat 层派生腾讯专用字段。所有映射只落在 `relay/channel/task/tencentvod/adaptor.go`，通过读取统一 `TaskSubmitReq` 的通用字段和原始 `metadata`，构造腾讯 VOD 的 `Prompt`、`OutputConfig`、`FileInfos`、`LastFrameUrl` 等字段；未在映射文档中列出的字段一律忽略。

**Tech Stack:** Go, Gin, task adaptor framework, `common/json.go`, `testify`, `go test`

---

## File Structure

### Modify

- `relay/channel/task/tencentvod/adaptor.go`
  - 唯一实现 `m-vidu -> TencentVODVideo` 映射逻辑的地方
  - 只允许增加读取 `metadata` 和构造腾讯 VOD 请求的代码
- `relay/channel/task/tencentvod/adaptor_test.go`
  - 为映射规则增加回归测试

### Reference

- `docs/vidu/2026-05-25-m-vidu-tencentvod-parameter-mapping.md`
  - 唯一验收依据
- `relay/common/relay_info.go`
  - `TaskSubmitReq` 定义
- `middleware/m_vidu_adapter.go`
  - `m-vidu` 请求进入统一视频任务链路时保留下来的字段形态

### Non-goals

- 不修改 `middleware/m_vidu_adapter.go`
- 不修改 `relay/channel/task/m_vidu/*`
- 不修改 `docs/vidu/2026-05-25-m-vidu-tencentvod-parameter-mapping.md`
- 不实现映射文档之外的“猜测型”字段桥接

---

## Task 1: 为共通字段映射补失败测试

**Files:**
- Modify: `relay/channel/task/tencentvod/adaptor_test.go`
- Test: `relay/channel/task/tencentvod/adaptor_test.go`

- [ ] **Step 1: 写失败测试，锁定共通字段映射**

```go
func TestConvertToRequestPayloadMapsMViduCommonFields(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:    "vidu-q3",
		Prompt:   "cinematic scene",
		Duration: 8,
		Metadata: map[string]interface{}{
			"seed":         123,
			"payload":      "trace-1",
			"resolution":   "1080p",
			"aspect_ratio": "16:9",
			"audio":        true,
			"bgm":          false,
			"off_peak":     true,
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.NotNil(t, payload.OutputConfig)
	assert.Equal(t, 123, payload.Seed)
	assert.Equal(t, "trace-1", payload.SessionContext)
	assert.Equal(t, "1080p", payload.OutputConfig.Resolution)
	assert.Equal(t, "16:9", payload.OutputConfig.AspectRatio)
	assert.Equal(t, "Enabled", payload.OutputConfig.AudioGeneration)
	assert.Equal(t, "Disabled", payload.OutputConfig.EnableBGM)
	assert.Equal(t, "Enabled", payload.OutputConfig.OffPeak)
	assert.Equal(t, 8, payload.OutputConfig.Duration)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadMapsMViduCommonFields -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 在 `applyMViduRawMappings` 中实现共通字段映射**

```go
func applyMViduRawMappings(meta *taskMeta, metadata map[string]interface{}) {
	if meta == nil || metadata == nil {
		return
	}
	ensureOutputConfig := func() *AigcVideoOutputConfig {
		if meta.OutputConfig == nil {
			meta.OutputConfig = &AigcVideoOutputConfig{}
		}
		return meta.OutputConfig
	}

	if meta.Seed == nil {
		if seed := intPointerValue(metadata["seed"]); seed != nil {
			meta.Seed = seed
		}
	}
	if meta.SessionContext == "" {
		meta.SessionContext = stringValue(metadata, "payload")
	}
	if aspectRatio := stringValue(metadata, "aspect_ratio"); aspectRatio != "" {
		outputConfig := ensureOutputConfig()
		if outputConfig.AspectRatio == "" {
			outputConfig.AspectRatio = aspectRatio
		}
	}
	if resolution := stringValue(metadata, "resolution"); resolution != "" {
		outputConfig := ensureOutputConfig()
		if outputConfig.Resolution == "" {
			outputConfig.Resolution = resolution
		}
	}
	if audio, ok := boolValue(metadata["audio"]); ok {
		outputConfig := ensureOutputConfig()
		if outputConfig.AudioGeneration == "" {
			outputConfig.AudioGeneration = enableFlag(audio)
		}
	}
	if bgm, ok := boolValue(metadata["bgm"]); ok {
		outputConfig := ensureOutputConfig()
		if outputConfig.EnableBGM == "" {
			outputConfig.EnableBGM = enableFlag(bgm)
		}
	}
	if offPeak, ok := boolValue(metadata["off_peak"]); ok {
		outputConfig := ensureOutputConfig()
		if outputConfig.OffPeak == "" {
			outputConfig.OffPeak = enableFlag(offPeak)
		}
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadMapsMViduCommonFields -count=1
```

Expected:

```text
ok
```

- [ ] **Step 5: 提交**

```bash
git add relay/channel/task/tencentvod/adaptor.go relay/channel/task/tencentvod/adaptor_test.go
git commit -m "feat: map common m-vidu fields to tencent vod"
```

---

## Task 2: 为 `img2video` 和 `start-end2video` 映射补测试和实现

**Files:**
- Modify: `relay/channel/task/tencentvod/adaptor.go`
- Modify: `relay/channel/task/tencentvod/adaptor_test.go`
- Test: `relay/channel/task/tencentvod/adaptor_test.go`

- [ ] **Step 1: 写失败测试，锁定图生和首尾帧映射**

```go
func TestConvertToRequestPayloadMapsMViduStartEndImages(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "smooth transition",
		Images: []string{
			"https://example.com/first.png",
			"https://example.com/last.png",
		},
		Metadata: map[string]interface{}{
			"images": []interface{}{
				"https://example.com/first.png",
				"https://example.com/last.png",
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionFirstTailGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	require.Len(t, payload.FileInfos, 1)
	assert.Equal(t, "FirstFrame", payload.FileInfos[0].Usage)
	assert.Equal(t, "https://example.com/first.png", payload.FileInfos[0].URL)
	assert.Equal(t, "https://example.com/last.png", payload.LastFrameURL)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadMapsMViduStartEndImages -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 在 `convertToRequestPayload` 中实现首尾帧映射**

```go
action := detectAction(req, meta)
if info != nil && info.TaskRelayInfo != nil && info.Action != "" {
	action = info.Action
}
if action == constant.TaskActionFirstTailGenerate && meta.LastFrameURL == "" {
	rawImages := stringSliceValue(req.Metadata["images"])
	if len(rawImages) >= 2 && strings.TrimSpace(rawImages[1]) != "" {
		meta.LastFrameURL = rawImages[1]
	}
}

fileInfos := cloneFileInfos(meta.FileInfos)
if action == constant.TaskActionFirstTailGenerate && len(fileInfos) == 0 && len(req.Images) > 0 {
	fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
		Type:     "Url",
		Category: "Image",
		URL:      req.Images[0],
		Usage:    "FirstFrame",
	})
} else if len(fileInfos) == 0 && len(req.Images) > 0 {
	for _, image := range req.Images {
		if strings.TrimSpace(image) == "" {
			continue
		}
		fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
			Type:     "Url",
			Category: "Image",
			URL:      image,
			Usage:    "Reference",
		})
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadMapsMViduStartEndImages -count=1
```

Expected:

```text
ok
```

- [ ] **Step 5: 提交**

```bash
git add relay/channel/task/tencentvod/adaptor.go relay/channel/task/tencentvod/adaptor_test.go
git commit -m "feat: map m-vidu image and start-end requests"
```

---

## Task 3: 为 `reference2video` 非主体调用映射补测试和实现

**Files:**
- Modify: `relay/channel/task/tencentvod/adaptor.go`
- Modify: `relay/channel/task/tencentvod/adaptor_test.go`
- Test: `relay/channel/task/tencentvod/adaptor_test.go`

- [ ] **Step 1: 写失败测试，锁定顶层 `videos[]` 和 `images[] + videos[]` 的组合映射**

```go
func TestConvertToRequestPayloadMapsMViduReferenceVideos(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "cinematic motion",
		Metadata: map[string]interface{}{
			"videos": []interface{}{
				"https://example.com/ref.mp4",
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReferenceGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.Len(t, payload.FileInfos, 1)
	assert.Equal(t, "Video", payload.FileInfos[0].Category)
	assert.Equal(t, "https://example.com/ref.mp4", payload.FileInfos[0].URL)
}

func TestConvertToRequestPayloadMapsMViduReferenceImagesAndVideos(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "reference mix",
		Images: []string{
			"https://example.com/ref.png",
		},
		Metadata: map[string]interface{}{
			"images": []interface{}{
				"https://example.com/ref.png",
			},
			"videos": []interface{}{
				"https://example.com/ref.mp4",
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReferenceGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.Len(t, payload.FileInfos, 2)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./relay/channel/task/tencentvod -run 'TestConvertToRequestPayloadMapsMViduReference(Videos|ImagesAndVideos)' -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 在 `convertToRequestPayload` 中实现非主体调用映射**

```go
func hasExplicitFileInfos(metadata map[string]interface{}) bool {
	if metadata == nil {
		return false
	}
	raw, ok := metadata["file_infos"]
	if !ok || raw == nil {
		return false
	}
	items, err := parseFileInfos(raw)
	return err == nil && len(items) > 0
}

if action == constant.TaskActionReferenceGenerate && !hasExplicitFileInfos(req.Metadata) {
	for _, videoURL := range stringSliceValue(req.Metadata["videos"]) {
		if strings.TrimSpace(videoURL) == "" {
			continue
		}
		fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
			Type:     "Url",
			Category: "Video",
			URL:      videoURL,
		})
	}
}
```

- [ ] **Step 4: 运行测试确认通过**

Run:

```bash
go test ./relay/channel/task/tencentvod -run 'TestConvertToRequestPayloadMapsMViduReference(Videos|ImagesAndVideos)' -count=1
```

Expected:

```text
ok
```

- [ ] **Step 5: 提交**

```bash
git add relay/channel/task/tencentvod/adaptor.go relay/channel/task/tencentvod/adaptor_test.go
git commit -m "feat: map m-vidu reference images and videos"
```

---

## Task 4: 为 `reference2video` 主体调用映射补测试和实现

**Files:**
- Modify: `relay/channel/task/tencentvod/adaptor.go`
- Modify: `relay/channel/task/tencentvod/adaptor_test.go`
- Test: `relay/channel/task/tencentvod/adaptor_test.go`

- [ ] **Step 1: 写失败测试，锁定 `subjects` 只映射到 `FileInfos`**

```go
func TestConvertToRequestPayloadMapsMViduSubjectsToFileInfosOnly(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "subject showcase",
		Metadata: map[string]interface{}{
			"subjects": []interface{}{
				map[string]interface{}{
					"name":      "hero",
					"server_id": "obj_123",
					"voice_id":  "voice_1",
					"images": []interface{}{
						"https://example.com/hero-1.png",
					},
					"videos": []interface{}{
						"https://example.com/hero-1.mp4",
					},
				},
			},
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
		TaskRelayInfo: &relaycommon.TaskRelayInfo{
			Action: constant.TaskActionReferenceGenerate,
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	assert.Empty(t, payload.SubjectInfos)
	require.Len(t, payload.FileInfos, 2)
	assert.Equal(t, "obj_123", payload.FileInfos[0].ObjectID)
	assert.Equal(t, "hero", payload.FileInfos[0].Text)
	assert.Equal(t, "voice_1", payload.FileInfos[0].VoiceID)
	assert.Equal(t, "Reference", payload.FileInfos[0].Usage)
}
```

- [ ] **Step 2: 跑测试确认失败**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadMapsMViduSubjectsToFileInfosOnly -count=1
```

Expected:

```text
FAIL
```

- [ ] **Step 3: 添加 `parseMViduSubjects`，只读取文档中列出的字段**

```go
func parseMViduSubjects(raw any) ([]AigcVideoTaskInputFileInfo, error) {
	if raw == nil {
		return nil, nil
	}
	data, err := common.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	if err := common.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	fileInfos := make([]AigcVideoTaskInputFileInfo, 0)
	for _, item := range items {
		name := stringValue(item, "name", "Name")
		objectID := stringValue(item, "server_id", "ServerId")
		if objectID == "" {
			objectID = name
		}
		voiceID := stringValue(item, "voice_id", "VoiceId")
		text := name
		for _, imageURL := range stringSliceValue(item["images"]) {
			if strings.TrimSpace(imageURL) == "" {
				continue
			}
			fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
				Type:     "Url",
				Category: "Image",
				URL:      imageURL,
				ObjectID: objectID,
				VoiceID:  voiceID,
				Text:     text,
				Usage:    "Reference",
			})
		}
		for _, videoURL := range stringSliceValue(item["videos"]) {
			if strings.TrimSpace(videoURL) == "" {
				continue
			}
			fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
				Type:     "Url",
				Category: "Video",
				URL:      videoURL,
				ObjectID: objectID,
				VoiceID:  voiceID,
				Text:     text,
			})
		}
	}
	return fileInfos, nil
}
```

- [ ] **Step 4: 在 `convertToRequestPayload` 中仅对 `referenceGenerate` 使用 `subjects`**

```go
if action == constant.TaskActionReferenceGenerate && !hasExplicitFileInfos(req.Metadata) {
	if subjectFileInfos, err := parseMViduSubjects(req.Metadata["subjects"]); err == nil && len(subjectFileInfos) > 0 {
		fileInfos = append(fileInfos, subjectFileInfos...)
	}
}
```

- [ ] **Step 5: 运行测试确认通过**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadMapsMViduSubjectsToFileInfosOnly -count=1
```

Expected:

```text
ok
```

- [ ] **Step 6: 提交**

```bash
git add relay/channel/task/tencentvod/adaptor.go relay/channel/task/tencentvod/adaptor_test.go
git commit -m "feat: map m-vidu subject references to file infos"
```

---

## Task 5: 为忽略策略和编译级回归补验证

**Files:**
- Modify: `relay/channel/task/tencentvod/adaptor_test.go`
- Test: `relay/channel/task/tencentvod/adaptor_test.go`

- [ ] **Step 1: 写测试，确认不消费文档未列出的字段**

```go
func TestConvertToRequestPayloadIgnoresUnsupportedMViduFields(t *testing.T) {
	adaptor := &TaskAdaptor{}
	req := &relaycommon.TaskSubmitReq{
		Model:  "vidu-q3",
		Prompt: "ignore unsupported fields",
		Metadata: map[string]interface{}{
			"style":              "anime",
			"movement_amplitude": "large",
			"watermark":          true,
			"wm_position":        "bottom-right",
			"callback_url":       "https://example.com/callback",
		},
	}
	info := &relaycommon.RelayInfo{
		ChannelMeta: &relaycommon.ChannelMeta{
			UpstreamModelName: "vidu-q3",
		},
	}

	payload, err := adaptor.convertToRequestPayload(req, info)
	require.NoError(t, err)
	require.NotNil(t, payload)
	assert.Empty(t, payload.ExtInfo)
	assert.Empty(t, payload.SceneType)
	assert.Empty(t, payload.Procedure)
}
```

- [ ] **Step 2: 运行针对性测试**

Run:

```bash
go test ./relay/channel/task/tencentvod -run TestConvertToRequestPayloadIgnoresUnsupportedMViduFields -count=1
```

Expected:

```text
ok
```

- [ ] **Step 3: 跑腾讯 VOD adaptor 全部测试**

Run:

```bash
go test ./relay/channel/task/tencentvod -count=1
```

Expected:

```text
ok
```

- [ ] **Step 4: 跑 `m_vidu` 和编译级回归**

Run:

```bash
go test ./middleware ./relay/channel/task/m_vidu ./relay/channel/task/vidu -count=1
go test ./middleware ./relay/channel/task/m_vidu ./relay/channel/task/tencentvod ./relay/channel/task/vidu ./router ./relay ./controller -run TestDoesNotExist -count=1
```

Expected:

```text
ok
```

- [ ] **Step 5: 提交**

```bash
git add relay/channel/task/tencentvod/adaptor.go relay/channel/task/tencentvod/adaptor_test.go
git commit -m "test: lock m-vidu tencent vod mapping behavior"
```

---

## Self-Review

### Spec coverage

- `text2video` 共通字段映射：Task 1
- `img2video` 图片映射：Task 2
- `start-end2video` 首尾帧映射：Task 2
- `reference2video` 非主体调用：Task 3
- `reference2video` 主体调用：Task 4
- 忽略未映射字段：Task 5
- 只在 `relay/channel/task/tencentvod/adaptor.go` 实现映射：Task 1-4

### Placeholder scan

- 无 `TODO/TBD/implement later`
- 每个测试步骤都提供了实际测试代码
- 每个命令都给了预期输出

### Type consistency

- 使用的核心类型与现有代码一致：
  - `relaycommon.TaskSubmitReq`
  - `relaycommon.RelayInfo`
  - `relaycommon.TaskRelayInfo`
  - `AigcVideoTaskInputFileInfo`
  - `AigcVideoOutputConfig`
- `TaskActionReferenceGenerate` / `TaskActionFirstTailGenerate` 与现有常量名保持一致

---

Plan complete and saved to `docs/vidu/2026-05-25-m-vidu-tencentvod-execution-plan.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
