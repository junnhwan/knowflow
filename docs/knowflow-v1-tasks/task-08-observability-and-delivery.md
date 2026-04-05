# KnowFlow V1 Task 08：可观测性与交付收尾实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 补齐结构化日志、Prometheus 指标、验证清单和简历说明文档，并完成端到端可演示验证。

**Architecture:** 这一阶段不是“顺手埋点”，而是把前面已经实现的稳定能力变成可观察、可验证、可交付的系统。日志和指标要挂在稳定边界上，验证清单和简历文案必须严格对应真实实现。

**Tech Stack:** Go 1.24, Prometheus client_golang, Gin, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：
  - [task-03-document-ingestion.md](./task-03-document-ingestion.md)
  - [task-04-retrieval-pipeline.md](./task-04-retrieval-pipeline.md)
  - [task-05-redis-session-memory.md](./task-05-redis-session-memory.md)
  - [task-06-chat-orchestration-and-sse.md](./task-06-chat-orchestration-and-sse.md)
  - [task-07-tool-agent-and-knowledge-writeback.md](./task-07-tool-agent-and-knowledge-writeback.md)

## 文件范围

**Files:**
- Create: `internal/platform/observability/logger.go`
- Create: `internal/platform/observability/metrics.go`
- Create: `docs/KnowFlow-V1-Verification-Checklist.md`
- Create: `docs/KnowFlow-V1-Resume-Notes.md`
- Modify: `internal/service/retrieval/service.go`
- Modify: `internal/service/chat/orchestrator.go`
- Modify: `internal/service/tools/registry.go`

## 阶段交付物

- 结构化日志工具
- 自定义 Prometheus 指标
- `/metrics` 暴露
- 验证清单
- 对应真实实现的简历说明

- [ ] **Step 1: 先写失败的指标测试**

```go
func TestMetricsRegistry_RegistersRAGCounters(t *testing.T) {
    reg := newMetricsRegistry()
    reg.RecordRAGHit("demo-user", "s-1")

    body := reg.Expose()
    require.Contains(t, body, "knowflow_rag_hit_total")
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/platform/observability -run TestMetricsRegistry_RegistersRAGCounters -v`
Expected: FAIL，因为指标模块尚未实现。

- [ ] **Step 3: 实现指标注册表**

`metrics.go` 至少暴露以下指标：

- `knowflow_llm_request_total`
- `knowflow_llm_latency_seconds`
- `knowflow_rag_hit_total`
- `knowflow_rag_miss_total`
- `knowflow_rerank_fallback_total`
- `knowflow_tool_call_total`
- `knowflow_tool_call_fail_total`

- [ ] **Step 4: 实现结构化日志模块**

`logger.go` 至少便于输出：

- `request_id`
- `user_id`
- `session_id`
- `document_id`
- `tool_name`
- `fallback_reason`

- [ ] **Step 5: 给核心服务补埋点**

至少在这些位置增加日志/指标：

- retrieval service
- chat orchestrator
- tool registry
- ingestion service

- [ ] **Step 6: 增加 `/metrics` 与交付文档**

新增：

- `/metrics` 路由
- `docs/KnowFlow-V1-Verification-Checklist.md`
- `docs/KnowFlow-V1-Resume-Notes.md`

其中验证清单至少覆盖：

- 文档摄取检查
- 检索链路检查
- 记忆系统检查
- 工具链路检查
- SSE 检查
- 指标检查

- [ ] **Step 7: 跑全量验证**

Run:

- `go test ./...`
- `curl http://localhost:8080/metrics`
- `curl http://localhost:8080/healthz`

Expected:

- 所有测试 PASS
- `/metrics` 能暴露自定义指标
- `/healthz` 返回 `200`

- [ ] **Step 8: 做端到端演练**

至少完整走一遍：

1. 上传 markdown 文档
2. 提问一个基于知识库的问题
3. 检查返回的 citations
4. 连续追问直到触发记忆压缩
5. 新增一条知识反写
6. 重建索引后再次提问
7. 检查 `/metrics` 中的 hit / fallback / tool 指标

- [ ] **Step 9: 提交这一阶段**

```bash
git add internal/platform/observability internal/service docs/KnowFlow-V1-Verification-Checklist.md docs/KnowFlow-V1-Resume-Notes.md
git commit -m "feat: add observability and verification docs"
```
