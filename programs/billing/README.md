# billing

TimescaleDB 账单查询独立程序，读取 `logsagg` 写入的 `logs_1m` 聚合表并提供 HTTP API。

本地开发：

```bash
cd programs/billing
go test ./...
go run .
```

构建 Docker 镜像：

```bash
cd programs/billing
docker build -t new-api-billing:latest .
```

启动 Docker 容器：

```bash
docker run --rm \
  -p 3011:3011 \
  -e BILLING_DSN='postgres://user:pass@host.docker.internal:5432/agg_db?sslmode=disable' \
  -e BILLING_HOST='0.0.0.0' \
  -e BILLING_PORT='3011' \
  new-api-billing:latest
```

如果是 Linux 宿主机，`host.docker.internal` 不可用时，改成实际数据库地址或使用：

```bash
docker run --rm \
  --add-host=host.docker.internal:host-gateway \
  -p 3011:3011 \
  -e BILLING_DSN='postgres://user:pass@host.docker.internal:5432/agg_db?sslmode=disable' \
  -e BILLING_HOST='0.0.0.0' \
  -e BILLING_PORT='3011' \
  new-api-billing:latest
```

配置：

- `BILLING_DSN`：账单查询数据库 DSN，必填
- `BILLING_HOST`：默认 `127.0.0.1`
- `BILLING_PORT`：默认 `3011`
- `BILLING_DEFAULT_PAGE_SIZE`：默认 `20`
- `BILLING_MAX_PAGE_SIZE`：默认 `100`
- `BILLING_READ_TIMEOUT_SECONDS`：默认 `15`

接口：

- `GET /api/v1/billing/healthz`
- `GET /api/v1/billing/query`
- `GET /api/v1/billing/swagger/index.html`
- `GET /api/v1/billing/swagger/openapi.json`

Swagger UI：

- 浏览器访问 `http://127.0.0.1:3011/api/v1/billing/swagger/index.html`
- OpenAPI JSON 地址：`http://127.0.0.1:3011/api/v1/billing/swagger/openapi.json`

## API 文档

### 1. 健康检查

请求：

```http
GET /api/v1/billing/healthz
```

成功响应：

```json
{
  "success": true,
  "message": "",
  "data": {
    "status": "ok"
  }
}
```

### 2. 账单查询

请求：

```http
GET /api/v1/billing/query
```

#### 请求参数

必填参数：

- `start_timestamp`：开始时间，Unix 秒
- `end_timestamp`：结束时间，Unix 秒，必须大于 `start_timestamp`

可选过滤参数：

- `user_id`：按用户 ID 过滤
- `username`：按用户名过滤
- `token_id`：按令牌 ID 过滤
- `token_name`：按令牌名过滤
- `channel_id`：按渠道 ID 过滤
- `channel_name`：按渠道名过滤
- `model_name`：按模型名过滤
- `group_name`：按分组名过滤
- `is_stream`：是否流式，支持 `true`、`false`、`1`
- `type`：日志类型，默认 `2`，表示消费日志

可选分组参数：

- `group_by`：逗号分隔，允许值：
  - `user`
  - `token`
  - `channel`
  - `model`
  - `group`
  - `stream`
  - `type`

可选分页参数：

- `page`：页码，从 `1` 开始，默认 `1`
- `page_size`：每页条数，默认 `20`，最大值由 `BILLING_MAX_PAGE_SIZE` 控制

可选排序参数：

- `order_by`：允许值：
  - `amount`
  - `quota`
  - `requests`
  - `total_tokens`
  - `prompt_tokens`
  - `completion_tokens`
  - `avg_use_time_ms`
  - `max_use_time_ms`
- `order`：`asc` 或 `desc`，默认 `desc`

#### 请求示例

查询某个用户在一段时间内的总账单：

```http
GET /api/v1/billing/query?start_timestamp=1748736000&end_timestamp=1748822400&user_id=123
```

查询某个用户按渠道拆分账单：

```http
GET /api/v1/billing/query?start_timestamp=1748736000&end_timestamp=1748822400&user_id=123&group_by=channel
```

查询所有用户并分页：

```http
GET /api/v1/billing/query?start_timestamp=1748736000&end_timestamp=1748822400&group_by=user&page=1&page_size=3
```

`curl` 示例：

```bash
curl "http://127.0.0.1:3011/api/v1/billing/query?start_timestamp=1748736000&end_timestamp=1748822400&group_by=user&page=1&page_size=3"
```

#### 响应结构

成功响应格式：

```json
{
  "success": true,
  "message": "",
  "data": {
    "summary": {
      "requests": 1000,
      "quota": 12000000,
      "amount": 24.0,
      "prompt_tokens": 800000,
      "completion_tokens": 400000,
      "total_tokens": 1200000,
      "avg_use_time_ms": 350,
      "max_use_time_ms": 2200
    },
    "items": [
      {
        "user_id": 1,
        "username": "alice",
        "requests": 80,
        "quota": 1000000,
        "amount": 2.0,
        "prompt_tokens": 70000,
        "completion_tokens": 30000,
        "total_tokens": 100000,
        "avg_use_time_ms": 400,
        "max_use_time_ms": 900
      }
    ],
    "page": {
      "page": 1,
      "page_size": 3,
      "total": 53,
      "total_pages": 18
    },
    "meta": {
      "group_by": [
        "user"
      ],
      "order_by": "amount",
      "order": "desc",
      "amount_divisor": 500000,
      "filters": {
        "start_timestamp": 1748736000,
        "end_timestamp": 1748822400,
        "user_id": 0,
        "username": "",
        "token_id": 0,
        "token_name": "",
        "channel_id": 0,
        "channel_name": "",
        "model_name": "",
        "group_name": "",
        "type": 2,
        "is_stream": null
      }
    }
  }
}
```

字段说明：

- `summary`：整次查询范围内的总计，不受当前分页影响
- `items`：当前页的数据
- `page.total`：总条数，不是总页数
- `page.total_pages`：总页数
- `meta.amount_divisor`：金额换算除数，当前固定为 `500000`

#### `items` 的含义

不传 `group_by` 时：

- `items` 只有 `1` 条
- 表示整个筛选范围内的总账单
- 不包含 `user_id`、`channel_id` 这类分组字段

示例：

```json
{
  "items": [
    {
      "requests": 86,
      "quota": 1350000,
      "amount": 2.7,
      "prompt_tokens": 92000,
      "completion_tokens": 43000,
      "total_tokens": 135000,
      "avg_use_time_ms": 410,
      "max_use_time_ms": 1600
    }
  ]
}
```

传 `group_by=user` 时：

- `items` 中每一条表示一个用户的聚合账单

传 `group_by=user,channel` 时：

- `items` 中每一条表示某个用户在某个渠道下的聚合账单

#### 分页说明

如果请求：

```http
GET /api/v1/billing/query?start_timestamp=1748736000&end_timestamp=1748822400&group_by=user&page=1&page_size=3
```

则表示：

- 查询按用户聚合的账单
- 取第 `1` 页
- 每页返回 `3` 条

如何知道一共有多少条：

- 看响应中的 `page.total`

如何知道一共有多少页：

- 看响应中的 `page.total_pages`

#### 错误响应

参数错误示例：

```json
{
  "success": false,
  "message": "invalid group_by: bad",
  "data": null
}
```

常见错误：

- `BILLING_DSN is required`
- `start_timestamp is required`
- `end_timestamp is required`
- `end_timestamp must be greater than start_timestamp`
- `invalid group_by: xxx`
- `invalid order_by: xxx`
- `invalid order: xxx`
