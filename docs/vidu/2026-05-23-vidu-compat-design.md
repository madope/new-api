# m-vidu 用户侧兼容接口设计

## 目标

为 `new-api` 增加一套 `/m-vidu/...` 用户侧兼容接口，使使用 Vidu 官方 API 的用户程序尽量只修改 `base URL` 即可迁移。

本次设计明确收敛为：

- 路由前缀统一为 `/m-vidu`
- 对外请求路径使用 Vidu 官网 URI
- 首版支持 4 个生成接口和单任务查询
- 请求侧尽量兼容 Vidu 官方格式
- 响应侧返回 Vidu 官方核心结构
- 内部尽量简单，参考现有 `volces` 用户侧兼容实现
- 不在兼容层派生新的 provider-specific 字段

## 范围

首版支持：

- `POST /m-vidu/ent/v2/text2video`
- `POST /m-vidu/ent/v2/img2video`
- `POST /m-vidu/ent/v2/start-end2video`
- `POST /m-vidu/ent/v2/reference2video`
- `GET /m-vidu/ent/v2/tasks/:task_id/creations`

首版不做：

- `list`
- `cancel`
- webhook/callback 兼容
- 对所有非核心字段“所有上游都必须生效”的承诺

## 总体方案

采用“轻量兼容型”方案，风格参考：

- `middleware/kling_adapter.go`
- `middleware/volces_seedance_adapter.go`

核心原则：

1. compat 层只做请求改写和响应转换
2. 通用字段映射到现有 `TaskSubmitReq`
3. Vidu 特有字段不建专用内部结构，不派生新字段
4. 原始请求中的特殊字段直接保留到 `metadata`
5. 上游 adaptor 根据自身需要从 `metadata` 解析

## 与上一版方案的差异

本次明确放弃以下做法：

- 不新增 `relay/channel/task/m_vidu/m_vidu_request.go`
- 不为 Vidu 官方请求建立一整套强类型 DTO
- 不在 compat 层派生 `file_infos`
- 不在 compat 层派生 `last_frame_url`
- 不增加 `compat_provider / compat_mode`
- 不要求上游统一消费某种 compat 内部结构

原因：

- Vidu 官方字段可能继续增加
- 如果 compat 层把 Vidu 特有字段全部写死，就需要频繁跟随官网修改代码
- 如果 compat 层派生额外内部字段，会让所有上游 adaptor 和这套派生结构耦合

## 路由与入口

新增：

- `/m-vidu/ent/v2/text2video`
- `/m-vidu/ent/v2/img2video`
- `/m-vidu/ent/v2/start-end2video`
- `/m-vidu/ent/v2/reference2video`
- `/m-vidu/ent/v2/tasks/:task_id/creations`

提交接口继续复用：

- `controller.RelayTask`

查询接口继续复用：

- `controller.RelayTaskFetch`

## compat 层职责

建议只新增一个轻量入口文件：

- `middleware/m_vidu_adapter.go`

职责：

- 识别 `/m-vidu/ent/v2/...`
- 读取原始请求 `map[string]any`
- 抽取少量通用字段
- 把剩余字段直接保留到 `metadata`
- 设置内部 `action`
- 改写到统一 `/v1/video/generations` 链路

不负责：

- 定义 Vidu 全量字段 DTO
- 发明新的内部桥接字段
- 直接构造某个上游专用请求结构

## 请求转换原则

### 通用字段

只抽取已有统一任务模型真正需要的字段，例如：

- `model`
- `prompt`
- `images`
- `duration`

这些字段进入 `TaskSubmitReq`。

### 特有字段

Vidu 原始请求中除通用字段外的其余字段：

- 原样进入 `metadata`
- 不在 compat 层改名
- 不在 compat 层重组
- 不在 compat 层派生新的内部字段

例如：

- `subjects`
- `videos`
- `audio_type`
- `voice_id`
- `callback_url`
- `payload`
- 以及未来官网新增字段

都应直接保留到 `metadata`。

### 未知新字段

这是本版设计的关键点：

- 如果 Vidu 官网新增了新字段
- compat 层不需要先定义结构体字段
- 只要它仍然走原始 `map[string]any` 改写路径
- 新字段就会自动进入 `metadata`

这样可以避免每次官网加参数都必须同步改 compat 层代码。

## action 的作用

继续使用现有内部 `action` 做粗粒度分类：

- `text2video` -> `TaskActionTextGenerate`
- `img2video` -> `TaskActionGenerate`
- `start-end2video` -> `TaskActionFirstTailGenerate`
- `reference2video` -> `TaskActionReferenceGenerate`

这里只保留内部粗粒度语义，不承担保存 Vidu 官方字段的职责。

## 上游 adaptor 边界

兼容层不派生 `file_infos`、`last_frame_url` 等字段后，上游 adaptor 的职责变成：

- 从 `TaskSubmitReq` 读取通用字段
- 从 `metadata` 中读取自己关心的 Vidu 特有字段
- 自己决定如何映射到实际上游请求

这意味着：

- `TencentVODVideo` 可以按需要解析 `metadata.images`、`metadata.videos`、`metadata.subjects`
- 未来 `Volces`、`Ali`、其他上游如果承接 `/m-vidu`，也可以按各自需求读取原始字段
- 不强迫所有上游都理解统一派生结构

## 响应设计

### 提交响应

返回 Vidu 官方核心结构，不返回 OpenAI video object。

至少包含：

- `task_id`
- `state`

若某些官方字段可以从上游原始提交响应稳定获得，也可补充；否则不伪造。

### 查询响应

返回 Vidu 官方核心结构，不返回通用 `TaskResponse[TaskDto]`。

至少包含：

- `id`
- `state`
- `err_code`
- `creations`

如果上游原始查询结果里有可直接复用的字段，则优先复用；没有则按本地任务状态最小构造。

## 开发时的官方文档约束

实现时必须逐接口核对当时的 Vidu 官网文档：

- 请求字段名
- 请求字段类型
- 请求字段层级
- 响应字段名
- 响应字段类型
- 状态字段与错误字段

但这次不要求把所有字段都固化进 Go struct。  
兼容层允许：

- 核心字段显式抽取
- 非核心字段原样透传

## 为什么这版更适合当前项目

相比重型 DTO 方案，这版优点是：

- 和现有 `volces` compat 层风格一致
- 与上游 adaptor 解耦更好
- 对 Vidu 官方字段演进更友好
- 代码量更少
- 更容易重新开发和逐步迭代

核心取舍是：

- 不追求“compat 层完全理解 Vidu 全量字段”
- 只保证：
  - 请求入口兼容
  - 核心字段生效
  - 特有字段透传
  - 响应核心结构兼容
