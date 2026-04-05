# KnowFlow V1 Task 02：数据库结构与仓储层实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 建立 `KnowFlow` 的核心表结构、Migration、领域对象和仓储层，为后续摄取、检索、会话、知识反写提供持久化基础。

**Architecture:** 这一阶段的重点是把“数据契约”固定下来。表结构、索引、领域结构体和仓储接口必须先明确，后面的摄取、检索和聊天主链路都依赖这里的边界。

**Tech Stack:** Go 1.24, PostgreSQL, pgx/v5, pgvector-go, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：[task-01-runtime-bootstrap.md](./task-01-runtime-bootstrap.md)

## 文件范围

**Files:**
- Create: `migrations/0001_enable_extensions.sql`
- Create: `migrations/0002_create_core_tables.sql`
- Create: `migrations/0003_create_indexes.sql`
- Create: `internal/domain/document/document.go`
- Create: `internal/domain/chat/session.go`
- Create: `internal/domain/chat/message.go`
- Create: `internal/domain/knowledge/entry.go`
- Create: `internal/platform/postgres/postgres.go`
- Create: `internal/platform/postgres/migrator.go`
- Create: `internal/repository/postgres/document_repository.go`
- Create: `internal/repository/postgres/chat_repository.go`
- Create: `internal/repository/postgres/knowledge_repository.go`
- Test: `internal/repository/postgres/document_repository_test.go`

## 阶段交付物

- 核心业务表结构
- `vector` 和 `pg_trgm` 扩展开启
- 基础领域结构体
- Postgres 连接与 Migration 执行器
- 文档、会话、知识条目的基础仓储实现

- [ ] **Step 1: 先写失败的仓储测试**

```go
func TestDocumentRepository_CreateDocument(t *testing.T) {
    repo := newTestDocumentRepository(t)
    doc := document.Document{
        ID:         "doc-1",
        UserID:     "demo-user",
        SourceName: "intro.md",
        Status:     "indexed",
    }

    err := repo.Create(context.Background(), doc)
    require.NoError(t, err)

    saved, err := repo.GetByID(context.Background(), "doc-1")
    require.NoError(t, err)
    require.Equal(t, "intro.md", saved.SourceName)
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/repository/postgres -run TestDocumentRepository_CreateDocument -v`
Expected: FAIL，因为表和仓储还没有实现。

- [ ] **Step 3: 编写 SQL Migration**

`0001_enable_extensions.sql` 至少开启：

- `vector`
- `pg_trgm`

`0002_create_core_tables.sql` 至少创建：

- `documents`
- `document_chunks`
- `sessions`
- `messages`
- `knowledge_entries`

`0003_create_indexes.sql` 至少创建：

- `document_chunks.embedding` 的向量索引
- 常用外键/查询字段的 BTree 索引
- `document_chunks.content` 的 trigram 索引

- [ ] **Step 4: 实现领域结构体**

为以下对象建立清晰的领域结构：

- 文档元信息
- Chunk 元信息
- 会话与消息
- 知识条目

- [ ] **Step 5: 实现 Postgres 基础设施**

在 `internal/platform/postgres` 中实现：

- 连接池初始化
- `Ping`
- 优雅关闭
- Migration 读取与按顺序执行

- [ ] **Step 6: 实现第一批仓储方法**

至少实现：

- `DocumentRepository.Create`
- `DocumentRepository.GetByID`
- `DocumentRepository.DeleteByID`
- `ChatRepository.CreateSession`
- `ChatRepository.AppendMessage`
- `KnowledgeRepository.Create`

- [ ] **Step 7: 重新运行仓储测试**

Run: `go test ./internal/repository/postgres -run TestDocumentRepository_CreateDocument -v`
Expected: PASS

- [ ] **Step 8: 跑完整个 Postgres 仓储包测试**

Run: `go test ./internal/repository/postgres -v`
Expected: PASS

- [ ] **Step 9: 提交这一阶段**

```bash
git add migrations internal/domain internal/platform/postgres internal/repository/postgres
git commit -m "feat: add core schema and postgres repositories"
```
