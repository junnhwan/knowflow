# KnowFlow V1 知识反写检索闭环 Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 `POST /api/kb/knowledge` 写入的知识条目能够在下一次检索和问答中真实命中，形成“知识落库 -> 建索引 -> 查询命中 -> 局部刷新”的闭环。

**Architecture:** 保持 `knowledge_entries` 作为业务沉淀主表，不把它直接当作检索表使用；新增 `knowledge_chunks` 作为知识条目的检索索引层，并让检索主链路同时搜索 `document_chunks` 和 `knowledge_chunks`。知识反写成功后立即执行切块、向量化和索引写入，`reindex` 再补齐按知识条目范围刷新的能力。

**Tech Stack:** Go, Gin, pgx/v5, PostgreSQL, PgVector, pg_trgm, Redis, Testify

---

## 范围说明

这份计划只解决“知识反写进入检索闭环”这一件事，不同时处理：

- 真实远程 `embedding`
- 真实远程 `rerank`
- Guardrail
- 自动化 reload/watch

这些放到下一轮计划处理，避免这轮范围失控。

## 文件结构

### 预计新增

- `migrations/0004_create_knowledge_chunks.sql`
- `internal/domain/knowledge/chunk.go`
- `internal/repository/postgres/knowledge_repository_test.go`

### 预计修改

- `internal/repository/postgres/knowledge_repository.go`
- `internal/service/tools/upsert_knowledge.go`
- `internal/service/retrieval/service.go`
- `internal/service/retrieval/service_test.go`
- `internal/service/tools/retrieve_knowledge.go`
- `internal/service/tools/refresh_document_index.go`
- `internal/transport/http/handler/knowledge_handler.go`
- `internal/app/app.go`
- `docs/architecture/KnowFlow-V1-Design.md`
- `docs/testing/KnowFlow-V1-Verification-Checklist.md`

### 可能需要补充的测试文件

- `internal/transport/http/handler/knowledge_handler_test.go`
- `internal/service/tools/upsert_knowledge_test.go`

## Task 1: 增加知识索引表结构与仓储接口

**Files:**
- Create: `migrations/0004_create_knowledge_chunks.sql`
- Create: `internal/domain/knowledge/chunk.go`
- Modify: `internal/repository/postgres/knowledge_repository.go`
- Test: `internal/repository/postgres/knowledge_repository_test.go`

- [ ] **Step 1: 写仓储层失败测试**

新增测试覆盖：

- 创建 `knowledge_entry`
- 替换某个条目的 `knowledge_chunks`
- 按向量搜索知识块
- 按关键词搜索知识块
- 删除知识条目对应知识块后不可再命中

- [ ] **Step 2: 运行仓储测试，确认失败**

Run:

```powershell
go test ./internal/repository/postgres -run Knowledge -count=1
```

Expected:

- 因缺少 `knowledge_chunks` 表或相关方法而失败

- [ ] **Step 3: 补 SQL migration 和仓储实现**

实现要点：

- `knowledge_chunks` 必须独立于 `knowledge_entries`
- 保留 `knowledge_entry_id`
- 支持 `embedding`
- 支持 `pg_trgm` 关键词检索
- 支持按条目删除和替换索引块

- [ ] **Step 4: 再次运行仓储测试，确认通过**

Run:

```powershell
go test ./internal/repository/postgres -run Knowledge -count=1
```

Expected:

- PASS

- [ ] **Step 5: 建议提交**

```bash
git add migrations/0004_create_knowledge_chunks.sql internal/domain/knowledge/chunk.go internal/repository/postgres/knowledge_repository.go internal/repository/postgres/knowledge_repository_test.go
git commit -m "feat: 增加知识条目检索索引表"
```

## Task 2: 补知识反写后的切块与建索引服务

**Files:**
- Modify: `internal/service/tools/upsert_knowledge.go`
- Modify: `internal/app/app.go`
- Test: `internal/service/tools/upsert_knowledge_test.go`

- [ ] **Step 1: 写工具层失败测试**

新增测试覆盖：

- `upsert_knowledge` 成功时同时写 `knowledge_entries` 和 `knowledge_chunks`
- `content` 为空时报错
- 建索引失败时返回错误，不伪装成成功
- 成功返回中包含条目 ID、状态、块数量

- [ ] **Step 2: 运行工具测试，确认失败**

Run:

```powershell
go test ./internal/service/tools -run UpsertKnowledge -count=1
```

Expected:

- 因当前工具只落 `knowledge_entries` 而失败

- [ ] **Step 3: 实现最小闭环**

实现要点：

- `upsert_knowledge` 内部不再只是 `Create(entry)`
- 需要新增知识切块、向量化、索引替换流程
- 条目状态从 `pending_index` 调整为更准确的状态流转
- 尽量保持工具输入结构不破坏现有接口

- [ ] **Step 4: 运行工具测试，确认通过**

Run:

```powershell
go test ./internal/service/tools -run UpsertKnowledge -count=1
```

Expected:

- PASS

- [ ] **Step 5: 建议提交**

```bash
git add internal/service/tools/upsert_knowledge.go internal/service/tools/upsert_knowledge_test.go internal/app/app.go
git commit -m "feat: 打通知识反写建索引闭环"
```

## Task 3: 让检索主链路同时命中文档块和知识块

**Files:**
- Modify: `internal/service/retrieval/service.go`
- Modify: `internal/service/tools/retrieve_knowledge.go`
- Test: `internal/service/retrieval/service_test.go`

- [ ] **Step 1: 写检索层失败测试**

新增测试覆盖：

- 同时存在文档块和知识块时，两路都可参与融合
- 仅知识块命中时也能返回引用和命中元信息
- `Rerank` 失败时知识块仍能走融合粗排回退
- 无证据时仍维持拒答语义

- [ ] **Step 2: 运行检索测试，确认失败**

Run:

```powershell
go test ./internal/service/retrieval -run Retrieve -count=1
```

Expected:

- 因当前检索只查 `document_chunks` 而失败

- [ ] **Step 3: 实现检索融合扩展**

实现要点：

- 不新增新的对外 API
- 保持 `RetrieveRequest/Result` 结构尽量稳定
- 引用里要能区分来源是文档还是知识条目
- 命中统计要把知识块纳入主链路

- [ ] **Step 4: 运行检索测试，确认通过**

Run:

```powershell
go test ./internal/service/retrieval -run Retrieve -count=1
```

Expected:

- PASS

- [ ] **Step 5: 建议提交**

```bash
git add internal/service/retrieval/service.go internal/service/retrieval/service_test.go internal/service/tools/retrieve_knowledge.go
git commit -m "feat: 支持知识条目参与混合检索"
```

## Task 4: 补局部刷新与 HTTP 层验证

**Files:**
- Modify: `internal/service/tools/refresh_document_index.go`
- Modify: `internal/transport/http/handler/knowledge_handler.go`
- Test: `internal/transport/http/handler/knowledge_handler_test.go`

- [ ] **Step 1: 写 Handler/刷新失败测试**

新增测试覆盖：

- `POST /api/kb/knowledge` 成功返回索引结果
- `POST /api/kb/reindex` 支持按 `document_id` 刷新
- `POST /api/kb/reindex` 支持按 `knowledge_entry_id` 刷新
- 缺失关键参数时返回 `400`

- [ ] **Step 2: 运行 Handler 测试，确认失败**

Run:

```powershell
go test ./internal/transport/http/handler -run Knowledge -count=1
```

Expected:

- 因当前 `reindex` 只认文档、返回信息不完整而失败

- [ ] **Step 3: 实现局部刷新增强**

实现要点：

- 保持 `/api/kb/reindex` 路径不变
- 通过请求体识别刷新目标
- 明确文档刷新和知识条目刷新两种路径
- 响应里返回目标类型、目标 ID、块数量、状态

- [ ] **Step 4: 运行 Handler 测试，确认通过**

Run:

```powershell
go test ./internal/transport/http/handler -run Knowledge -count=1
```

Expected:

- PASS

- [ ] **Step 5: 建议提交**

```bash
git add internal/service/tools/refresh_document_index.go internal/transport/http/handler/knowledge_handler.go internal/transport/http/handler/knowledge_handler_test.go
git commit -m "feat: 增强知识索引局部刷新能力"
```

## Task 5: 回归验证与文档同步

**Files:**
- Modify: `docs/architecture/KnowFlow-V1-Design.md`
- Modify: `docs/testing/KnowFlow-V1-Verification-Checklist.md`
- Modify: `README.md`

- [ ] **Step 1: 更新设计与测试文档**

补充内容：

- `knowledge_entries` 与 `knowledge_chunks` 的职责边界
- 知识反写后的索引流程
- 新的回归验证步骤

- [ ] **Step 2: 运行核心回归测试**

Run:

```powershell
go test ./internal/service/tools ./internal/service/retrieval ./internal/repository/postgres ./internal/transport/http/handler -count=1
```

Expected:

- PASS

- [ ] **Step 3: 运行更大范围回归**

Run:

```powershell
go test ./... -count=1
```

Expected:

- PASS

- [ ] **Step 4: 手工联调验证**

验证顺序：

1. 上传一份面试知识文档
2. 发起一次问答，确认命中文档块
3. 调用 `POST /api/kb/knowledge` 新增知识条目
4. 再次发起问答，确认命中新知识
5. 调用 `POST /api/kb/reindex`，分别验证文档与知识条目刷新
6. 检查 `/playground` 与 `/metrics`

- [ ] **Step 5: 建议提交**

```bash
git add docs/architecture/KnowFlow-V1-Design.md docs/testing/KnowFlow-V1-Verification-Checklist.md README.md
git commit -m "docs: 更新知识反写检索闭环说明"
```

## 完成定义

满足以下条件才算这轮任务完成：

- `POST /api/kb/knowledge` 成功后生成知识检索块
- 下一次查询可直接命中新写入知识
- 检索结果能区分文档来源和知识来源
- `/api/kb/reindex` 同时支持文档和知识条目范围刷新
- 相关单测和回归测试通过
- 文档已更新，能支撑后续演示和简历表达

## 风险提醒

- 不要把 `knowledge_entries` 直接当检索表使用，否则会重新把业务层和索引层混起来。
- 不要顺手把真实远程 `embedding/rerank` 一起塞进这一轮，否则测试面会快速失控。
- 如果 `RetrieveResult` 结构变化过大，要优先检查 `/playground`、`curl` 脚本和现有测试是否受影响。

## 后续衔接

这轮完成后，下一轮优先进入：

1. 真实 `embedding` 适配器
2. 真实 `rerank` 适配器
3. 基础 Guardrail
4. 自动 reload/watch

Plan complete and saved to `docs/plans/KnowFlow-V1-Knowledge-Writeback-Closure-Implementation-Plan.md`. Ready to execute?
