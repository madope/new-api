#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

GRAFANA_URL='http://60.188.115.89:3000/' \
GRAFANA_TOKEN='glsa_yclaQNBeIBBL8rel8yRHAH6eeZJpnsj8_2b41cfe1' \
GRAFANA_DATASOURCE_UID='bfmid3a3dqjuoa' \
"$SCRIPT_DIR/apply-grafana-dashboard.sh"
