# KnowFlow Project Context

## Project Positioning

- `KnowFlow` 的定位是“面向后端面试知识运营场景的 `Go + Eino` AI 后端”。
- `KnowFlow V1` 采用“简历型 MVP”路线，优先做深主链路，不追求平台化铺设。
- 项目主链路已经锁定为：
  `文档摄取 -> 混合检索 -> Rerank -> 引用式回答 -> Redis 会话记忆 -> 知识反写 -> 索引热更新 -> 可观测性`
- 项目对外叙事统一围绕后端面试知识库、项目复盘沉淀、八股问答和知识运营闭环展开，不写成泛聊天平台。

## Reference Projects

- `D:\dev\go_proj\GopherAI\GopherAI-v2`
  - 主要借鉴：Go 项目结构、Eino 接入方式、聊天接口、SSE 流式响应、基础 RAG 组织方式。
- `D:\dev\learn_proj\ByteCoach`
  - 主要借鉴：递归切块、查询预处理、Rerank、Redis 会话记忆压缩、知识反写、可观测性亮点。
- `D:\dev\learn_proj\ragent`
  - 主要借鉴：检索通道接口、后处理器链、Memory/Trace 服务拆分等工程边界。

## Key Judgement

- `ByteCoach` 当前更明确落地的是“向量召回 + 查询预处理 + Rerank + Redis 记忆 + 反写思路”，并不是现成的“双路召回 + RRF”代码模板。
- 因此 `KnowFlow` 的“PgVector 语义召回 + pg_trgm 关键词召回 + RRF 融合”应视为项目自身增强设计。
- `GopherAI-v2` 更适合作为 Go/Eino 技术底座参考，不适合作为简历亮点叙事模板。
- `ragent` 只参考工程拆分模式，不把其意图树、多模型路由、后台管理、多线程池、限流排队等复杂能力纳入 `V1`。

## V1 Locked Scope

- 纯后端优先，只提供 HTTP API 与 SSE 流式输出。
- 文档类型先只支持 `txt / md`。
- 主存储使用 `Postgres + PgVector`，会话与缓存使用 `Redis`。
- Agent 仅保留轻量工具调用，不做复杂多 Agent 编排。
- 核心能力聚焦为：
  - 高质量 RAG
  - Redis 会话记忆
  - 知识反写热更新
  - 可观测性
- 固定后端 API：
  - `POST /api/kb/documents`
  - `POST /api/chat/query`
  - `POST /api/chat/query/stream`
  - `GET /api/chat/sessions`
  - `GET /api/chat/sessions/{session_id}/messages`
  - `POST /api/kb/knowledge`
  - `POST /api/kb/reindex`

## V1 Non-Goals

- 不做前端重构。
- 不做 PDF/Word/网页抓取等复杂摄取。
- 不做多租户、复杂权限、审批流。
- 不把 Guardrail、安全体系、MCP 生态扩展作为 V1 主卖点。
- 不直接照搬参考项目的“企业级平台”叙事。
- 不引入意图树、多模型路由、复杂限流队列、后台管理等 `ragent` 风格的大体量系统能力。

## Resume Positioning

- 简历里少写框架本身，多写自己实现的系统链路。
- 重点强调检索质量、会话状态管理、索引热更新、降级策略、可观测性。
- 默认叙事为“面试知识运营后端”，突出知识沉淀闭环，不写成“通用 AI 平台”。
- 没有真实数据支撑前，不写固定收益数字。

## Planning Docs

- 设计记录：`docs/KnowFlow-V1-Design.md`
- 总执行计划索引：`docs/KnowFlow-V1-Implementation-Plan.md`
- 分任务执行计划目录：`docs/knowflow-v1-tasks/`
