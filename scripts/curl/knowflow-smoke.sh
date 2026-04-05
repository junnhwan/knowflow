#!/usr/bin/env bash
set -euo pipefail

BASE_URL="${KNOWFLOW_BASE_URL:-http://localhost:8080}"
USER_ID="${KNOWFLOW_USER_ID:-demo-user}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MARKDOWN_PATH="${1:-$SCRIPT_DIR/data/backend-interview-notes.md}"
QUESTION="${KNOWFLOW_QUESTION:-总结一下 KnowFlow 的亮点}"
KNOWLEDGE_CONTENT="${KNOWFLOW_KNOWLEDGE_CONTENT:-KnowFlow 将知识反写落到结构化 knowledge_entries，再触发受影响范围的索引更新，而不是直接把内容追加到 markdown 文件。}"

if ! command -v jq >/dev/null 2>&1; then
  echo "需要先安装 jq 才能运行该脚本" >&2
  exit 1
fi

if [[ ! -f "$MARKDOWN_PATH" ]]; then
  echo "测试文档不存在: $MARKDOWN_PATH" >&2
  exit 1
fi

echo ""
echo "==> 1. 健康检查"
curl -sS "$BASE_URL/healthz"
echo ""

echo ""
echo "==> 2. 上传文档"
UPLOAD_RESPONSE="$(curl -sS -X POST "$BASE_URL/api/kb/documents" \
  -H "X-User-ID: $USER_ID" \
  -F "file=@$MARKDOWN_PATH")"
echo "$UPLOAD_RESPONSE"
DOCUMENT_ID="$(echo "$UPLOAD_RESPONSE" | jq -r '.DocumentID')"

echo ""
echo "==> 3. 普通问答"
QUERY_RESPONSE="$(curl -sS -X POST "$BASE_URL/api/chat/query" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"\",\"message\":\"$QUESTION\"}")"
echo "$QUERY_RESPONSE"
SESSION_ID="$(echo "$QUERY_RESPONSE" | jq -r '.session_id')"

echo ""
echo "==> 4. 查询会话列表"
curl -sS "$BASE_URL/api/chat/sessions" -H "X-User-ID: $USER_ID"
echo ""

echo ""
echo "==> 5. 查询会话消息"
curl -sS "$BASE_URL/api/chat/sessions/$SESSION_ID/messages" -H "X-User-ID: $USER_ID"
echo ""

echo ""
echo "==> 6. 知识反写"
curl -sS -X POST "$BASE_URL/api/kb/knowledge" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d "{\"session_id\":\"$SESSION_ID\",\"content\":\"$KNOWLEDGE_CONTENT\",\"source_type\":\"manual\"}"
echo ""

echo ""
echo "==> 7. 重建索引"
curl -sS -X POST "$BASE_URL/api/kb/reindex" \
  -H "X-User-ID: $USER_ID" \
  -H "Content-Type: application/json" \
  -d "{\"document_id\":\"$DOCUMENT_ID\"}"
echo ""

echo ""
echo "==> 8. 查看指标"
curl -sS "$BASE_URL/metrics" | grep "knowflow_" || true
echo ""

echo ""
echo "完成: document_id=$DOCUMENT_ID session_id=$SESSION_ID"
