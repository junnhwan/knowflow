# KnowFlow V1 Task 05：Redis 会话记忆实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现基于 Redis 的会话记忆系统，支持会话隔离、最近消息窗口、历史摘要压缩、TTL 和并发写保护。

**Architecture:** 这一阶段要把“记忆”当成显式状态管理问题来做，而不是框架自动附带功能。最近消息和历史摘要要分开存，压缩触发、锁保护和失败降级都必须能单独测试。

**Tech Stack:** Go 1.24, go-redis/v9, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：
  - [task-01-runtime-bootstrap.md](./task-01-runtime-bootstrap.md)
  - [task-02-schema-and-repositories.md](./task-02-schema-and-repositories.md)

## 文件范围

**Files:**
- Create: `internal/platform/redis/redis.go`
- Create: `internal/repository/redis/memory_store.go`
- Create: `internal/service/memory/compressor.go`
- Create: `internal/service/memory/service.go`
- Test: `internal/repository/redis/memory_store_test.go`
- Test: `internal/service/memory/service_test.go`

## 阶段交付物

- Redis 客户端初始化
- 稳定的 memory key 设计
- 最近消息和摘要分层存储
- 压缩触发与降级
- 锁保护与并发写处理

- [ ] **Step 1: 先写失败的 Redis memory store 测试**

```go
func TestMemoryStore_SaveAndLoadRecentMessages(t *testing.T) {
    store := newTestMemoryStore(t)
    err := store.SaveRecent(context.Background(), "demo-user", "s-1", []MessageMemory{
        {Role: "user", Content: "你好"},
        {Role: "assistant", Content: "你好，我是 KnowFlow"},
    })
    require.NoError(t, err)

    got, err := store.LoadRecent(context.Background(), "demo-user", "s-1")
    require.NoError(t, err)
    require.Len(t, got, 2)
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/repository/redis -run TestMemoryStore_SaveAndLoadRecentMessages -v`
Expected: FAIL，因为 Redis memory store 还不存在。

- [ ] **Step 3: 实现 Redis 基础设施**

`internal/platform/redis/redis.go` 负责：

- 初始化 Redis 客户端
- 提供 `Ping`
- 支持优雅关闭

- [ ] **Step 4: 固定 Redis Key 设计**

统一采用类似以下 key：

- `knowflow:memory:{user_id}:{session_id}:recent`
- `knowflow:memory:{user_id}:{session_id}:summary`
- `knowflow:memory:lock:{user_id}:{session_id}`

- [ ] **Step 5: 实现 memory store**

`memory_store.go` 至少提供：

- `LoadRecent`
- `SaveRecent`
- `LoadSummary`
- `SaveSummary`
- `AcquireLock`
- `ReleaseLock`
- 成功写入后的 TTL 刷新

- [ ] **Step 6: 先写失败的压缩测试**

```go
func TestMemoryService_CompressesWhenThresholdExceeded(t *testing.T) {
    svc := newTestMemoryService(t)
    result, err := svc.Update(context.Background(), UpdateRequest{
        UserID:    "demo-user",
        SessionID: "s-1",
        Incoming:  buildConversation(24),
    })
    require.NoError(t, err)
    require.True(t, result.Compressed)
    require.NotEmpty(t, result.Summary)
}
```

- [ ] **Step 7: 实现压缩器**

`compressor.go` 负责：

- 用 rune/词数估算 token
- 保留最近 `N` 轮完整消息
- 将更早历史压缩成摘要块
- 摘要失败时回退为只保留最近消息

- [ ] **Step 8: 实现 memory service**

`service.go` 负责：

- 加载 recent + summary
- 判断是否需要压缩
- 完成更新写入
- 在锁获取失败时，降级为最近消息追加策略

- [ ] **Step 9: 重新运行所有记忆测试**

Run: `go test ./internal/repository/redis ./internal/service/memory -v`
Expected: PASS

- [ ] **Step 10: 提交这一阶段**

```bash
git add internal/platform/redis internal/repository/redis internal/service/memory
git commit -m "feat: add redis session memory with compression"
```
