# Volces Seedance Compatible User API Design

## Goal

Add a user-facing API surface that feels like Volcengine Ark Seedance video generation while continuing to use the existing `new-api` task pipeline internally.

Target paths:

- `POST /volces/api/v3/contents/generations/tasks`
- `GET /volces/api/v3/contents/generations/tasks/:task_id`

Compatibility goals:

- Request path and request body format are compatible with the Volces Seedance API.
- Response field names, field nesting, and field semantics are compatible with the Volces Seedance API.
- The submit response keeps the native shape but replaces upstream `id` with `new-api` public task id.
- The fetch response also returns the `new-api` public task id in `id`.
- Existing `new-api` task creation, polling, billing, retry, and permission checks remain the single source of truth.

Non-goals:

- Byte-for-byte transparent proxying of upstream responses.
- Exposing upstream task ids to users.
- Introducing a second independent async task system.

## Current State

Existing task entrypoints:

- `POST /v1/video/generations`
- `GET /v1/video/generations/:task_id`
- `POST /v1/videos`
- `GET /v1/videos/:task_id`

Relevant code:

- Routing: [router/video-router.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/router/video-router.go)
- Task submit/fetch orchestration: [relay/relay_task.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/relay/relay_task.go)
- Task controller: [controller/relay.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/controller/relay.go)
- Doubao task adaptor: [relay/channel/task/doubao/adaptor.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/relay/channel/task/doubao/adaptor.go)
- Task model: [model/task.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/model/task.go)

Current limitations:

1. The Doubao task adaptor only accepts the unified `new-api` task request body.
2. Submit responses currently return OpenAI-style video payloads instead of Volces-compatible submit payloads.
3. Fetch responses currently return either:
   - OpenAI video payloads, or
   - generic `TaskResponse`
4. `task.Data` is reused for different purposes over time and cannot cleanly represent both submit-response and fetch-response snapshots.

## Recommended Approach

Add a dedicated `/volces` compatibility layer on top of the existing task pipeline.

Why this approach:

- It keeps Volces-specific request and response adaptation isolated.
- It avoids polluting `/v1/video/generations` with provider-private protocol rules.
- It preserves current billing, retries, task polling, and storage behavior.
- It creates a clean extension point for future Volces-native APIs.

Rejected alternatives:

1. Overload `/v1/video/generations` to accept both unified and Volces-native bodies.
   - Rejected because it couples the unified API with a provider-private schema.
2. Bypass the task system and proxy directly to upstream.
   - Rejected because it would lose current public task id behavior, internal billing lifecycle, and polling consistency.

## API Surface

### Submit

`POST /volces/api/v3/contents/generations/tasks`

Request body: Volces-native Seedance task submission body.

Response body shape:

```json
{
  "id": "task_xxx"
}
```

Notes:

- Field set remains compatible with upstream submit response.
- `id` value is replaced with the `new-api` public task id.

### Fetch

`GET /volces/api/v3/contents/generations/tasks/:task_id`

Response body shape:

- Compatible with the upstream Volces task fetch response shape.
- `id` is replaced with the `new-api` public task id.
- Other fields are sourced from the latest upstream fetch response.

## Request Conversion

Add a Volces-specific middleware, tentatively:

- `middleware/volces_seedance_adapter.go`

Responsibilities:

1. Parse native Volces request body.
2. Convert it into the existing unified task request format used by `/v1/video/generations`.
3. Rewrite the internal path to `/v1/video/generations`.
4. Store the rewritten body in reusable request storage.

### Unified Internal Task Body

Internal shape is the existing task request DTO:

- `model`
- `prompt`
- `image`
- `images`
- `seconds`
- `duration`
- `size`
- `input_reference`
- `metadata`

### Mapping: Volces Native -> Unified Task Body

- `model` -> `model`
- first `content[].text` -> `prompt`
- `duration` -> `seconds`
- non-text `content[]` -> `metadata.content`
- `generate_audio` -> `metadata.generate_audio`
- `ratio` -> `metadata.ratio`
- `resolution` -> `metadata.resolution`
- `watermark` -> `metadata.watermark`
- `seed` -> `metadata.seed`
- `camera_fixed` -> `metadata.camera_fixed`
- `service_tier` -> `metadata.service_tier`
- `execution_expires_after` -> `metadata.execution_expires_after`
- `return_last_frame` -> `metadata.return_last_frame`
- `draft` -> `metadata.draft`
- `tools` -> `metadata.tools`
- `callback_url` -> `metadata.callback_url`

Rules:

- Text content is normalized into a single `prompt`.
- Non-text content stays in `metadata.content`.
- Do not duplicate text entries by keeping text both in `prompt` and `metadata.content`.

## Upstream Conversion

No architectural change is required in the task pipeline:

1. The middleware converts native Volces request body into unified task body.
2. Existing `RelayTask` path submits through the unified flow.
3. Existing Doubao task adaptor converts unified task body back into upstream Seedance body.

Current behavior in the Doubao adaptor is already close to this:

- `model` -> upstream `model`
- `prompt` -> appended as a `content` text item
- `seconds` -> upstream `duration`
- `metadata` -> JSON-expanded into Doubao request payload

Required refinement:

- Ensure Volces-native `content` items from `metadata.content` and the generated text item compose correctly without duplicate text entries.

## Response Compatibility

### Submit Response

For `/volces/...` submit requests:

- Return Volces-compatible submit response body shape.
- Replace upstream `id` with `info.PublicTaskID`.

For non-`/volces/...` requests:

- Keep existing behavior unchanged.

### Fetch Response

For `/volces/...` fetch requests:

- Return Volces-compatible fetch response body shape.
- Replace `id` with public task id.
- Source all other compatible fields from the latest upstream fetch response snapshot.

For non-`/volces/...` requests:

- Keep existing OpenAI/generic fetch behavior unchanged.

## Persistence Strategy

To support Volces-compatible fetch responses, persist separate upstream response snapshots.

### Current Stored Data

Current task persistence already stores:

- public task id: `task.TaskID`
- upstream task id: `task.PrivateData.UpstreamTaskID`
- latest `task.Data`
- task lifecycle status, progress, quota, action, channel id

### Current Gaps

The system does not separately preserve:

- original submit response body
- latest upstream fetch response body
- timestamp of latest fetch snapshot

### Proposed Additions

Extend `TaskPrivateData` in [model/task.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/model/task.go):

- `SubmitResponse json.RawMessage`
- `FetchResponse json.RawMessage`
- `FetchResponseUpdatedAt int64`

Why `private_data`:

- no new relational columns required
- cross-database friendly
- scoped to task-internal transport metadata

### Write Points

On submit success in [controller/relay.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/controller/relay.go):

- continue storing `task.Data = result.TaskData`
- additionally store `task.PrivateData.SubmitResponse = result.TaskData`

On polling refresh in [service/task_polling.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/service/task_polling.go):

- continue current `task.Data` handling
- additionally store:
  - `task.PrivateData.FetchResponse = responseBody`
  - `task.PrivateData.FetchResponseUpdatedAt = now`

## Fetch Strategy

Recommended behavior for `/volces/...` fetch:

1. Load task by public task id.
2. If a recent `FetchResponse` snapshot exists, build Volces-compatible response from it.
3. If no snapshot exists yet, fetch upstream once using `UpstreamTaskID`, persist the fetch snapshot, and build response from that body.

Reason:

- First user fetch may occur before background polling writes a full upstream fetch snapshot.
- This avoids a compatibility gap immediately after submit.

## File-Level Change Plan

### Routing

Update [router/video-router.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/router/video-router.go):

- add `POST /volces/api/v3/contents/generations/tasks`
- add `GET /volces/api/v3/contents/generations/tasks/:task_id`

### Middleware

Add:

- `middleware/volces_seedance_adapter.go`

Responsibilities:

- native body parsing
- unified body rewriting
- rewritten body persistence in request storage
- path rewrite to `/v1/video/generations`

### Task Persistence

Update [model/task.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/model/task.go):

- extend `TaskPrivateData`

Update [controller/relay.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/controller/relay.go):

- persist submit response snapshot

Update [service/task_polling.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/service/task_polling.go):

- persist fetch response snapshot

### Fetch/Response Builders

Update [relay/relay_task.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/relay/relay_task.go):

- add a `/volces`-specific fetch response builder path
- implement compatibility response generation
- add fallback realtime upstream fetch when snapshot is absent

### Doubao Adaptor

Update [relay/channel/task/doubao/adaptor.go](/Users/cxxhb/projects/new-api-v1.0.0-rc.6-online/new-api/relay/channel/task/doubao/adaptor.go):

- for `/volces/...` submit requests, return Volces-compatible submit response shape
- keep current behavior for `/v1/video/generations` and `/v1/videos`

## Error Handling

Submit path:

- request parsing errors return Volces-compatible JSON error where feasible
- upstream failure remains mapped through existing task error handling

Fetch path:

- unknown public task id returns task-not-found style error
- upstream fetch failure should preserve current retry-safe behavior internally
- when compatibility response cannot be constructed, return a clear server error rather than partial malformed Volces payload

## Testing

Add tests for:

1. request conversion
   - native Volces content array converts correctly into unified task body
2. submit response compatibility
   - `/volces/...` submit returns `{"id":"task_xxx"}`
3. persistence
   - submit response snapshot is saved
   - fetch response snapshot is saved
4. fetch compatibility
   - response body shape matches Volces-compatible structure
   - `id` is replaced by public task id
5. fallback fetch
   - first fetch without cached snapshot performs upstream fetch and stores it
6. regression
   - existing `/v1/video/generations` and `/v1/videos` behavior remains unchanged

## Risks

1. Upstream Volces response schema may evolve.
   - Mitigation: preserve fetch snapshot bodies and minimize hard-coded field synthesis.
2. Multiple text items in native `content`.
   - Mitigation: define deterministic normalization into one `prompt`.
3. First fetch race before polling writes a snapshot.
   - Mitigation: realtime fetch fallback on `/volces/...` query.

## Open Decisions Already Resolved

- Use `/volces` prefix instead of raw upstream path at root.
- Keep public task id visible to users.
- Keep Volces-compatible request and response experience.
- Do not expose upstream task id.
- Do not introduce a second task system.
