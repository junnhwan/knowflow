param(
    [string]$BaseUrl = $(if ($env:KNOWFLOW_BASE_URL) { $env:KNOWFLOW_BASE_URL } else { "http://localhost:8080" }),
    [string]$UserId = $(if ($env:KNOWFLOW_USER_ID) { $env:KNOWFLOW_USER_ID } else { "demo-user" }),
    [Parameter(Mandatory = $true)][string]$SessionId,
    [string]$Message = "继续解释一下 Redis 记忆压缩的设计"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$body = @{
    session_id = $SessionId
    message    = $Message
} | ConvertTo-Json -Compress

Write-Host "开始请求 SSE 流，请观察 data: 事件输出..." -ForegroundColor Cyan
& curl.exe -N -X POST "$BaseUrl/api/chat/query/stream" `
    -H "Content-Type: application/json" `
    -H "X-User-ID: $UserId" `
    -d $body
