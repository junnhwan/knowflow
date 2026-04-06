# KnowFlow

`KnowFlow` 是一个面向后端面试知识运营场景的 `Go + AI` 后端项目，主线围绕：

`文档摄取 -> 混合检索 -> Rerank -> 引用式回答 -> Redis 会话记忆 -> 知识反写 -> 索引热更新 -> 可观测性`

## 当前能力

- 文档摄取：支持 `md / txt`
- 混合检索：`PgVector` 语义召回 + `pg_trgm` 关键词召回 + `RRF` 融合，统一覆盖文档块与知识块
- 回答生成：引用式回答、无证据拒答、可切本地模式或真实模型模式
- 会话管理：Redis 最近窗口 + 历史摘要双层记忆
- 知识反写：`knowledge_entries` 结构化沉淀，并生成 `knowledge_chunks` 检索索引
- 索引更新：支持文档和知识条目范围刷新
- 可观测性：结构化日志、Prometheus 指标
- 联调界面：内置 `/playground`
- 流式问答：`/api/chat/query/stream` 已接通真实 SSE 增量输出

## 技术栈

- Go 1.26
- Gin
- Eino
- Postgres / PgVector / pg_trgm
- Redis
- Prometheus

## 快速启动

### 1. 配置环境

复制 `.env.example` 为 `.env`，至少确认下面几项：

```env
HTTP_PORT=8080
POSTGRES_DSN=postgres://postgres:你的密码@你的主机:5432/knowflow?sslmode=disable
REDIS_ADDR=你的主机:6379
REDIS_PASSWORD=你的密码
REDIS_DB=0
MODEL_PROVIDER=local
EMBEDDING_DIMENSION=64
```

说明：

- `MODEL_PROVIDER=local` 时，可不依赖外部模型先跑通主链路
- 切真实模型时，设置：
  - `MODEL_PROVIDER`
  - `MODEL_BASE_URL`
  - `MODEL_API_KEY`
  - `MODEL_CHAT_NAME`

### 2. 启动服务

```bash
go run ./cmd/server
```

默认地址：

- 服务：`http://localhost:8080`
- 联调页：`http://localhost:8080/playground`
- 指标：`http://localhost:8080/metrics`

### 3. 最短验证路径

1. 打开 `/playground`
2. 点击“健康检查”
3. 上传一份 `md/txt` 文档
4. 先测普通问答
5. 再测 SSE 问答
6. 查看会话列表与历史消息
7. 写入一条知识反写内容并再次提问，确认命中新知识
8. 再测重建索引

## 重要接口

- `POST /api/kb/documents`
- `POST /api/chat/query`
- `POST /api/chat/query/stream`
- `GET /api/chat/sessions`
- `GET /api/chat/sessions/{session_id}/messages`
- `POST /api/kb/knowledge`
- `POST /api/kb/reindex`

## 当前定位

`KnowFlow V1` 的重点不是“框架功能展示”，而是把这些后端问题显式做出来：

- 检索质量控制
- 会话状态管理
- 结构化知识沉淀
- 热更新与降级策略
- AI 链路可观测性
