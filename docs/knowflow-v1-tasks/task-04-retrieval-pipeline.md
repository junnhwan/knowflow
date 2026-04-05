# KnowFlow V1 Task 04：混合检索与 Rerank 实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现高质量检索链路，包括查询预处理、向量召回、关键词召回、融合、Rerank、引用组装和降级策略。

**Architecture:** 这一阶段是 `KnowFlow` 的第一主亮点。每个检索阶段都要保持独立，返回清晰的中间结果和状态，方便后面补监控、补日志、补降级，而不是把所有逻辑塞进一个大函数。

**Tech Stack:** Go 1.24, PostgreSQL, pgvector-go, CloudWeGo Eino, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：
  - [task-02-schema-and-repositories.md](./task-02-schema-and-repositories.md)
  - [task-03-document-ingestion.md](./task-03-document-ingestion.md)

## 文件范围

**Files:**
- Create: `internal/service/retrieval/preprocessor.go`
- Create: `internal/service/retrieval/vector_retriever.go`
- Create: `internal/service/retrieval/keyword_retriever.go`
- Create: `internal/service/retrieval/fusion.go`
- Create: `internal/service/retrieval/reranker.go`
- Create: `internal/service/retrieval/citation.go`
- Create: `internal/service/retrieval/service.go`
- Create: `internal/platform/llm/reranker.go`
- Test: `internal/service/retrieval/service_test.go`

## 阶段交付物

- 查询预处理
- 向量召回
- 关键词召回
- `RRF` 融合
- Rerank 精排
- 引用信息返回
- 检索失败和 Rerank 失败降级

- [ ] **Step 1: 先写失败的检索测试**

```go
func TestRetrievalService_HybridRetrieveReturnsCitations(t *testing.T) {
    svc := newTestRetrievalService(t)

    result, err := svc.Retrieve(context.Background(), RetrieveRequest{
        UserID: "demo-user",
        Query:  "KnowFlow 如何降低幻觉？",
        TopK:   5,
    })

    require.NoError(t, err)
    require.NotEmpty(t, result.Citations)
    require.False(t, result.Meta.Fallback)
}
```

- [ ] **Step 2: 运行测试确认当前失败**

Run: `go test ./internal/service/retrieval -run TestRetrievalService_HybridRetrieveReturnsCitations -v`
Expected: FAIL，因为检索服务尚未实现。

- [ ] **Step 3: 实现查询预处理**

`preprocessor.go` 负责：

- 去掉多余空白和噪音标点
- 统一全角/半角空格
- 去重部分停用词
- 输出：
  - 规范化查询串
  - 关键词 token 列表

- [ ] **Step 4: 实现向量召回**

`vector_retriever.go` 负责：

- 对查询做 embedding
- 基于 `pgvector` 相似度检索 chunk
- 返回候选结果和原始向量分数

- [ ] **Step 5: 实现关键词召回**

`keyword_retriever.go` 负责：

- 基于 `pg_trgm` 或关键词命中做查询
- 返回候选结果和关键词分数

- [ ] **Step 6: 实现融合策略**

`fusion.go` 使用 `RRF`：

```go
score = 1.0/(k + rankVector) + 1.0/(k + rankKeyword)
```

要求：

- 对 `chunk_id` 去重
- 相同分数时有稳定排序规则

- [ ] **Step 7: 实现 Rerank 与降级**

`reranker.go` 和 `internal/platform/llm/reranker.go` 负责：

- 对融合后的候选做精排
- 最终返回 `TopN = 5`
- Rerank 失败或返回空时，自动回退到融合结果

- [ ] **Step 8: 实现引用组装**

`citation.go` 输出字段至少包括：

- `document_id`
- `source_name`
- `chunk_id`
- `snippet`

- [ ] **Step 9: 实现检索服务总编排**

`service.go` 负责串起：

- preprocess
- vector retrieve
- keyword retrieve
- fusion
- rerank
- citation

并返回检索元信息，例如：

- 候选数量
- 是否发生降级
- 最终结果数量

- [ ] **Step 10: 重新运行检索测试**

Run: `go test ./internal/service/retrieval -v`
Expected: PASS

- [ ] **Step 11: 补降级场景测试**

至少覆盖：

- Rerank 超时
- 完全无检索命中
- 关键词召回失败但向量召回成功

- [ ] **Step 12: 提交这一阶段**

```bash
git add internal/service/retrieval internal/platform/llm/reranker.go
git commit -m "feat: add hybrid retrieval with rerank and citations"
```
