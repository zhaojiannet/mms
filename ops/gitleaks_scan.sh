#!/usr/bin/env bash
# 本地手动跑 gitleaks 扫描（容器化，不污染宿主机）
#
# 推荐：作为 git pre-commit hook 或在发版前手动跑

set -e
REPO=$(git rev-parse --show-toplevel)
docker run --rm -v "$REPO:/repo" zricethezav/gitleaks:latest \
  detect --source /repo --config /repo/.gitleaks.toml --no-banner --redact "$@"
