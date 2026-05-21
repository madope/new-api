#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCRIPT_PATH="$ROOT_DIR/scripts/apply-grafana-dashboard.sh"
TMP_DIR="$(mktemp -d)"
PAYLOAD_PATH="$TMP_DIR/payload.json"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

if ! command -v node >/dev/null 2>&1; then
  echo "node is required for this test" >&2
  exit 1
fi

echo "test: generate dry-run payload"
GRAFANA_URL="http://grafana.example.com" \
GRAFANA_TOKEN="test-token" \
GRAFANA_DATASOURCE_UID="postgres-main" \
DRY_RUN=1 \
DRY_RUN_OUTPUT="$PAYLOAD_PATH" \
"$SCRIPT_PATH"

node - "$PAYLOAD_PATH" <<'EOF'
const fs = require('fs');
const path = process.argv[2];
const payload = JSON.parse(fs.readFileSync(path, 'utf8'));

function assert(condition, message) {
  if (!condition) {
    throw new Error(message);
  }
}

assert(!('folderUid' in payload), 'folderUid should be omitted');
assert(payload.overwrite === true, 'overwrite mismatch');
assert(payload.dashboard.uid === 'logsagg-dimensions', 'dashboard uid mismatch');
assert(payload.dashboard.title === 'mmodel', 'dashboard title mismatch');
assert(Array.isArray(payload.dashboard.panels) && payload.dashboard.panels.length >= 27, 'panels missing');
assert(Array.isArray(payload.dashboard.templating.list) && payload.dashboard.templating.list.length >= 6, 'variables missing');
const firstDataPanel = payload.dashboard.panels.find((panel) => panel.type !== 'row');
assert(firstDataPanel && firstDataPanel.datasource.uid === 'postgres-main', 'panel datasource mismatch');
assert(firstDataPanel.targets[0].datasource.uid === 'postgres-main', 'target datasource mismatch');

const titles = new Set(payload.dashboard.panels.map((panel) => panel.title));
[
  '用户维度',
  '渠道维度',
  '令牌维度',
  '模型维度',
  '分组维度',
  '用户请求数',
  '用户 Token 消耗',
  '用户输入 Token 消耗',
  '用户输出 Token 消耗',
  '用户额度消耗',
  '模型请求数',
  '模型 Token 消耗',
  '模型输入 Token 消耗',
  '模型输出 Token 消耗',
  '模型额度消耗',
  '渠道请求数',
  '渠道 Token 消耗',
  '渠道输入 Token 消耗',
  '渠道输出 Token 消耗',
  '渠道额度消耗',
  '令牌请求数',
  '令牌 Token 消耗',
  '令牌输入 Token 消耗',
  '令牌输出 Token 消耗',
  '令牌额度消耗',
  '分组请求数',
  '分组 Token 消耗',
  '分组输入 Token 消耗',
  '分组输出 Token 消耗',
  '分组额度消耗',
  '用户总览明细',
  '用户-令牌明细',
  '渠道总览明细',
  '渠道-模型明细',
  '令牌总览明细',
  '令牌-模型明细',
  '模型总览明细',
].forEach((title) => assert(titles.has(title), `missing panel: ${title}`));

assert(payload.dashboard.panels.filter((panel) => panel.type === 'barchart').length === 0, 'barcharts should be removed');

const quotaPanels = payload.dashboard.panels.filter((panel) => typeof panel?.title === 'string' && panel.title.includes('额度消耗'));
assert(quotaPanels.length === 5, 'unexpected quota panel count');
quotaPanels.forEach((panel) => {
  assert(panel.fieldConfig?.defaults?.unit === 'currencyUSD', `quota panel unit mismatch: ${panel.title}`);
  assert(panel.targets?.[0]?.rawSql?.includes('/ 500000.0 AS value'), `quota SQL mismatch: ${panel.title}`);
});

const quotaTables = payload.dashboard.panels.filter((panel) => Array.isArray(panel?.targets) && panel.targets[0]?.rawSql?.includes('/ 500000.0 AS quota_used'));
assert(quotaTables.length >= 7, 'unexpected quota table count');
quotaTables.forEach((panel) => {
  const override = panel.fieldConfig?.overrides?.find((item) => item?.matcher?.id === 'byName' && item?.matcher?.options === 'quota_used');
  assert(override, `missing quota_used override: ${panel.title}`);
  assert(override.properties?.some((item) => item.id === 'unit' && item.value === 'currencyUSD'), `quota_used unit mismatch: ${panel.title}`);
});

const variableNames = new Set(payload.dashboard.templating.list.map((item) => item.name));
assert(variableNames.has('v_token'), 'missing token variable');
EOF

echo "ok"
