# KnowFlow V1 Task 03：文档摄取与建索引实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现 `txt / md` 文档上传、文本规范化、递归切块、Embedding 生成和 Chunk 入库，打通知识入库入口。

**Architecture:** 这一阶段把“知识是如何进入系统的”固定下来。解析、切块、向量化和落库必须拆开实现，后续重建索引和知识反写都需要复用这一套能力。

**Tech Stack:** Go 1.24, Gin, CloudWeGo Eino, pgvector-go, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：
  - [task-01-runtime-bootstrap.md](./task-01-runtime-bootstrap.md)
  - [task-02-schema-and-repositories.md](./task-02-schema-and-repositories.md)

## 文件范围

**Files:**
- Create: `internal/service/ingestion/parser.go`
- Create: `internal/service/ingestion/splitter.go`
- Create: `internal/service/ingestion/service.go`
- Create: `internal/platform/llm/embedder.go`
- Create: `internal/transport/http/handler/document_handler.go`
- Test: `internal/service/ingestion/service_test.go`

## 阶段交付物

- `txt / md` 文件校验
- 文本规范化
- 递归切块器
- 批量向量化
- 文档上传接口

- [ ] **Step 1: 先写失败的摄取测试**

```go
func TestIngestionService_IngestMarkdownDocument(t *testing.T) {
    svc := newTestIngestionService(t)
    req := IngestRequest{
        UserID:     "demo-user",
        SourceName: "rag.md",
        Content:    "# Title\n\nKnowFlow keeps citations.",
    }

    result, err := svc.Ingest(context.Background(), req)
    require.NoError(t, err)
    require.NotEmpty(t, result.DocumentID)
    require.Greater(t, result.ChunkCount, 0)
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/service/ingestion -run TestIngestionService_IngestMarkdownDocument -v`
Expected: FAIL，因为摄取服务尚未实现。

- [ ] **Step 3: 实现解析器**

`parser.go` 负责：

- 仅接受 `.txt` 和 `.md`
- 统一换行符
- 合并重复空行
- 保留标题和段落边界

- [ ] **Step 4: 实现递归切块器**

`splitter.go` 负责：

- 优先按标题、段落、句子切分
- 切不动时按字符长度回退
- 默认参数：
  - `chunk_size = 700`
  - `chunk_overlap = 150`
- 生成稳定的 `chunk_index`

- [ ] **Step 5: 实现 Embedding 适配层**

在 `internal/platform/llm/embedder.go` 中定义接口：

```go
type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
}
```

再补一个基于 `Eino` 的具体实现。

- [ ] **Step 6: 实现摄取服务**

`service.go` 负责：

- 创建 `documents` 记录
- 执行解析与切块
- 批量生成 embedding
- 写入 `document_chunks`
- 返回文档 ID、Chunk 数量和状态

- [ ] **Step 7: 实现上传接口**

`document_handler.go` 至少支持：

- `POST /api/kb/documents`
- multipart 上传
- 原始文本调试模式
- 不支持文件类型时返回清晰错误

- [ ] **Step 8: 重新运行摄取测试**

Run: `go test ./internal/service/ingestion -run TestIngestionService_IngestMarkdownDocument -v`
Expected: PASS

- [ ] **Step 9: 补一个重传覆盖测试**

验证相同文档重新上传时：

- 旧 chunk 被替换
- 不会遗留过期向量数据

- [ ] **Step 10: 提交这一阶段**

```bash
git add internal/service/ingestion internal/platform/llm/embedder.go internal/transport/http/handler/document_handler.go
git commit -m "feat: add document ingestion and indexing"
```
