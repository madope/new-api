# `/m-vidu` 到 `TencentVODVideo` 参数映射清单

## 目的

这份文档只描述一件事：

- 当用户侧调用 `/m-vidu/...` 接口
- 实际命中的上游渠道是 `TencentVODVideo`
- 哪些 Vidu 官方参数可以映射
- 如何映射
- 哪些参数当前不能映射，应忽略

约束：

1. `m-vidu` compat 层仍保持轻量
2. 不在 compat 层派生 `file_infos`、`last_frame_url` 等字段
3. Vidu 原始特殊字段继续原样保留到 `metadata`
4. 映射逻辑全部落在 `relay/channel/task/tencentvod/adaptor.go`

## 总体规则

### 统一规则

- 只映射 **腾讯 VOD 官方支持** 且 **存在稳定对应关系** 的字段
- 无稳定对应关系的 Vidu 字段，直接忽略
- 忽略不代表删除，原始字段仍保留在 `metadata`
- 如果字段已映射，但当前腾讯 VOD 具体模型版本不支持，则由上游腾讯 VOD 报错

### 共通字段映射

这几项对 4 个接口都适用：

| Vidu 字段 | Tencent VOD 字段 | 说明 |
|---|---|---|
| `model` | `ModelName` + `ModelVersion` | 继续走现有模型解析和 `model_mapping` |
| `prompt` | `Prompt` | 直接映射 |
| `duration` | `OutputConfig.Duration` | 直接映射 |
| `seed` | `Seed` | 直接映射 |
| `resolution` | `OutputConfig.Resolution` | 直接映射 |
| `aspect_ratio` | `OutputConfig.AspectRatio` | 直接映射 |
| `payload` | `SessionContext` | 作为透传上下文 |

### 共通输出/音频字段映射

| Vidu 字段 | Tencent VOD 字段 | 映射规则 |
|---|---|---|
| `audio` | `OutputConfig.AudioGeneration` | `true -> Enabled`，`false -> Disabled` |
| `bgm` | `OutputConfig.EnableBGM` | `true -> Enabled`，`false -> Disabled` |
| `off_peak` | `OutputConfig.OffPeak` | `true -> Enabled`，`false -> Disabled` |

## 接口 1：`POST /m-vidu/ent/v2/text2video`

### 可映射字段

| Vidu 字段 | Tencent VOD 字段 | 说明 |
|---|---|---|
| `model` | `ModelName` + `ModelVersion` | 统一模型解析 |
| `prompt` | `Prompt` | 直接映射 |
| `duration` | `OutputConfig.Duration` | 直接映射 |
| `seed` | `Seed` | 直接映射 |
| `resolution` | `OutputConfig.Resolution` | 直接映射 |
| `aspect_ratio` | `OutputConfig.AspectRatio` | 直接映射 |
| `payload` | `SessionContext` | 透传上下文 |
| `audio` | `OutputConfig.AudioGeneration` | 仅在腾讯 VOD 对应模型支持时生效 |
| `bgm` | `OutputConfig.EnableBGM` | 仅在腾讯 VOD 对应模型支持时生效 |
| `off_peak` | `OutputConfig.OffPeak` | 仅在腾讯 VOD 对应模型支持时生效 |

### 不能映射、应忽略的字段

| Vidu 字段 | 原因 |
|---|---|
| `style` | 腾讯 VOD 无稳定对应字段 |
| `callback_url` | `new-api` 任务链路不依赖上游回调 |
| `watermark` | 腾讯 VOD 无直接对应的统一字段 |
| `wm_position` | 腾讯 VOD 无直接对应的统一字段 |
| `movement_amplitude` | 腾讯 VOD 无稳定对应字段 |
| `is_rec` | 腾讯 VOD 无推荐提示词对应机制 |
| `audio_type` | 腾讯 VOD 无稳定的一一对应字段 |

## 接口 2：`POST /m-vidu/ent/v2/img2video`

### 可映射字段

除共通字段外，增加：

| Vidu 字段 | Tencent VOD 字段 | 说明 |
|---|---|---|
| `images[0]` | `FileInfos[]` | 映射为一张图片输入 |

映射建议：

- `Type=Url`
- `Category=Image`
- `Url=<images[0]>`
- `Usage=Reference`

### 不能映射、应忽略的字段

| Vidu 字段 | 原因 |
|---|---|
| `voice_id` | 图生视频顶层 `voice_id` 在腾讯 VOD 无稳定统一字段 |
| `callback_url` | 不依赖上游回调 |
| `watermark` | 无统一稳定对应 |
| `wm_position` | 无统一稳定对应 |
| `is_rec` | 无推荐提示词对应机制 |
| `movement_amplitude` | 无稳定对应 |
| `style` | 无稳定对应 |

说明：

- `audio_type` 当前也不映射
- 其余未知字段继续保留在 `metadata`

## 接口 3：`POST /m-vidu/ent/v2/start-end2video`

### 可映射字段

除共通字段外，增加：

| Vidu 字段 | Tencent VOD 字段 | 说明 |
|---|---|---|
| `images[0]` | `FileInfos[]` | 作为首帧输入 |
| `images[1]` | `LastFrameUrl` | 作为尾帧输入 |

映射建议：

- 首帧：
  - `Type=Url`
  - `Category=Image`
  - `Url=<images[0]>`
  - `Usage=FirstFrame`
- 尾帧：
  - `LastFrameUrl=<images[1]>`

### 不能映射、应忽略的字段

| Vidu 字段 | 原因 |
|---|---|
| `callback_url` | 不依赖上游回调 |
| `watermark` | 无统一稳定对应 |
| `wm_position` | 无统一稳定对应 |
| `is_rec` | 无推荐提示词对应机制 |
| `movement_amplitude` | 无稳定对应 |

说明：

- `audio`、`bgm`、`off_peak` 继续走共通映射
- `audio_type` 当前仍不映射

## 接口 4：`POST /m-vidu/ent/v2/reference2video`

### 4.1 主体调用

这里的“主体调用”指 Vidu 官方 `reference2video` 请求中使用 `subjects` 数组描述主体信息的调用方式。

Vidu 官方语义：

- `subjects` 是主体输入集合
- `subjects[].name` 是主体名
- `subjects[].images` / `subjects[].videos` 是主体素材
- `subjects[].server_id` 是已存在主体的服务端标识
- `subjects[].voice_id` 是主体音色标识

Tencent VOD 官方可承接的对应结构：

- `FileInfos[]`
  - `Category`
  - `Type`
  - `Url`
  - `ObjectId`
  - `VoiceId`
  - `Text`
  - `Usage`

#### 可映射字段

##### `subjects` 到 `FileInfos`

| Vidu 字段 | Tencent VOD 字段 | 说明 |
|---|---|---|
| `subjects[].name` | `FileInfos[].Text` | 作为主体名称语义透传到文件项 |
| `subjects[].server_id` | `FileInfos[].ObjectId` | 作为主体对象标识透传到文件项；没有时可回退到 `name` |
| `subjects[].images[]` | `FileInfos[]` | 每张图生成一项图片主体资源 |
| `subjects[].videos[]` | `FileInfos[]` | 每个视频生成一项视频主体资源 |
| `subjects[].voice_id` | `FileInfos[].VoiceId` | 绑定到对应主体资源项 |

建议映射格式：

图片主体：

- `Type=Url`
- `Category=Image`
- `Url=<subject image url>`
- `ObjectId=<server_id 或 name>`
- `Text=<subject name>`
- `VoiceId=<voice_id>`
- `Usage=Reference`

视频主体：

- `Type=Url`
- `Category=Video`
- `Url=<subject video url>`
- `ObjectId=<server_id 或 name>`
- `Text=<subject name>`
- `VoiceId=<voice_id>`

#### 当前不映射、应忽略的字段

| Vidu 字段 | 原因 |
|---|---|
| `auto_subjects` | 腾讯 VOD 无明确对应开关 |
| `audio_type` | 腾讯 VOD 无稳定一一对应字段 |
| `callback_url` | 不依赖上游回调 |
| `watermark` | 无统一稳定对应 |
| `wm_position` | 无统一稳定对应 |
| `movement_amplitude` | 无稳定对应 |

#### 特别说明

- 当前方案中，`subjects` 不生成 `SubjectInfos`，主体语义全部压到 `FileInfos` 中表达
- `subjects[].videos[]` 是否真正生效，还受腾讯 VOD当前 `ModelName + ModelVersion` 是否支持视频主体或视频参考限制
- `subjects[].voice_id` 映射后，最终是否被上游消费，也受具体模型能力约束

### 4.2 非主体调用

这里的“非主体调用”指 Vidu 官方 `reference2video` 请求不通过 `subjects`，而是直接使用顶层参考素材字段进行调用的方式。

Vidu 官方语义：

- 顶层 `images` 表示参考图片
- 顶层 `videos` 表示参考视频
- 不引入主体名、主体库 `server_id` 等主体概念

Tencent VOD 官方可承接的对应结构：

- `FileInfos[]`
  - `Category`
  - `Type`
  - `Url`
  - `Usage`

#### 可映射字段

| Vidu 字段 | Tencent VOD 字段 | 说明 |
|---|---|---|
| `images[]` | `FileInfos[]` | 每张图生成一项参考图片输入 |
| `videos[]` | `FileInfos[]` | 每个视频生成一项参考视频输入 |

建议映射格式：

参考图片：

- `Type=Url`
- `Category=Image`
- `Url=<image url>`
- `Usage=Reference`

参考视频：

- `Type=Url`
- `Category=Video`
- `Url=<video url>`

#### 当前不映射、应忽略的字段

| Vidu 字段 | 原因 |
|---|---|
| `audio_type` | 腾讯 VOD 无稳定一一对应字段 |
| `callback_url` | 不依赖上游回调 |
| `watermark` | 无统一稳定对应 |
| `wm_position` | 无统一稳定对应 |
| `movement_amplitude` | 无稳定对应 |
| `auto_subjects` | 非主体调用下无对应意义，腾讯 VOD 也无对应开关 |

#### 特别说明

- 非主体调用不会生成 `SubjectInfos`
- 是否支持参考视频，仍受腾讯 VOD 当前 `ModelName + ModelVersion` 是否支持视频参考限制

## 直接保留并继续支持的腾讯 VOD 原生字段

如果用户直接通过 `metadata` 传入腾讯 VOD 原生字段，仍继续按现有逻辑支持：

- `negative_prompt`
- `enhance_prompt`
- `input_region`
- `scene_type`
- `procedure`
- `session_id`
- `session_context`
- `tasks_priority`
- `ext_info`
- `generation_mode`
- `output_config`
- `file_infos`
- `subject_infos`
- `last_frame_file_id`
- `last_frame_url`

## 当前统一忽略字段清单

这些字段目前都属于“保留在 metadata，但 TencentVODVideo adaptor 不消费”：

- `callback_url`
- `audio_type`
- `style`
- `is_rec`
- `movement_amplitude`
- `watermark`
- `wm_position`
- `auto_subjects`

## 实现约束

1. 映射逻辑只放在 `relay/channel/task/tencentvod/adaptor.go`
2. 不回改 `m_vidu` compat 层去派生腾讯专用字段
3. 原始 Vidu 字段仍继续保留在 `metadata`
4. 对模型能力不支持的情况，不在 compat 层抢先报错，优先让腾讯 VOD 上游报错
