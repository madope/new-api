#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
TEMPLATE_PATH="$ROOT_DIR/grafana/logsagg-dimensions-dashboard.json"

GRAFANA_URL="${GRAFANA_URL:-}"
GRAFANA_TOKEN="${GRAFANA_TOKEN:-}"
GRAFANA_FOLDER="${GRAFANA_FOLDER:-}"
GRAFANA_FOLDER_UID="${GRAFANA_FOLDER_UID:-}"
GRAFANA_DATASOURCE_UID="${GRAFANA_DATASOURCE_UID:-}"
GRAFANA_DASHBOARD_TITLE="${GRAFANA_DASHBOARD_TITLE:-mmodel}"
DRY_RUN="${DRY_RUN:-0}"
DRY_RUN_OUTPUT="${DRY_RUN_OUTPUT:-}"

usage() {
  cat <<'EOF'
Usage:
  GRAFANA_URL=... \
  GRAFANA_TOKEN=... \
  GRAFANA_DATASOURCE_UID=... \
  programs/logsagg/scripts/apply-grafana-dashboard.sh

Optional env:
  GRAFANA_FOLDER=<optional-folder-title>
  GRAFANA_FOLDER_UID=<optional-folder-uid>
  GRAFANA_DASHBOARD_TITLE="mmodel"
  DRY_RUN=1
  DRY_RUN_OUTPUT=/tmp/logsagg-dashboard-payload.json

Behavior:
  - If folder uid/title are set, ensures the target Grafana folder exists
  - Creates or updates the dashboard with a fixed uid
  - Uses overwrite=true so repeated runs are idempotent
  - In DRY_RUN mode, writes the final API payload instead of calling Grafana
EOF
}

require_tool() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required tool not found: $1" >&2
    exit 1
  fi
}

require_env() {
  local name="$1"
  local value="$2"
  if [[ -z "$value" ]]; then
    echo "required env missing: $name" >&2
    usage >&2
    exit 1
  fi
}

api_call() {
  local method="$1"
  local path="$2"
  local body_file="${3:-}"
  local response_file
  local status

  response_file="$(mktemp)"
  if [[ -n "$body_file" ]]; then
    status="$(curl -sS -o "$response_file" -w '%{http_code}' \
      -X "$method" \
      -H "Authorization: Bearer $GRAFANA_TOKEN" \
      -H "Content-Type: application/json" \
      --data @"$body_file" \
      "${GRAFANA_URL%/}$path")"
  else
    status="$(curl -sS -o "$response_file" -w '%{http_code}' \
      -X "$method" \
      -H "Authorization: Bearer $GRAFANA_TOKEN" \
      "${GRAFANA_URL%/}$path")"
  fi

  printf '%s\n%s\n' "$status" "$response_file"
}

build_payload() {
  local output_path="$1"

  node - "$TEMPLATE_PATH" "$output_path" "$GRAFANA_DATASOURCE_UID" "$GRAFANA_FOLDER_UID" "$GRAFANA_DASHBOARD_TITLE" <<'EOF'
const fs = require('fs');

const [templatePath, outputPath, datasourceUid, folderUid, title] = process.argv.slice(2);
const payload = JSON.parse(fs.readFileSync(templatePath, 'utf8'));
const datasource = { type: 'postgres', uid: datasourceUid };

if (folderUid) {
  payload.folderUid = folderUid;
} else {
  delete payload.folderUid;
}
payload.dashboard.title = title;
payload.dashboard.version = 0;

function visit(value) {
  if (Array.isArray(value)) {
    value.forEach(visit);
    return;
  }
  if (!value || typeof value !== 'object') {
    return;
  }
  if (Object.prototype.hasOwnProperty.call(value, 'datasource')) {
    value.datasource = datasource;
  }
  Object.values(value).forEach(visit);
}

visit(payload.dashboard);
fs.writeFileSync(outputPath, JSON.stringify(payload, null, 2));
EOF
}

ensure_folder() {
  local check
  local status
  local response_file
  local create_payload

  check="$(api_call GET "/api/folders/${GRAFANA_FOLDER_UID}")"
  status="$(printf '%s' "$check" | sed -n '1p')"
  response_file="$(printf '%s' "$check" | sed -n '2p')"
  if [[ "$status" == "200" ]]; then
    rm -f "$response_file"
    return 0
  fi
  rm -f "$response_file"

  create_payload="$(mktemp)"
  node - "$create_payload" "$GRAFANA_FOLDER_UID" "$GRAFANA_FOLDER" <<'EOF'
const fs = require('fs');
const [outputPath, uid, title] = process.argv.slice(2);
fs.writeFileSync(outputPath, JSON.stringify({ uid, title }, null, 2));
EOF

  check="$(api_call POST "/api/folders" "$create_payload")"
  status="$(printf '%s' "$check" | sed -n '1p')"
  response_file="$(printf '%s' "$check" | sed -n '2p')"
  if [[ "$status" != "200" ]]; then
    echo "failed to create Grafana folder ${GRAFANA_FOLDER_UID}" >&2
    cat "$response_file" >&2
    rm -f "$create_payload" "$response_file"
    exit 1
  fi

  rm -f "$create_payload" "$response_file"
}

main() {
  local payload_file
  local result
  local status
  local response_file

  require_tool curl
  require_tool node
  require_env GRAFANA_URL "$GRAFANA_URL"
  require_env GRAFANA_TOKEN "$GRAFANA_TOKEN"
  require_env GRAFANA_DATASOURCE_UID "$GRAFANA_DATASOURCE_UID"

  if [[ ! -f "$TEMPLATE_PATH" ]]; then
    echo "dashboard template not found: $TEMPLATE_PATH" >&2
    exit 1
  fi

  payload_file="$(mktemp)"
  build_payload "$payload_file"

  if [[ "$DRY_RUN" == "1" ]]; then
    if [[ -n "$DRY_RUN_OUTPUT" ]]; then
      cp "$payload_file" "$DRY_RUN_OUTPUT"
    else
      cat "$payload_file"
    fi
    rm -f "$payload_file"
    return 0
  fi

  if [[ -n "$GRAFANA_FOLDER_UID" ]]; then
    require_env GRAFANA_FOLDER "$GRAFANA_FOLDER"
    ensure_folder
  fi

  result="$(api_call POST "/api/dashboards/db" "$payload_file")"
  status="$(printf '%s' "$result" | sed -n '1p')"
  response_file="$(printf '%s' "$result" | sed -n '2p')"
  if [[ "$status" != "200" ]]; then
    echo "failed to apply Grafana dashboard" >&2
    cat "$response_file" >&2
    rm -f "$payload_file" "$response_file"
    exit 1
  fi

  echo "Grafana dashboard applied successfully:"
  cat "$response_file"
  rm -f "$payload_file" "$response_file"
}

main "$@"
