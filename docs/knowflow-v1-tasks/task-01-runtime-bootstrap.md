# KnowFlow V1 Task 01：项目底座与运行骨架实施计划

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 搭建 `KnowFlow` 的最小可运行骨架，包括配置读取、应用装配、HTTP 服务启动和本地依赖编排。

**Architecture:** 这一阶段不碰业务主链路，只负责把“项目能稳定跑起来”这件事做好。配置、应用装配、路由注册和本地运行方式都要先固定下来，后续所有能力都在这个壳上增量推进。

**Tech Stack:** Go 1.24, Gin, Docker Compose, Testify

---

## 依赖关系

- 上游设计文档：[../KnowFlow-V1-Design.md](../KnowFlow-V1-Design.md)
- 总计划索引：[../KnowFlow-V1-Implementation-Plan.md](../KnowFlow-V1-Implementation-Plan.md)
- 前置依赖：无

## 文件范围

**Files:**
- Create: `go.mod`
- Create: `.env.example`
- Create: `Makefile`
- Create: `deployments/docker-compose.yml`
- Create: `cmd/server/main.go`
- Create: `internal/app/app.go`
- Create: `internal/app/router.go`
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

## 阶段交付物

- 可读取环境变量的配置模块
- 可启动的 Gin 服务
- `GET /healthz` 健康检查接口
- 本地依赖编排文件
- 基础开发命令

- [ ] **Step 1: 先写失败的配置测试**

```go
func TestLoadConfigFromEnv(t *testing.T) {
    t.Setenv("HTTP_PORT", "8080")
    t.Setenv("POSTGRES_DSN", "postgres://knowflow:knowflow@localhost:5432/knowflow?sslmode=disable")
    t.Setenv("REDIS_ADDR", "localhost:6379")

    cfg, err := Load()
    require.NoError(t, err)
    require.Equal(t, "8080", cfg.HTTP.Port)
    require.Equal(t, "localhost:6379", cfg.Redis.Addr)
}
```

- [ ] **Step 2: 运行测试确认当前必然失败**

Run: `go test ./internal/config -run TestLoadConfigFromEnv -v`
Expected: FAIL，因为 `Load` 和配置结构体还不存在。

- [ ] **Step 3: 实现配置加载模块**

在 `internal/config/config.go` 中实现：

- `HTTP`、`Postgres`、`Redis`、`Model`、`Retrieval`、`Memory`、`Observability` 配置结构
- 环境变量读取和本地默认值
- 对关键配置做校验，例如 DSN、端口、模型相关配置

- [ ] **Step 4: 搭建应用装配与服务入口**

实现：

- `cmd/server/main.go`：读取配置并启动应用
- `internal/app/app.go`：应用级依赖装配
- `internal/app/router.go`：注册基础路由与健康检查

- [ ] **Step 5: 补齐本地运行文件**

实现：

- `.env.example`
- `deployments/docker-compose.yml`，至少包含：
  - `postgres + pgvector`
  - `redis`
  - `prometheus`
  - `grafana`
- `Makefile`，至少包含：
  - `make up`
  - `make down`
  - `make run`
  - `make test`

- [ ] **Step 6: 重新运行配置测试**

Run: `go test ./internal/config -run TestLoadConfigFromEnv -v`
Expected: PASS

- [ ] **Step 7: 做一次服务启动冒烟验证**

Run:

- `docker compose -f deployments/docker-compose.yml up -d`
- `go run ./cmd/server`

Expected: 服务正常启动，`GET /healthz` 返回 `200`。

- [ ] **Step 8: 提交这一阶段**

```bash
git add go.mod .env.example Makefile deployments/docker-compose.yml cmd/server/main.go internal/app internal/config
git commit -m "chore: bootstrap knowflow runtime"
```
