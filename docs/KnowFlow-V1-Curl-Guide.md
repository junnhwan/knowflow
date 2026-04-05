# KnowFlow V1 curl 测试指南

## 脚本位置

- PowerShell 全流程: [knowflow-smoke.ps1](D:/dev/my_proj/knowflow/scripts/curl/knowflow-smoke.ps1)
- PowerShell SSE: [knowflow-stream.ps1](D:/dev/my_proj/knowflow/scripts/curl/knowflow-stream.ps1)
- bash 全流程: [knowflow-smoke.sh](D:/dev/my_proj/knowflow/scripts/curl/knowflow-smoke.sh)
- bash SSE: [knowflow-stream.sh](D:/dev/my_proj/knowflow/scripts/curl/knowflow-stream.sh)
- 测试文档: [backend-interview-notes.md](D:/dev/my_proj/knowflow/scripts/curl/data/backend-interview-notes.md)

## Windows / PowerShell

全流程测试：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\curl\knowflow-smoke.ps1
```

指定地址和用户：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\curl\knowflow-smoke.ps1 `
  -BaseUrl "http://你的服务地址:8080" `
  -UserId "demo-user"
```

单独测 SSE：

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\curl\knowflow-stream.ps1 `
  -BaseUrl "http://你的服务地址:8080" `
  -UserId "demo-user" `
  -SessionId "上一步返回的session_id"
```

## Linux / macOS / 云服务器 bash

全流程测试：

```bash
chmod +x ./scripts/curl/knowflow-smoke.sh
./scripts/curl/knowflow-smoke.sh
```

指定服务地址：

```bash
KNOWFLOW_BASE_URL="http://你的服务地址:8080" \
KNOWFLOW_USER_ID="demo-user" \
./scripts/curl/knowflow-smoke.sh
```

单独测 SSE：

```bash
chmod +x ./scripts/curl/knowflow-stream.sh
KNOWFLOW_BASE_URL="http://你的服务地址:8080" \
KNOWFLOW_USER_ID="demo-user" \
./scripts/curl/knowflow-stream.sh "上一步返回的session_id"
```

## 测试顺序

脚本内部默认顺序是：

1. `/healthz`
2. `/api/kb/documents`
3. `/api/chat/query`
4. `/api/chat/sessions`
5. `/api/chat/sessions/{session_id}/messages`
6. `/api/kb/knowledge`
7. `/api/kb/reindex`
8. `/metrics`

## 注意点

- PowerShell 版使用的是 `curl.exe`，避免和 `Invoke-WebRequest` 别名混淆。
- bash 版依赖 `jq` 解析 JSON；如果没有 `jq`，脚本会直接报错退出。
- 如果你还没配真实模型，这套脚本也可以跑，因为当前默认 `MODEL_PROVIDER=local`。
