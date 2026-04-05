# KnowFlow V1 Task 07：工具层与知识反写实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现轻量工具层，包括知识检索工具、知识反写工具和索引重建工具，并暴露相应 HTTP 接口。

**Architecture:** 这一阶段不是追求“工具多”，而是把自然语言能力收敛成清晰的业务动作。工具层必须具备参数校验、超时控制、执行轨迹和失败回退，而不是把它做成不可追踪的黑盒。

**Tech Stack:** Go 1.24, Gin, CloudWeGo Eino, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：
  - [task-03-document-ingestion.md](./task-03-document-ingestion.md)
  - [task-04-retrieval-pipeline.md](./task-04-retrieval-pipeline.md)
  - [task-06-chat-orchestration-and-sse.md](./task-06-chat-orchestration-and-sse.md)

## 文件范围

**Files:**
- Create: `internal/service/tools/registry.go`
- Create: `internal/service/tools/retrieve_knowledge.go`
- Create: `internal/service/tools/upsert_knowledge.go`
- Create: `internal/service/tools/refresh_document_index.go`
- Create: `internal/transport/http/handler/knowledge_handler.go`
- Test: `internal/service/chat/orchestrator_test.go`

## 阶段交付物

- 工具注册表
- 知识反写能力
- 手动重建索引能力
- 工具调用轨迹
- 工具失败降级策略

- [ ] **Step 1: 先写失败的工具执行测试**

```go
func TestToolRegistry_UpsertKnowledgeCreatesEntry(t *testing.T) {
    reg := newTestToolRegistry(t)
    out, err := reg.Execute(context.Background(), "upsert_knowledge", map[string]any{
        "session_id": "s-1",
        "content":    "Go + Eino can be used to build a RAG backend.",
    })

    require.NoError(t, err)
    require.Equal(t, "success", out.Status)
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/service/... -run TestToolRegistry_UpsertKnowledgeCreatesEntry -v`
Expected: FAIL，因为工具注册表尚未实现。

- [ ] **Step 3: 实现工具注册表**

`registry.go` 负责：

- 注册 `retrieve_knowledge`
- 注册 `upsert_knowledge`
- 注册 `refresh_document_index`
- 在执行前统一做参数校验

- [ ] **Step 4: 实现三个工具**

`retrieve_knowledge.go`：

- 调用 retrieval service
- 输出工具可消费格式结果

`upsert_knowledge.go`：

- 写入 `knowledge_entries`
- 设置 `pending_index` 或 `indexed` 状态

`refresh_document_index.go`：

- 删除指定文档旧 chunk
- 重新切块、重新向量化、重新建索引

- [ ] **Step 5: 将工具链路接入 chat 编排层**

要求支持：

- 响应中带工具执行轨迹
- 工具失败时模型主链路仍可继续
- 每个工具有超时控制

- [ ] **Step 6: 增加知识接口**

`knowledge_handler.go` 至少支持：

- `POST /api/kb/knowledge`
- `POST /api/kb/reindex`

- [ ] **Step 7: 重新运行工具与 chat 回归测试**

Run:

- `go test ./internal/service/... -v`
- `go test ./internal/transport/http/... -v`

Expected: PASS

- [ ] **Step 8: 做一次接口手工验证**

Run:

- `curl -X POST http://localhost:8080/api/kb/knowledge -H "Content-Type: application/json" -d "{\"session_id\":\"s-1\",\"content\":\"混合检索可以降低幻觉\"}"`
- `curl -X POST http://localhost:8080/api/kb/reindex -H "Content-Type: application/json" -d "{\"document_id\":\"doc-1\"}"`

Expected: 返回成功结果，或在参数不合法时返回清晰错误。

- [ ] **Step 9: 提交这一阶段**

```bash
git add internal/service/tools internal/transport/http/handler/knowledge_handler.go
git commit -m "feat: add knowledge write-back and reindex tools"
```
