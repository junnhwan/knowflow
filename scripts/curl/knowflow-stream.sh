#!/usr/bin/env bash
set -euo pipefail

if [[ $# -lt 1 ]]; then
  echo "用法: ./knowflow-stream.sh <session_id> [message]" >&2
  exit 1
fi

BASE_URL="${KNOWFLOW_BASE_URL:-http://localhost:8080}"
USER_ID="${KNOWFLOW_USER_ID:-demo-user}"
SESSION_ID="$1"
MESSAGE="${2:-继续解释一下 Redis 记忆压缩的设计}"

curl -N -X POST "$BASE_URL/api/chat/query/stream" \
  -H "Content-Type: application/json" \
  -H "X-User-ID: $USER_ID" \
  -d "{\"session_id\":\"$SESSION_ID\",\"message\":\"$MESSAGE\"}"
