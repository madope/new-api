package tencentvod

type CreateAigcVideoTaskRequest struct {
	SubAppID        int64                           `json:"SubAppId"`
	ModelName       string                          `json:"ModelName"`
	ModelVersion    string                          `json:"ModelVersion"`
	FileInfos       []AigcVideoTaskInputFileInfo    `json:"FileInfos,omitempty"`
	SubjectInfos    []AigcVideoTaskInputSubjectInfo `json:"SubjectInfos,omitempty"`
	LastFrameFileID string                          `json:"LastFrameFileId,omitempty"`
	LastFrameURL    string                          `json:"LastFrameUrl,omitempty"`
	Prompt          string                          `json:"Prompt,omitempty"`
	NegativePrompt  string                          `json:"NegativePrompt,omitempty"`
	EnhancePrompt   string                          `json:"EnhancePrompt,omitempty"`
	OutputConfig    *AigcVideoOutputConfig          `json:"OutputConfig,omitempty"`
	InputRegion     string                          `json:"InputRegion,omitempty"`
	SceneType       string                          `json:"SceneType,omitempty"`
	Procedure       string                          `json:"Procedure,omitempty"`
	Seed            int                             `json:"Seed,omitempty"`
	SessionID       string                          `json:"SessionId,omitempty"`
	SessionContext  string                          `json:"SessionContext,omitempty"`
	TasksPriority   int                             `json:"TasksPriority,omitempty"`
	ExtInfo         string                          `json:"ExtInfo,omitempty"`
	GenerationMode  string                          `json:"GenerationMode,omitempty"`
}

type CreateAigcVideoTaskResponse struct {
	Response struct {
		TaskID    string `json:"TaskId"`
		RequestID string `json:"RequestId"`
	} `json:"Response"`
}

type DescribeTaskDetailRequest struct {
	TaskID   string `json:"TaskId"`
	SubAppID int64  `json:"SubAppId"`
}

type DescribeTaskDetailResponse struct {
	Response DescribeTaskDetailResponseBody `json:"Response"`
}

type DescribeTaskDetailResponseBody struct {
	AigcVideoTask    *AigcVideoTask `json:"AigcVideoTask"`
	Status           string         `json:"Status"`
	TaskType         string         `json:"TaskType"`
	CreateTime       string         `json:"CreateTime"`
	BeginProcessTime string         `json:"BeginProcessTime"`
	FinishTime       string         `json:"FinishTime"`
	RequestID        string         `json:"RequestId"`
}

type AigcVideoTask struct {
	ErrCode        int                 `json:"ErrCode"`
	ErrCodeExt     string              `json:"ErrCodeExt"`
	Input          AigcVideoTaskInput  `json:"Input"`
	Message        string              `json:"Message"`
	Output         AigcVideoTaskOutput `json:"Output"`
	Progress       int                 `json:"Progress"`
	SessionContext string              `json:"SessionContext"`
	SessionID      string              `json:"SessionId"`
	Status         string              `json:"Status"`
	TaskID         string              `json:"TaskId"`
}

type AigcVideoTaskInput struct {
	EnhancePrompt   string                          `json:"EnhancePrompt,omitempty"`
	FileInfos       []AigcVideoTaskInputFileInfo    `json:"FileInfos,omitempty"`
	GenerationMode  string                          `json:"GenerationMode,omitempty"`
	InputRegion     string                          `json:"InputRegion,omitempty"`
	LastFrameFileID string                          `json:"LastFrameFileId,omitempty"`
	LastFrameURL    string                          `json:"LastFrameUrl,omitempty"`
	ModelName       string                          `json:"ModelName,omitempty"`
	ModelVersion    string                          `json:"ModelVersion,omitempty"`
	NegativePrompt  string                          `json:"NegativePrompt,omitempty"`
	OutputConfig    AigcVideoOutputConfig           `json:"OutputConfig,omitempty"`
	Prompt          string                          `json:"Prompt,omitempty"`
	SceneType       string                          `json:"SceneType,omitempty"`
	SubjectInfos    []AigcVideoTaskInputSubjectInfo `json:"SubjectInfos,omitempty"`
}

type AigcVideoTaskInputFileInfo struct {
	Type              string `json:"Type,omitempty"`
	Category          string `json:"Category,omitempty"`
	URL               string `json:"Url,omitempty"`
	FileID            string `json:"FileId,omitempty"`
	Usage             string `json:"Usage,omitempty"`
	ObjectID          string `json:"ObjectId,omitempty"`
	ReferenceType     string `json:"ReferenceType,omitempty"`
	Text              string `json:"Text,omitempty"`
	VoiceID           string `json:"VoiceId,omitempty"`
	KeepOriginalSound string `json:"KeepOriginalSound,omitempty"`
}

type AigcVideoTaskInputSubjectInfo struct {
	ObjectID string `json:"ObjectId,omitempty"`
	Name     string `json:"Name,omitempty"`
}

type AigcVideoOutputConfig struct {
	StorageMode           string `json:"StorageMode,omitempty"`
	AspectRatio           string `json:"AspectRatio,omitempty"`
	AudioGeneration       string `json:"AudioGeneration,omitempty"`
	PersonGeneration      string `json:"PersonGeneration,omitempty"`
	InputComplianceCheck  string `json:"InputComplianceCheck,omitempty"`
	OutputComplianceCheck string `json:"OutputComplianceCheck,omitempty"`
	Duration              int    `json:"Duration,omitempty"`
	Resolution            string `json:"Resolution,omitempty"`
	EnableBGM             string `json:"EnableBGM,omitempty"`
	EnhanceSwitch         string `json:"EnhanceSwitch,omitempty"`
	FrameInterpolate      string `json:"FrameInterpolate,omitempty"`
	LogoAdd               string `json:"LogoAdd,omitempty"`
	MediaName             string `json:"MediaName,omitempty"`
	OffPeak               string `json:"OffPeak,omitempty"`
	ClassID               int    `json:"ClassId,omitempty"`
	ExpireTime            string `json:"ExpireTime,omitempty"`
}

type AigcVideoTaskOutput struct {
	FileInfos        []AigcVideoOutputFileInfo `json:"FileInfos,omitempty"`
	ProcedureTaskIDs []string                  `json:"ProcedureTaskIds,omitempty"`
}

type AigcVideoOutputFileInfo struct {
	ClassID     int                     `json:"ClassId,omitempty"`
	ExpireTime  string                  `json:"ExpireTime,omitempty"`
	FileContent string                  `json:"FileContent,omitempty"`
	FileID      string                  `json:"FileId,omitempty"`
	FileType    string                  `json:"FileType,omitempty"`
	FileURL     string                  `json:"FileUrl,omitempty"`
	MediaName   string                  `json:"MediaName,omitempty"`
	MetaData    AigcVideoOutputMetaData `json:"MetaData,omitempty"`
	StorageMode string                  `json:"StorageMode,omitempty"`
	UsageType   string                  `json:"UsageType,omitempty"`
}

type AigcVideoOutputMetaData struct {
	Duration      float64 `json:"Duration,omitempty"`
	VideoDuration float64 `json:"VideoDuration,omitempty"`
	Height        int     `json:"Height,omitempty"`
	Width         int     `json:"Width,omitempty"`
}
