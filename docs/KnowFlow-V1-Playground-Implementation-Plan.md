# KnowFlow V1 Playground Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 `KnowFlow` 增加一个由后端直接托管的 `/playground` 联调控制台，覆盖健康检查、文档摄取、问答、SSE、会话回查、知识反写、重建索引与调试视图。

**Architecture:** 继续沿用现有 Gin 路由与 handler 组织方式，在 `internal/transport/http/handler` 下新增 playground 静态页面处理器，并用 Go `embed` 打包 HTML/CSS/JS。页面只调用既有 API，不新增后端业务接口，保证低维护成本和后续可演进性。

**Tech Stack:** Go, Gin, embed, 原生 HTML/CSS/JavaScript, Go test

---

### Task 1: 固化测试与设计输入

**Files:**
- Modify: `internal/app/router_test.go`
- Create: `docs/KnowFlow-V1-Playground-Design.md`
- Create: `docs/KnowFlow-V1-Playground-Implementation-Plan.md`

- [ ] **Step 1: 保留 `/playground` 与静态资源路由测试**

确认测试覆盖：
- `GET /playground`
- `GET /playground/assets/playground.css`

- [ ] **Step 2: 运行定向测试确认红灯**

Run: `go test ./internal/app -run "TestNewRouter_Playground(Page|Assets)" -v`
Expected: `404` 失败，说明路由和资源尚未接入。

- [ ] **Step 3: 提交设计与计划文档**

Run:
```bash
git add docs/KnowFlow-V1-Playground-Design.md docs/KnowFlow-V1-Playground-Implementation-Plan.md internal/app/router_test.go
git commit -m "test: 增加 playground 路由测试与设计文档"
```

### Task 2: 接入后端托管的 playground 页面

**Files:**
- Create: `internal/transport/http/handler/playground_handler.go`
- Create: `internal/transport/http/handler/playground_assets/playground.html`
- Create: `internal/transport/http/handler/playground_assets/playground.css`
- Create: `internal/transport/http/handler/playground_assets/playground.js`
- Modify: `internal/app/router.go`

- [ ] **Step 1: 新增失败点以外的最小实现入口**

实现一个基于 `embed` 的静态资源处理器，至少能返回 HTML 和 CSS。

- [ ] **Step 2: 运行定向测试确认转绿**

Run: `go test ./internal/app -run "TestNewRouter_Playground(Page|Assets)" -v`
Expected: PASS

- [ ] **Step 3: 扩充页面能力**

补齐：
- 左侧控制栏与右侧结果区布局
- 健康检查、上传文档、普通问答、SSE、会话列表、消息历史、知识反写、重建索引、指标快照
- 原始响应与错误反馈展示

- [ ] **Step 4: 运行全量测试**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: 提交 playground 实现**

Run:
```bash
git add internal/app/router.go internal/transport/http/handler/playground_handler.go internal/transport/http/handler/playground_assets/playground.html internal/transport/http/handler/playground_assets/playground.css internal/transport/http/handler/playground_assets/playground.js
git commit -m "feat: 增加内置 playground 联调页"
```

### Task 3: 校验使用说明与验证清单

**Files:**
- Modify: `docs/KnowFlow-V1-Verification-Checklist.md`

- [ ] **Step 1: 补充 `/playground` 手工验证项**

把页面访问、联调链路和 SSE 展示加进验证清单。

- [ ] **Step 2: 提交文档更新**

Run:
```bash
git add docs/KnowFlow-V1-Verification-Checklist.md
git commit -m "docs: 更新 playground 验证清单"
```
