# logsagg

TimescaleDB 日志聚合独立程序，所有实现与测试均位于 `programs/logsagg/`。

在以下示例中，默认先进入当前目录：

```bash
cd programs/logsagg
```

本地开发常用命令：

```bash
go test ./...
go build .
go run .
```

配置规则：

- `LOGSAGG_SOURCE_DSN`：源数据库，读取原始 `logs`
- `LOGSAGG_TARGET_DSN`：目标数据库，写入 `logs_1m`、`logs_5m`、`logs_agg_state`
- `LOGSAGG_DSN`：兼容旧配置；未设置 `LOGSAGG_SOURCE_DSN` 时作为源库；未设置 `LOGSAGG_TARGET_DSN` 时目标库回退为源库

同库运行示例：

```bash
LOGSAGG_MODE=run \
LOGSAGG_DSN='postgres://user:pass@localhost:5432/db?sslmode=disable' \
go run .
```

分库运行示例：

```bash
LOGSAGG_MODE=run \
LOGSAGG_SOURCE_DSN='postgres://reader:pass@source-host:5432/raw_db?sslmode=disable' \
LOGSAGG_TARGET_DSN='postgres://writer:pass@target-host:5432/agg_db?sslmode=disable' \
go run .
```

历史回填示例：

```bash
LOGSAGG_MODE=backfill \
LOGSAGG_SOURCE_DSN='postgres://reader:pass@source-host:5432/raw_db?sslmode=disable' \
LOGSAGG_TARGET_DSN='postgres://writer:pass@target-host:5432/agg_db?sslmode=disable' \
LOGSAGG_BACKFILL_START='2026-05-14T00:00:00Z' \
LOGSAGG_BACKFILL_END='2026-05-14T01:00:00Z' \
go run .
```

说明：

- 源表 `logs` 的存在性与字段契约检查在源数据库执行
- `logs_1m`、`logs_5m`、continuous aggregate policy、`logs_agg_state` 的创建与维护都在目标数据库执行
- `logs_agg_state` 固定放在目标数据库，不单独提供第三个状态库配置

程序运行时会输出批次级日志，包含以下字段：

- `run`: `incremental`、`rewind`、`rewind-startup`、`backfill`
- `started_at`
- `duration`
- `source_rows`
- `facts_written`
- `max_row_id`
- `window_start`
- `window_end`
- `last_id`
- `last_success_bucket`
- `failures`

## Grafana dashboard 导入脚本

目录：

- Dashboard 模板：`grafana/logsagg-dimensions-dashboard.json`
- 导入脚本：`scripts/apply-grafana-dashboard.sh`
- 一键运行脚本：`scripts/run-apply-grafana-dashboard.sh`
- 本地 dry-run 测试：`scripts/test-apply-grafana-dashboard.sh`

用途：

- 假设 Grafana 的 PostgreSQL datasource 已存在
- 默认直接创建或更新顶层 dashboard `mmodel`
- 如有需要，也支持通过环境变量指定 folder
- 使用 `overwrite: true`，可重复执行

必需环境变量：

- `GRAFANA_URL`：Grafana 地址，例如 `https://grafana.example.com`
- `GRAFANA_TOKEN`：Grafana API token
- `GRAFANA_DATASOURCE_UID`：已存在的 PostgreSQL datasource uid

可选环境变量：

- `GRAFANA_FOLDER`：可选，指定 folder 标题
- `GRAFANA_FOLDER_UID`：可选，指定 folder uid；仅设置该值时才会自动创建/复用 folder
- `GRAFANA_DASHBOARD_TITLE`：dashboard 标题，默认 `mmodel`
- `DRY_RUN=1`：只生成最终 Grafana API payload，不调用 Grafana
- `DRY_RUN_OUTPUT=/tmp/payload.json`：指定 dry-run 输出文件

先做本地 dry-run 校验：

```bash
cd programs/logsagg
./scripts/test-apply-grafana-dashboard.sh
```

只生成 payload，不调用 Grafana：

```bash
cd programs/logsagg

GRAFANA_URL='https://grafana.example.com' \
GRAFANA_TOKEN='your-token' \
GRAFANA_DATASOURCE_UID='your-postgres-datasource-uid' \
DRY_RUN=1 \
DRY_RUN_OUTPUT='/tmp/logsagg-dashboard-payload.json' \
./scripts/apply-grafana-dashboard.sh
```

实际导入或更新 Grafana dashboard：

```bash
cd programs/logsagg

GRAFANA_URL='https://grafana.example.com' \
GRAFANA_TOKEN='your-token' \
GRAFANA_DATASOURCE_UID='your-postgres-datasource-uid' \
./scripts/apply-grafana-dashboard.sh
```

如果当前环境的 Grafana 参数已经固定，也可以直接执行一键脚本：

```bash
cd programs/logsagg
./scripts/run-apply-grafana-dashboard.sh
```

注意：

- `scripts/run-apply-grafana-dashboard.sh` 当前把 Grafana URL、token 和 datasource uid 写死在脚本里
- 适合本地或内网临时运维使用
- 如果仓库会推送到远端，建议改回环境变量方式，避免明文凭证进入版本库
