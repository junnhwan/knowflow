# KnowFlow V1 验证清单

更新时间：2026-04-05

## 自动化验证

- [x] `go test ./...`
- [x] 配置加载测试：`internal/config/config_test.go`
- [x] 健康检查路由测试：`internal/app/router_test.go`
- [x] Playground 页面与静态资源路由测试：`internal/app/router_test.go`
- [x] 文档摄取服务测试：`internal/service/ingestion/service_test.go`
- [x] 混合检索与降级测试：`internal/service/retrieval/service_test.go`
- [x] 会话记忆压缩测试：`internal/service/memory/service_test.go`
- [x] Chat 编排测试：`internal/service/chat/orchestrator_test.go`
- [x] Prometheus 指标暴露测试：`internal/platform/observability/metrics_test.go`
- [x] Postgres 文档仓储测试：`internal/repository/postgres/document_repository_test.go`
- [x] Redis memory store 测试：`internal/repository/redis/memory_store_test.go`

## 待本机依赖可用后的手工验证

- [ ] `docker compose -f deployments/docker-compose.yml up -d`
- [ ] `go run ./cmd/server`
- [ ] `GET /healthz` 返回 `200`
- [ ] `GET /playground` 页面可访问，且左控右显布局正常
- [ ] `POST /api/kb/documents` 上传 `md/txt` 成功并返回 `document_id`
- [ ] `POST /api/chat/query` 返回 `session_id + answer + citations + retrieval_meta`
- [ ] `POST /api/chat/query/stream` 能持续返回 SSE 分段内容与最终事件
- [ ] `GET /api/chat/sessions` 能查询会话列表
- [ ] `GET /api/chat/sessions/{session_id}/messages` 能查询消息历史
- [ ] `POST /api/kb/knowledge` 能落知识条目
- [ ] `POST /api/kb/reindex` 能触发指定文档重建索引
- [ ] `GET /metrics` 能看到 `knowflow_rag_hit_total`、`knowflow_tool_call_total` 等自定义指标
- [ ] 在 `/playground` 中完成一次上传文档 -> 普通问答 -> SSE 问答 -> 会话回查 -> 知识反写 -> 重建索引 的联调闭环
- [ ] `/playground` 调试抽屉能查看 raw response、retrieval_meta、tool_traces 与 metrics 快照

## 当前阻塞

- 本机 Docker daemon 当前不可用，执行 `docker compose -f deployments/docker-compose.yml up -d` 时返回：
  `open //./pipe/dockerDesktopLinuxEngine: The system cannot find the file specified`
- 因此当前回合已完成“代码编译 + 单元测试 + 仓储测试”验证，尚未完成真实容器依赖上的端到端手工演练。

## 建议下一步

1. 启动 Docker Desktop 或修复 Docker daemon。
2. 重新执行本清单中的手工验证部分。
3. 若本地要切到真实模型链路，配置 `MODEL_PROVIDER`、`MODEL_API_KEY`、`MODEL_BASE_URL` 和 `MODEL_CHAT_NAME`。
