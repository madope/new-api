package tencentvod

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/relay/channel"
	mvidu "github.com/QuantumNous/new-api/relay/channel/task/m_vidu"
	taskcommon "github.com/QuantumNous/new-api/relay/channel/task/taskcommon"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/pkg/errors"
)

const (
	defaultAPIVersion = "2018-07-17"
	defaultRegion     = "ap-guangzhou"
	defaultDuration   = 4
	defaultHeight     = 720
	serviceName       = "vod"

	BillingRatioMode       = "mode"
	BillingRatioDuration   = "duration"
	BillingRatioResolution = "resolution"
)

type BillingMode string

const (
	BillingModeTextToVideo         BillingMode = "text_to_video"
	BillingModeReferenceImage      BillingMode = "reference_image"
	BillingModeFirstLastFrame      BillingMode = "first_last_frame"
	BillingModeReferenceVideo      BillingMode = "reference_video"
	BillingModeVideoEdit           BillingMode = "video_edit"
	BillingModeSceneMotionControl  BillingMode = "scene_motion_control"
	BillingModeSceneAvatarI2V      BillingMode = "scene_avatar_i2v"
	BillingModeSceneLipSync        BillingMode = "scene_lip_sync"
	BillingModeSceneTemplateEffect BillingMode = "scene_template_effect"
)

var ModeRatioMap = map[BillingMode]float64{
	BillingModeTextToVideo:         1.0,
	BillingModeReferenceImage:      1.1,
	BillingModeFirstLastFrame:      1.2,
	BillingModeReferenceVideo:      1.3,
	BillingModeVideoEdit:           1.35,
	BillingModeSceneMotionControl:  1.25,
	BillingModeSceneAvatarI2V:      1.25,
	BillingModeSceneLipSync:        1.25,
	BillingModeSceneTemplateEffect: 1.25,
}

type KeyConfig struct {
	SecretID  string
	SecretKey string
	SubAppID  int64
}

type ModelSpec struct {
	Canonical    string
	ModelName    string
	ModelVersion string
}

var modelRegistry = map[string]ModelSpec{
	"kling-1.6":       {Canonical: "kling-1.6", ModelName: "Kling", ModelVersion: "1.6"},
	"kling-2.0":       {Canonical: "kling-2.0", ModelName: "Kling", ModelVersion: "2.0"},
	"kling-2.1":       {Canonical: "kling-2.1", ModelName: "Kling", ModelVersion: "2.1"},
	"kling-2.5":       {Canonical: "kling-2.5", ModelName: "Kling", ModelVersion: "2.5"},
	"kling-2.6":       {Canonical: "kling-2.6", ModelName: "Kling", ModelVersion: "2.6"},
	"kling-o1":        {Canonical: "kling-o1", ModelName: "Kling", ModelVersion: "O1"},
	"kling-3.0":       {Canonical: "kling-3.0", ModelName: "Kling", ModelVersion: "3.0"},
	"kling-3.0-omni":  {Canonical: "kling-3.0-omni", ModelName: "Kling", ModelVersion: "3.0-Omni"},
	"vidu-q2":         {Canonical: "vidu-q2", ModelName: "Vidu", ModelVersion: "q2"},
	"vidu-q2-pro":     {Canonical: "vidu-q2-pro", ModelName: "Vidu", ModelVersion: "q2-pro"},
	"vidu-q2-turbo":   {Canonical: "vidu-q2-turbo", ModelName: "Vidu", ModelVersion: "q2-turbo"},
	"vidu-q3":         {Canonical: "vidu-q3", ModelName: "Vidu", ModelVersion: "q3"},
	"vidu-q3-pro":     {Canonical: "vidu-q3-pro", ModelName: "Vidu", ModelVersion: "q3-pro"},
	"vidu-q3-turbo":   {Canonical: "vidu-q3-turbo", ModelName: "Vidu", ModelVersion: "q3-turbo"},
	"hailuo-02":       {Canonical: "hailuo-02", ModelName: "Hailuo", ModelVersion: "02"},
	"hailuo-2.3":      {Canonical: "hailuo-2.3", ModelName: "Hailuo", ModelVersion: "2.3"},
	"hailuo-2.3-fast": {Canonical: "hailuo-2.3-fast", ModelName: "Hailuo", ModelVersion: "2.3-fast"},
	"hunyuan-1.5":     {Canonical: "hunyuan-1.5", ModelName: "Hunyuan", ModelVersion: "1.5"},
	"mingmou-1.0":     {Canonical: "mingmou-1.0", ModelName: "Mingmou", ModelVersion: "1.0"},
	"gv-3.1":          {Canonical: "gv-3.1", ModelName: "GV", ModelVersion: "3.1"},
	"gv-3.1-fast":     {Canonical: "gv-3.1-fast", ModelName: "GV", ModelVersion: "3.1-fast"},
	"os-2.0":          {Canonical: "os-2.0", ModelName: "OS", ModelVersion: "2.0"},
	"pixverse-v5.6":   {Canonical: "pixverse-v5.6", ModelName: "PixVerse", ModelVersion: "v5.6"},
	"pixverse-v6":     {Canonical: "pixverse-v6", ModelName: "PixVerse", ModelVersion: "v6"},
	"pixverse-c1":     {Canonical: "pixverse-c1", ModelName: "PixVerse", ModelVersion: "c1"},
	"jimeng-3.0pro":   {Canonical: "jimeng-3.0pro", ModelName: "Jimeng", ModelVersion: "3.0pro"},
}

var providerNameMap = map[string]string{
	"kling":    "Kling",
	"vidu":     "Vidu",
	"hailuo":   "Hailuo",
	"hunyuan":  "Hunyuan",
	"mingmou":  "Mingmou",
	"gv":       "GV",
	"os":       "OS",
	"pixverse": "PixVerse",
	"jimeng":   "Jimeng",
}

type taskMeta struct {
	FileInfos       []AigcVideoTaskInputFileInfo    `json:"file_infos,omitempty"`
	SubjectInfos    []AigcVideoTaskInputSubjectInfo `json:"subject_infos,omitempty"`
	NegativePrompt  string                          `json:"negative_prompt,omitempty"`
	EnhancePrompt   string                          `json:"enhance_prompt,omitempty"`
	InputRegion     string                          `json:"input_region,omitempty"`
	SceneType       string                          `json:"scene_type,omitempty"`
	Procedure       string                          `json:"procedure,omitempty"`
	Seed            *int                            `json:"seed,omitempty"`
	SessionID       string                          `json:"session_id,omitempty"`
	SessionContext  string                          `json:"session_context,omitempty"`
	TasksPriority   *int                            `json:"tasks_priority,omitempty"`
	ExtInfo         string                          `json:"ext_info,omitempty"`
	LastFrameFileID string                          `json:"last_frame_file_id,omitempty"`
	LastFrameURL    string                          `json:"last_frame_url,omitempty"`
	GenerationMode  string                          `json:"generation_mode,omitempty"`
	OutputConfig    *AigcVideoOutputConfig          `json:"output_config,omitempty"`
}

type TaskAdaptor struct {
	taskcommon.BaseBilling
	ChannelType int
	baseURL     string
	keyConfig   KeyConfig
	region      string
}

func ParseKey(raw string) (KeyConfig, error) {
	parts := strings.Split(raw, "|")
	if len(parts) != 3 {
		return KeyConfig{}, fmt.Errorf("invalid tencent vod key format")
	}
	subAppID, err := strconv.ParseInt(strings.TrimSpace(parts[2]), 10, 64)
	if err != nil || subAppID <= 0 {
		return KeyConfig{}, fmt.Errorf("invalid tencent vod sub_app_id")
	}
	cfg := KeyConfig{
		SecretID:  strings.TrimSpace(parts[0]),
		SecretKey: strings.TrimSpace(parts[1]),
		SubAppID:  subAppID,
	}
	if cfg.SecretID == "" || cfg.SecretKey == "" {
		return KeyConfig{}, fmt.Errorf("invalid tencent vod key format")
	}
	return cfg, nil
}

func ResolveModel(name string) (ModelSpec, error) {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if normalized == "" {
		return ModelSpec{}, fmt.Errorf("model is required")
	}
	if spec, ok := modelRegistry[normalized]; ok {
		return spec, nil
	}
	parts := strings.SplitN(normalized, "-", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return ModelSpec{}, fmt.Errorf("unsupported vod model: %s", name)
	}
	provider, ok := providerNameMap[parts[0]]
	if !ok {
		return ModelSpec{}, fmt.Errorf("unsupported vod model: %s", name)
	}
	return ModelSpec{
		Canonical:    normalized,
		ModelName:    provider,
		ModelVersion: parts[1],
	}, nil
}

func (a *TaskAdaptor) Init(info *relaycommon.RelayInfo) {
	a.ChannelType = info.ChannelType
	a.baseURL = info.ChannelBaseUrl
	a.region = defaultRegion
	cfg, err := ParseKey(info.ApiKey)
	if err == nil {
		a.keyConfig = cfg
	}
}

func (a *TaskAdaptor) ValidateRequestAndSetAction(c *gin.Context, info *relaycommon.RelayInfo) (taskErr *dto.TaskError) {
	if info.TaskRelayInfo == nil {
		info.TaskRelayInfo = &relaycommon.TaskRelayInfo{}
	}
	var req relaycommon.TaskSubmitReq
	if err := common.UnmarshalBodyReusable(c, &req); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if strings.TrimSpace(req.Model) == "" {
		return service.TaskErrorWrapperLocal(fmt.Errorf("model field is required"), "missing_model", http.StatusBadRequest)
	}
	if len(req.Images) == 0 && strings.TrimSpace(req.Image) != "" {
		req.Images = []string{req.Image}
	}
	meta, err := parseTaskMeta(req.Metadata)
	if err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if err := validateTencentVODRequest(&req, meta); err != nil {
		return service.TaskErrorWrapperLocal(err, "invalid_request", http.StatusBadRequest)
	}
	if action := c.GetString("action"); action != "" {
		info.Action = action
	} else {
		info.Action = detectAction(&req, meta)
	}
	c.Set("task_request", req)
	return nil
}

func (a *TaskAdaptor) EstimateBilling(c *gin.Context, info *relaycommon.RelayInfo) map[string]float64 {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil
	}
	return a.estimateBillingFromRequest(&req, info)
}

func (a *TaskAdaptor) estimateBillingFromRequest(req *relaycommon.TaskSubmitReq, _ *relaycommon.RelayInfo) map[string]float64 {
	mode := DetectBillingMode(req)
	duration := resolveDuration(req)
	resolutionRatio := resolveResolutionRatio(req)
	return map[string]float64{
		BillingRatioMode:       ModeRatioMap[mode],
		BillingRatioDuration:   float64(duration) / float64(defaultDuration),
		BillingRatioResolution: resolutionRatio,
	}
}

func (a *TaskAdaptor) AdjustBillingOnComplete(task *model.Task, taskResult *relaycommon.TaskInfo) int {
	if task == nil || taskResult == nil || taskResult.Status != model.TaskStatusSuccess {
		return 0
	}
	var resp DescribeTaskDetailResponse
	if err := common.Unmarshal(task.Data, &resp); err != nil {
		return 0
	}
	if resp.Response.AigcVideoTask == nil {
		return 0
	}
	bc := task.PrivateData.BillingContext
	if bc == nil || len(bc.OtherRatios) == 0 {
		return 0
	}
	modeRatio := bc.OtherRatios[BillingRatioMode]
	oldDurationRatio := bc.OtherRatios[BillingRatioDuration]
	oldResolutionRatio := bc.OtherRatios[BillingRatioResolution]
	if modeRatio <= 0 || oldDurationRatio <= 0 || oldResolutionRatio <= 0 {
		return 0
	}
	baseQuota := float64(task.Quota) / (modeRatio * oldDurationRatio * oldResolutionRatio)
	if baseQuota <= 0 {
		return 0
	}
	durationRatio := float64(resolveFinalDuration(resp.Response.AigcVideoTask)) / float64(defaultDuration)
	if durationRatio <= 0 {
		durationRatio = oldDurationRatio
	}
	resolutionRatio := resolveFinalResolutionRatio(resp.Response.AigcVideoTask)
	if resolutionRatio <= 0 {
		resolutionRatio = oldResolutionRatio
	}
	actual := int(math.Round(baseQuota * modeRatio * durationRatio * resolutionRatio))
	if actual <= 0 {
		return 0
	}
	return actual
}

func (a *TaskAdaptor) BuildRequestURL(info *relaycommon.RelayInfo) (string, error) {
	return fmt.Sprintf("%s/", strings.TrimRight(info.ChannelBaseUrl, "/")), nil
}

func (a *TaskAdaptor) BuildRequestHeader(_ *gin.Context, req *http.Request, _ *relaycommon.RelayInfo) error {
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	timestamp := time.Now().Unix()
	auth := buildAuthorization(a.keyConfig.SecretID, a.keyConfig.SecretKey, "CreateAigcVideoTask", a.region, timestamp, bodyBytes)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Host", "vod.tencentcloudapi.com")
	req.Header.Set("X-TC-Action", "CreateAigcVideoTask")
	req.Header.Set("X-TC-Version", defaultAPIVersion)
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Authorization", auth)
	if a.region != "" {
		req.Header.Set("X-TC-Region", a.region)
	}
	return nil
}

func (a *TaskAdaptor) BuildRequestBody(c *gin.Context, info *relaycommon.RelayInfo) (io.Reader, error) {
	req, err := relaycommon.GetTaskRequest(c)
	if err != nil {
		return nil, err
	}
	payload, err := a.convertToRequestPayload(&req, info)
	if err != nil {
		return nil, err
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(data), nil
}

func (a *TaskAdaptor) DoRequest(c *gin.Context, info *relaycommon.RelayInfo, requestBody io.Reader) (*http.Response, error) {
	return channel.DoTaskApiRequest(a, c, info, requestBody)
}

func (a *TaskAdaptor) DoResponse(c *gin.Context, resp *http.Response, info *relaycommon.RelayInfo) (taskID string, taskData []byte, taskErr *dto.TaskError) {
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, service.TaskErrorWrapper(err, "read_response_body_failed", http.StatusInternalServerError)
	}
	_ = resp.Body.Close()
	var submitResp CreateAigcVideoTaskResponse
	if err := common.Unmarshal(responseBody, &submitResp); err != nil {
		return "", nil, service.TaskErrorWrapper(errors.Wrapf(err, "body: %s", responseBody), "unmarshal_response_body_failed", http.StatusInternalServerError)
	}
	if submitResp.Response.TaskID == "" {
		return "", nil, service.TaskErrorWrapper(fmt.Errorf("task_id is empty"), "invalid_response", http.StatusInternalServerError)
	}
	if common.GetContextKeyBool(c, constant.ContextKeyMViduCompat) {
		compatResp, err := mvidu.BuildSubmitResponse(info.PublicTaskID)
		if err != nil {
			return "", nil, service.TaskErrorWrapper(err, "marshal_response_body_failed", http.StatusInternalServerError)
		}
		c.Data(http.StatusOK, "application/json", compatResp)
		return submitResp.Response.TaskID, responseBody, nil
	}
	ov := dto.NewOpenAIVideo()
	ov.ID = info.PublicTaskID
	ov.TaskID = info.PublicTaskID
	ov.CreatedAt = time.Now().Unix()
	ov.Model = info.OriginModelName
	c.JSON(http.StatusOK, ov)
	return submitResp.Response.TaskID, responseBody, nil
}

func (a *TaskAdaptor) FetchTask(baseURL, key string, body map[string]any, proxy string) (*http.Response, error) {
	cfg, err := ParseKey(key)
	if err != nil {
		return nil, err
	}
	taskID, _ := body["task_id"].(string)
	payload := DescribeTaskDetailRequest{
		TaskID:   taskID,
		SubAppID: cfg.SubAppID,
	}
	data, err := common.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/", strings.TrimRight(baseURL, "/")), bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Host", "vod.tencentcloudapi.com")
	req.Header.Set("X-TC-Action", "DescribeTaskDetail")
	req.Header.Set("X-TC-Version", defaultAPIVersion)
	timestamp := time.Now().Unix()
	req.Header.Set("X-TC-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("Authorization", buildAuthorization(cfg.SecretID, cfg.SecretKey, "DescribeTaskDetail", defaultRegion, timestamp, data))
	if defaultRegion != "" {
		req.Header.Set("X-TC-Region", defaultRegion)
	}
	client, err := service.GetHttpClientWithProxy(proxy)
	if err != nil {
		return nil, fmt.Errorf("new proxy http client failed: %w", err)
	}
	return client.Do(req)
}

func (a *TaskAdaptor) ParseTaskResult(respBody []byte) (*relaycommon.TaskInfo, error) {
	var resp DescribeTaskDetailResponse
	if err := common.Unmarshal(respBody, &resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal task result failed")
	}
	if resp.Response.AigcVideoTask == nil {
		return nil, fmt.Errorf("aigc_video_task is nil")
	}
	task := resp.Response.AigcVideoTask
	result := &relaycommon.TaskInfo{
		TaskID: task.TaskID,
	}
	status := strings.ToUpper(strings.TrimSpace(task.Status))
	switch status {
	case "WAITING", "PENDING", "SUBMIT", "SUBMITTED":
		result.Status = model.TaskStatusSubmitted
		result.Progress = "10%"
	case "PROCESSING", "RUNNING":
		result.Status = model.TaskStatusInProgress
		if task.Progress > 0 {
			result.Progress = fmt.Sprintf("%d%%", task.Progress)
		} else {
			result.Progress = "50%"
		}
	case "FINISH", "SUCCESS", "SUCCEED", "SUCCEEDED":
		if task.ErrCode != 0 {
			result.Status = model.TaskStatusFailure
			result.Progress = "100%"
			result.Reason = task.Message
			break
		}
		result.Status = model.TaskStatusSuccess
		result.Progress = "100%"
		result.Url = pickPrimaryVideoURL(task.Output.FileInfos)
	case "FAIL", "FAILED", "ERROR":
		result.Status = model.TaskStatusFailure
		result.Progress = "100%"
		result.Reason = task.Message
	default:
		if strings.EqualFold(resp.Response.Status, "FINISH") && task.ErrCode == 0 {
			result.Status = model.TaskStatusSuccess
			result.Progress = "100%"
			result.Url = pickPrimaryVideoURL(task.Output.FileInfos)
		} else {
			result.Status = model.TaskStatusInProgress
			result.Progress = "30%"
		}
	}
	return result, nil
}

func (a *TaskAdaptor) GetModelList() []string {
	out := make([]string, 0, len(modelRegistry))
	for _, spec := range modelRegistry {
		out = append(out, spec.Canonical)
	}
	return out
}

func (a *TaskAdaptor) GetChannelName() string {
	return "tencent_vod_video"
}

func (a *TaskAdaptor) ConvertToOpenAIVideo(task *model.Task) ([]byte, error) {
	var resp DescribeTaskDetailResponse
	if err := common.Unmarshal(task.Data, &resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal tencent vod task data failed")
	}
	video := task.ToOpenAIVideo()
	video.Model = task.Properties.OriginModelName
	if resp.Response.AigcVideoTask != nil {
		video.SetMetadata("output", resp.Response.AigcVideoTask.Output)
		video.SetMetadata("input", resp.Response.AigcVideoTask.Input)
		video.SetMetadata("request_id", resp.Response.RequestID)
		video.CreatedAt = parseTimeUnix(resp.Response.CreateTime, task.CreatedAt)
		video.CompletedAt = parseTimeUnix(resp.Response.FinishTime, task.UpdatedAt)
		if task.Status == model.TaskStatusFailure {
			video.Error = &dto.OpenAIVideoError{
				Message: resp.Response.AigcVideoTask.Message,
				Code:    strconv.Itoa(resp.Response.AigcVideoTask.ErrCode),
			}
		}
	}
	return common.Marshal(video)
}

func (a *TaskAdaptor) convertToRequestPayload(req *relaycommon.TaskSubmitReq, info *relaycommon.RelayInfo) (*CreateAigcVideoTaskRequest, error) {
	spec, err := ResolveModel(info.UpstreamModelName)
	if err != nil {
		return nil, err
	}
	meta, err := parseTaskMeta(req.Metadata)
	if err != nil {
		return nil, err
	}
	fileInfos := cloneFileInfos(meta.FileInfos)
	if len(fileInfos) == 0 && len(req.Images) > 0 {
		usage := "Reference"
		if meta.LastFrameFileID != "" || meta.LastFrameURL != "" {
			usage = "FirstFrame"
		}
		for _, image := range req.Images {
			if strings.TrimSpace(image) == "" {
				continue
			}
			fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
				Type:     "Url",
				Category: "Image",
				URL:      image,
				Usage:    usage,
			})
		}
	}
	if len(fileInfos) == 0 && strings.TrimSpace(req.InputReference) != "" {
		fileInfos = append(fileInfos, AigcVideoTaskInputFileInfo{
			Type:     "Url",
			Category: "Image",
			URL:      req.InputReference,
			Usage:    "Reference",
		})
	}
	payload := &CreateAigcVideoTaskRequest{
		SubAppID:        a.keyConfig.SubAppID,
		ModelName:       spec.ModelName,
		ModelVersion:    spec.ModelVersion,
		FileInfos:       fileInfos,
		SubjectInfos:    meta.SubjectInfos,
		LastFrameFileID: meta.LastFrameFileID,
		LastFrameURL:    meta.LastFrameURL,
		Prompt:          req.Prompt,
		NegativePrompt:  meta.NegativePrompt,
		EnhancePrompt:   meta.EnhancePrompt,
		OutputConfig:    meta.OutputConfig,
		InputRegion:     strings.TrimSpace(meta.InputRegion),
		SceneType:       meta.SceneType,
		Procedure:       meta.Procedure,
		SessionID:       meta.SessionID,
		SessionContext:  meta.SessionContext,
		ExtInfo:         meta.ExtInfo,
		GenerationMode:  meta.GenerationMode,
	}
	if meta.Seed != nil {
		payload.Seed = *meta.Seed
	}
	if meta.TasksPriority != nil {
		payload.TasksPriority = *meta.TasksPriority
	}
	if payload.OutputConfig == nil {
		payload.OutputConfig = &AigcVideoOutputConfig{}
	}
	if payload.OutputConfig.Duration == 0 {
		payload.OutputConfig.Duration = resolveDuration(req)
	}
	if payload.OutputConfig.Resolution == "" && req.Size != "" {
		payload.OutputConfig.Resolution = normalizeResolution(req.Size)
	}
	return payload, nil
}

func parseTaskMeta(metadata map[string]interface{}) (*taskMeta, error) {
	meta := &taskMeta{}
	if metadata == nil {
		return meta, nil
	}
	data, err := common.Marshal(metadata)
	if err != nil {
		return nil, err
	}
	if err := common.Unmarshal(data, meta); err != nil {
		return nil, err
	}
	if raw, ok := metadata["file_infos"]; ok {
		files, err := parseFileInfos(raw)
		if err != nil {
			return nil, err
		}
		meta.FileInfos = files
	}
	if raw, ok := metadata["subject_infos"]; ok {
		subjects, err := parseSubjectInfos(raw)
		if err != nil {
			return nil, err
		}
		meta.SubjectInfos = subjects
	}
	return meta, nil
}

func validateTencentVODRequest(req *relaycommon.TaskSubmitReq, meta *taskMeta) error {
	hasReference := len(req.Images) > 0 || len(meta.FileInfos) > 0 || meta.LastFrameFileID != "" || meta.LastFrameURL != "" || strings.TrimSpace(req.InputReference) != ""
	if strings.TrimSpace(req.Prompt) == "" && strings.TrimSpace(meta.SceneType) == "" && !hasReference {
		return fmt.Errorf("prompt is required")
	}
	if (meta.LastFrameFileID != "" || meta.LastFrameURL != "") && len(req.Images) == 0 && !hasFirstFrame(meta.FileInfos) {
		return fmt.Errorf("last frame requires first frame")
	}
	return nil
}

func detectAction(req *relaycommon.TaskSubmitReq, meta *taskMeta) string {
	switch meta.SceneType {
	case "motion_control":
		return "sceneMotionControlGenerate"
	case "avatar_i2v":
		return "sceneAvatarI2VGenerate"
	case "lip_sync":
		return "sceneLipSyncGenerate"
	case "template_effect":
		return "sceneTemplateEffectGenerate"
	}
	if containsVideo(meta.FileInfos) {
		if containsVideoEdit(meta.FileInfos) {
			return "videoEditGenerate"
		}
		return "videoReferenceGenerate"
	}
	if meta.LastFrameFileID != "" || meta.LastFrameURL != "" || hasFirstFrame(meta.FileInfos) {
		return constant.TaskActionFirstTailGenerate
	}
	if len(req.Images) > 0 || strings.TrimSpace(req.InputReference) != "" || len(meta.FileInfos) > 0 {
		return constant.TaskActionReferenceGenerate
	}
	return constant.TaskActionTextGenerate
}

func DetectBillingMode(req *relaycommon.TaskSubmitReq) BillingMode {
	meta, _ := parseTaskMeta(req.Metadata)
	if meta != nil {
		switch meta.SceneType {
		case "motion_control":
			return BillingModeSceneMotionControl
		case "avatar_i2v":
			return BillingModeSceneAvatarI2V
		case "lip_sync":
			return BillingModeSceneLipSync
		case "template_effect":
			return BillingModeSceneTemplateEffect
		}
		if containsVideo(meta.FileInfos) {
			if containsVideoEdit(meta.FileInfos) {
				return BillingModeVideoEdit
			}
			return BillingModeReferenceVideo
		}
		if meta.LastFrameFileID != "" || meta.LastFrameURL != "" || hasFirstFrame(meta.FileInfos) {
			return BillingModeFirstLastFrame
		}
	}
	if len(req.Images) > 0 || strings.TrimSpace(req.InputReference) != "" {
		if meta != nil && (meta.LastFrameFileID != "" || meta.LastFrameURL != "") {
			return BillingModeFirstLastFrame
		}
		return BillingModeReferenceImage
	}
	return BillingModeTextToVideo
}

func resolveDuration(req *relaycommon.TaskSubmitReq) int {
	meta, _ := parseTaskMeta(req.Metadata)
	if meta != nil && meta.OutputConfig != nil && meta.OutputConfig.Duration > 0 {
		return meta.OutputConfig.Duration
	}
	if req.Duration > 0 {
		return req.Duration
	}
	if sec, err := strconv.Atoi(req.Seconds); err == nil && sec > 0 {
		return sec
	}
	return defaultDuration
}

func resolveResolutionRatio(req *relaycommon.TaskSubmitReq) float64 {
	height := defaultHeight
	meta, _ := parseTaskMeta(req.Metadata)
	if meta != nil && meta.OutputConfig != nil && meta.OutputConfig.Resolution != "" {
		if h := parseResolutionHeight(meta.OutputConfig.Resolution); h > 0 {
			height = h
		}
	} else if req.Size != "" {
		if h := parseSizeHeight(req.Size); h > 0 {
			height = h
		}
	}
	return float64(height) / float64(defaultHeight)
}

func resolveFinalDuration(task *AigcVideoTask) int {
	if task == nil {
		return 0
	}
	if task.Input.OutputConfig.Duration > 0 {
		return task.Input.OutputConfig.Duration
	}
	if len(task.Output.FileInfos) > 0 {
		dur := task.Output.FileInfos[0].MetaData.VideoDuration
		if dur <= 0 {
			dur = task.Output.FileInfos[0].MetaData.Duration
		}
		if dur > 0 {
			return int(math.Round(dur))
		}
	}
	return 0
}

func resolveFinalResolutionRatio(task *AigcVideoTask) float64 {
	if task == nil {
		return 0
	}
	if len(task.Output.FileInfos) > 0 {
		meta := task.Output.FileInfos[0].MetaData
		if meta.Height > 0 {
			return float64(meta.Height) / float64(defaultHeight)
		}
	}
	if task.Input.OutputConfig.Resolution != "" {
		if h := parseResolutionHeight(task.Input.OutputConfig.Resolution); h > 0 {
			return float64(h) / float64(defaultHeight)
		}
	}
	return 0
}

func pickPrimaryVideoURL(files []AigcVideoOutputFileInfo) string {
	for _, file := range files {
		if strings.Contains(strings.ToLower(file.FileType), "mp4") && file.FileURL != "" {
			return file.FileURL
		}
	}
	for _, file := range files {
		if file.FileURL != "" {
			return file.FileURL
		}
	}
	return ""
}

func buildAuthorization(secretID, secretKey, action, region string, timestamp int64, payload []byte) string {
	host := "vod.tencentcloudapi.com"
	date := time.Unix(timestamp, 0).UTC().Format("2006-01-02")
	canonicalHeaders := fmt.Sprintf("content-type:%s\nhost:%s\nx-tc-action:%s\n",
		"application/json; charset=utf-8", host, strings.ToLower(action))
	signedHeaders := "content-type;host;x-tc-action"
	hashedRequestPayload := sha256hex(string(payload))
	canonicalRequest := fmt.Sprintf("%s\n%s\n%s\n%s\n%s\n%s",
		http.MethodPost, "/", "", canonicalHeaders, signedHeaders, hashedRequestPayload)
	credentialScope := fmt.Sprintf("%s/%s/tc3_request", date, serviceName)
	stringToSign := fmt.Sprintf("TC3-HMAC-SHA256\n%d\n%s\n%s", timestamp, credentialScope, sha256hex(canonicalRequest))
	secretDate := hmacsha256(date, "TC3"+secretKey)
	secretService := hmacsha256(serviceName, secretDate)
	secretSigning := hmacsha256("tc3_request", secretService)
	signature := hex.EncodeToString([]byte(hmacsha256(stringToSign, secretSigning)))
	return fmt.Sprintf("TC3-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		secretID, credentialScope, signedHeaders, signature)
}

func sha256hex(s string) string {
	b := sha256.Sum256([]byte(s))
	return hex.EncodeToString(b[:])
}

func hmacsha256(s, key string) string {
	hashed := hmac.New(sha256.New, []byte(key))
	hashed.Write([]byte(s))
	return string(hashed.Sum(nil))
}

func containsVideo(files []AigcVideoTaskInputFileInfo) bool {
	for _, file := range files {
		if strings.EqualFold(file.Category, "Video") {
			return true
		}
	}
	return false
}

func containsVideoEdit(files []AigcVideoTaskInputFileInfo) bool {
	for _, file := range files {
		if strings.EqualFold(file.Category, "Video") && strings.EqualFold(file.ReferenceType, "Edit") {
			return true
		}
	}
	return false
}

func hasFirstFrame(files []AigcVideoTaskInputFileInfo) bool {
	for _, file := range files {
		if strings.EqualFold(file.Usage, "FirstFrame") {
			return true
		}
	}
	return false
}

func cloneFileInfos(files []AigcVideoTaskInputFileInfo) []AigcVideoTaskInputFileInfo {
	if len(files) == 0 {
		return nil
	}
	out := make([]AigcVideoTaskInputFileInfo, len(files))
	copy(out, files)
	return out
}

func parseFileInfos(raw any) ([]AigcVideoTaskInputFileInfo, error) {
	data, err := common.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	if err := common.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	out := make([]AigcVideoTaskInputFileInfo, 0, len(items))
	for _, item := range items {
		out = append(out, AigcVideoTaskInputFileInfo{
			Type:              stringValue(item, "Type", "type"),
			Category:          stringValue(item, "Category", "category"),
			URL:               stringValue(item, "Url", "url"),
			FileID:            stringValue(item, "FileId", "file_id"),
			Usage:             stringValue(item, "Usage", "usage"),
			ObjectID:          stringValue(item, "ObjectId", "object_id"),
			ReferenceType:     stringValue(item, "ReferenceType", "reference_type"),
			Text:              stringValue(item, "Text", "text"),
			VoiceID:           stringValue(item, "VoiceId", "voice_id"),
			KeepOriginalSound: stringValue(item, "KeepOriginalSound", "keep_original_sound"),
		})
	}
	return out, nil
}

func parseSubjectInfos(raw any) ([]AigcVideoTaskInputSubjectInfo, error) {
	data, err := common.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var items []map[string]any
	if err := common.Unmarshal(data, &items); err != nil {
		return nil, err
	}
	out := make([]AigcVideoTaskInputSubjectInfo, 0, len(items))
	for _, item := range items {
		out = append(out, AigcVideoTaskInputSubjectInfo{
			ObjectID: stringValue(item, "ObjectId", "object_id"),
			Name:     stringValue(item, "Name", "name"),
		})
	}
	return out, nil
}

func stringValue(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			if s, ok := value.(string); ok {
				return s
			}
		}
	}
	return ""
}

func parseResolutionHeight(resolution string) int {
	s := strings.ToLower(strings.TrimSpace(resolution))
	s = strings.TrimSuffix(s, "p")
	if h, err := strconv.Atoi(s); err == nil && h > 0 {
		return h
	}
	return 0
}

func parseSizeHeight(size string) int {
	parts := strings.Split(strings.ToLower(size), "x")
	if len(parts) != 2 {
		return 0
	}
	h, _ := strconv.Atoi(parts[1])
	return h
}

func normalizeResolution(size string) string {
	if h := parseSizeHeight(size); h > 0 {
		return fmt.Sprintf("%dp", h)
	}
	return ""
}

func parseTimeUnix(s string, fallback int64) int64 {
	if s == "" {
		return fallback
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return fallback
	}
	return t.Unix()
}
