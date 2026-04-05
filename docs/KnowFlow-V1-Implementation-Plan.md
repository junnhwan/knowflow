# KnowFlow V1 实施总计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建一个面向后端面试知识运营场景的 `Go + Eino` AI 后端 MVP，覆盖文档摄取、混合检索、Rerank、Redis 会话记忆、知识反写与可观测性。

**Architecture:** 这份文档只负责做“总索引”和“执行契约”，不再承载每个阶段的全部细节。真正的落地步骤拆到独立的 `Task` 文档里，这样后续修改某一阶段时可以精确到单个文件，不会把整份总计划改乱。

**Tech Stack:** Go 1.24, Gin, pgx/v5, pgvector-go, go-redis/v9, CloudWeGo Eino, Prometheus client_golang, Docker Compose, Testify

---

## 使用方式

- 先看 [KnowFlow-V1-Design.md](./KnowFlow-V1-Design.md)，理解项目定位、边界和设计原因。
- 再看这份总计划，确认整体执行顺序、里程碑和完成标准。
- 真正执行时，直接进入对应的 `Task` 子计划文档。
- 如果后续只修改某一个阶段，优先改对应 `Task` 文档；只有影响全局顺序、范围或交付定义时，才回改这份总计划。

## 全局原则

- `V1` 只做纯后端，不引入登录鉴权复杂度。
- `user_id` 虽然在实现中保留，但 `V1` 交付范围仍然按单用户知识库场景推进。
- 文档类型只支持 `txt / md`。
- 检索质量优先于“功能看起来很多”。
- 检索链路和记忆链路都必须有明确降级策略。
- 尽量用本地接口封装外部框架能力，避免业务代码被框架类型污染。
- 计划拆分遵循“总计划 + 分 Task 文档”的方式，方便后续精确调整。
- Demo 数据、Prompt、样例问答和知识反写示例统一围绕后端面试知识运营场景。
- `KnowFlow` 的混合检索属于项目自身增强设计，不按 `ByteCoach` 源码做原样复刻。

## 文档结构

### 核心文档

- 设计记录：[KnowFlow-V1-Design.md](./KnowFlow-V1-Design.md)
- 旧版计划草稿：[KnowFlow-V1-Plan.md](./KnowFlow-V1-Plan.md)
- 总实施计划：[KnowFlow-V1-Implementation-Plan.md](./KnowFlow-V1-Implementation-Plan.md)

### 分任务子计划

1. 项目底座与运行骨架：
   [task-01-runtime-bootstrap.md](./knowflow-v1-tasks/task-01-runtime-bootstrap.md)
2. 数据库结构与仓储层：
   [task-02-schema-and-repositories.md](./knowflow-v1-tasks/task-02-schema-and-repositories.md)
3. 文档摄取与建索引：
   [task-03-document-ingestion.md](./knowflow-v1-tasks/task-03-document-ingestion.md)
4. 混合检索与 Rerank：
   [task-04-retrieval-pipeline.md](./knowflow-v1-tasks/task-04-retrieval-pipeline.md)
5. Redis 会话记忆：
   [task-05-redis-session-memory.md](./knowflow-v1-tasks/task-05-redis-session-memory.md)
6. Chat 编排与 SSE：
   [task-06-chat-orchestration-and-sse.md](./knowflow-v1-tasks/task-06-chat-orchestration-and-sse.md)
7. 工具层与知识反写：
   [task-07-tool-agent-and-knowledge-writeback.md](./knowflow-v1-tasks/task-07-tool-agent-and-knowledge-writeback.md)
8. 可观测性与交付收尾：
   [task-08-observability-and-delivery.md](./knowflow-v1-tasks/task-08-observability-and-delivery.md)

## 里程碑

- `M1`：本地运行底座可用，配置、服务启动、基础路由打通
- `M2`：文档摄取、切块、向量入库可用
- `M3`：检索链路可返回引用，具备融合与 Rerank 降级
- `M4`：Redis 会话记忆和聊天主链路打通，支持 SSE
- `M5`：知识反写、重建索引、指标监控、验证清单全部完成

## 执行顺序

必须严格按照以下顺序执行：

1. [Task 01](./knowflow-v1-tasks/task-01-runtime-bootstrap.md)
2. [Task 02](./knowflow-v1-tasks/task-02-schema-and-repositories.md)
3. [Task 03](./knowflow-v1-tasks/task-03-document-ingestion.md)
4. [Task 04](./knowflow-v1-tasks/task-04-retrieval-pipeline.md)
5. [Task 05](./knowflow-v1-tasks/task-05-redis-session-memory.md)
6. [Task 06](./knowflow-v1-tasks/task-06-chat-orchestration-and-sse.md)
7. [Task 07](./knowflow-v1-tasks/task-07-tool-agent-and-knowledge-writeback.md)
8. [Task 08](./knowflow-v1-tasks/task-08-observability-and-delivery.md)

不要在 `Task 03 / 04 / 05 / 06` 主链路未稳定前，就提前做工具层、指标埋点或简历文案润色。

## 各 Task 摘要

### Task 01：项目底座与运行骨架

交付项目最小可运行壳：

- 配置读取
- 应用装配
- HTTP 服务启动
- Docker Compose
- Makefile

### Task 02：数据库结构与仓储层

交付持久化基础：

- SQL migration
- `PgVector` / `pg_trgm`
- 核心表结构
- 仓储接口与基础实现

### Task 03：文档摄取与建索引

交付知识入库入口：

- 文件校验
- 文本规范化
- 递归切块
- Embedding 生成
- Chunk 落库

### Task 04：混合检索与 Rerank

交付第一主亮点：

- 查询预处理
- 向量召回
- 关键词召回
- `RRF` 融合
- Rerank
- 引用返回
- 拒答与降级

### Task 05：Redis 会话记忆

交付第二主亮点：

- 会话隔离
- 最近窗口
- 历史摘要
- TTL
- 分布式锁
- 并发写降级

### Task 06：Chat 编排与 SSE

交付用户主链路：

- 请求上下文
- 日志中间件
- 问答编排
- 会话持久化
- SSE 输出

### Task 07：工具层与知识反写

交付第三主亮点：

- 工具注册表
- 知识反写
- 索引重建
- 工具调用轨迹
- 工具层只保留 `retrieve_knowledge`、`upsert_knowledge`、`refresh_document_index`

### Task 08：可观测性与交付收尾

交付第四主亮点与验收材料：

- Prometheus 指标
- 结构化日志
- 验证清单
- 简历说明文档

## 计划修改规则

- 只影响单个阶段的变更：优先修改对应 `Task` 文档。
- 影响阶段依赖关系的变更：同时修改相关 `Task` 文档和这份总计划。
- 影响项目定位、范围或完成标准的变更：同时修改
  - 这份总计划
  - [KnowFlow-V1-Design.md](./KnowFlow-V1-Design.md)
  - `context/knowflow-project-context.md`

## 完成定义

只有满足以下条件，`KnowFlow V1` 才算进入可演示状态：

- 一条命令可启动本地运行环境
- 文档上传可完成 `txt / md` 入库与向量索引
- 问答接口返回 `answer + citations + retrieval_meta`
- Rerank 失败时能正确降级
- Redis 会话记忆能维持上下文并压缩长会话
- 知识反写能落业务表并触发重建索引
- `/metrics` 能暴露自定义指标
- 验证清单可完整走通，无需临时补救
- Demo 叙事和样例数据能够稳定支撑“后端面试知识运营后端”的项目介绍

## 执行提醒

- 虽然 `V1` 按单用户推进，但 `user_id` 在表结构、Redis key、日志和指标里都要保留。
- 一旦实现开始偏向“AI 平台大而全叙事”，就回到主线：
  `文档摄取 -> 检索 -> 记忆 -> 反写 -> 可观测性`
- 如果被框架细节卡住，优先补一层本地适配器，不要让业务层直接依赖外部框架细节。
