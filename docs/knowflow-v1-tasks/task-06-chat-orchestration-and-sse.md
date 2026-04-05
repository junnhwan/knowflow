# KnowFlow V1 Task 06：Chat 编排与 SSE 实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将会话创建、记忆加载、检索增强、回答生成、消息持久化和 SSE 输出串成完整的聊天主链路。

**Architecture:** 这一阶段负责把前面已经完成的子系统真正串起来。编排层应保持“薄且明确”：先建/取 session，再取 memory，再做 retrieval，再决定 grounded answer 或拒答，最后落消息和返回结果。

**Tech Stack:** Go 1.24, Gin, CloudWeGo Eino, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：
  - [task-02-schema-and-repositories.md](./task-02-schema-and-repositories.md)
  - [task-04-retrieval-pipeline.md](./task-04-retrieval-pipeline.md)
  - [task-05-redis-session-memory.md](./task-05-redis-session-memory.md)

## 文件范围

**Files:**
- Create: `internal/service/chat/orchestrator.go`
- Create: `internal/service/chat/stream.go`
- Create: `internal/transport/http/middleware/request_context.go`
- Create: `internal/transport/http/middleware/logging.go`
- Create: `internal/transport/http/handler/chat_handler.go`
- Test: `internal/service/chat/orchestrator_test.go`

## 阶段交付物

- 请求上下文注入
- 日志中间件
- 标准 JSON 问答接口
- SSE 输出接口
- 无依据时明确拒答

- [ ] **Step 1: 先写失败的 chat 编排测试**

```go
func TestOrchestrator_QueryReturnsAnswerAndCitations(t *testing.T) {
    orch := newTestOrchestrator(t)
    resp, err := orch.Query(context.Background(), QueryRequest{
        UserID:    "demo-user",
        SessionID: "s-1",
        Message:   "总结一下 KnowFlow 的亮点",
    })

    require.NoError(t, err)
    require.NotEmpty(t, resp.Answer)
    require.NotEmpty(t, resp.Citations)
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/service/chat -run TestOrchestrator_QueryReturnsAnswerAndCitations -v`
Expected: FAIL，因为编排层还没有实现。

- [ ] **Step 3: 实现请求上下文中间件**

`request_context.go` 负责：

- 注入 `request_id`
- 从请求头读取 `X-User-ID`
- 缺省时回退为 `demo-user`
- 在有 `session_id` 时挂到上下文

- [ ] **Step 4: 实现日志中间件**

`logging.go` 至少记录：

- 路由路径
- `request_id`
- `user_id`
- `session_id`
- 延迟
- 状态码

- [ ] **Step 5: 实现 chat orchestrator**

`orchestrator.go` 负责：

- 无 session 时创建会话
- 加载 memory 状态
- 调用 retrieval pipeline
- 决定拒答还是走回答生成
- 组装模型输入
- 持久化用户消息和 AI 回复

- [ ] **Step 6: 实现 SSE 输出**

`stream.go` 和 `chat_handler.go` 负责：

- `POST /api/chat/query` 的标准 JSON 输出
- 长回答场景的 SSE 输出
- 最终事件中带回 citation 和检索元信息

- [ ] **Step 7: 补无命中拒答测试**

至少验证：

- `citations` 为空
- `retrieval_meta.hit` 为 `false`
- 返回内容明确表示证据不足

- [ ] **Step 8: 重新运行 chat 测试**

Run: `go test ./internal/service/chat -v`
Expected: PASS

- [ ] **Step 9: 做一次接口手工验证**

Run:

- `curl -X POST http://localhost:8080/api/chat/query -H "Content-Type: application/json" -H "X-User-ID: demo-user" -d "{\"message\":\"总结 KnowFlow 亮点\"}"`

Expected: 返回体中包含 `session_id`、`answer`、`citations`、`retrieval_meta`。

- [ ] **Step 10: 提交这一阶段**

```bash
git add internal/service/chat internal/transport/http/middleware internal/transport/http/handler/chat_handler.go
git commit -m "feat: add chat orchestration and sse query flow"
```
