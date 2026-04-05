param(
    [string]$BaseUrl = $(if ($env:KNOWFLOW_BASE_URL) { $env:KNOWFLOW_BASE_URL } else { "http://localhost:8080" }),
    [string]$UserId = $(if ($env:KNOWFLOW_USER_ID) { $env:KNOWFLOW_USER_ID } else { "demo-user" }),
    [string]$MarkdownPath = $(Join-Path $PSScriptRoot "data\backend-interview-notes.md"),
    [string]$Question = "总结一下 KnowFlow 的亮点",
    [string]$KnowledgeContent = "KnowFlow 将知识反写落到结构化 knowledge_entries，再触发受影响范围的索引更新，而不是直接把内容追加到 markdown 文件。"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Write-Step {
    param([string]$Message)
    Write-Host ""
    Write-Host "==> $Message" -ForegroundColor Cyan
}

function Invoke-CurlText {
    param(
        [Parameter(Mandatory = $true)][string[]]$Arguments
    )

    $output = & curl.exe @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "curl 请求失败，参数: $($Arguments -join ' ')"
    }
    return $output
}

function Invoke-JsonRequest {
    param(
        [Parameter(Mandatory = $true)][string]$Method,
        [Parameter(Mandatory = $true)][string]$Url,
        [string]$Body = ""
    )

    $args = @("-sS", "-X", $Method, $Url, "-H", "X-User-ID: $UserId")
    if ($Body -ne "") {
        $args += @("-H", "Content-Type: application/json", "-d", $Body)
    }
    return Invoke-CurlText -Arguments $args
}

if (-not (Test-Path $MarkdownPath)) {
    throw "测试文档不存在: $MarkdownPath"
}

Write-Step "1. 健康检查"
$health = Invoke-CurlText -Arguments @("-sS", "$BaseUrl/healthz")
Write-Host $health

Write-Step "2. 上传文档"
$upload = Invoke-CurlText -Arguments @(
    "-sS",
    "-X", "POST",
    "$BaseUrl/api/kb/documents",
    "-H", "X-User-ID: $UserId",
    "-F", "file=@$MarkdownPath"
)
Write-Host $upload
$uploadJson = $upload | ConvertFrom-Json
$documentId = $uploadJson.DocumentID
if ([string]::IsNullOrWhiteSpace($documentId)) {
    throw "上传文档返回中未找到 DocumentID"
}

Write-Step "3. 普通问答"
$queryBody = @{
    session_id = ""
    message    = $Question
} | ConvertTo-Json
$query = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/chat/query" -Body $queryBody
Write-Host $query
$queryJson = $query | ConvertFrom-Json
$sessionId = $queryJson.session_id
if ([string]::IsNullOrWhiteSpace($sessionId)) {
    throw "问答返回中未找到 session_id"
}

Write-Step "4. 查询会话列表"
$sessions = Invoke-CurlText -Arguments @("-sS", "-H", "X-User-ID: $UserId", "$BaseUrl/api/chat/sessions")
Write-Host $sessions

Write-Step "5. 查询会话消息"
$messages = Invoke-CurlText -Arguments @("-sS", "-H", "X-User-ID: $UserId", "$BaseUrl/api/chat/sessions/$sessionId/messages")
Write-Host $messages

Write-Step "6. 知识反写"
$knowledgeBody = @{
    session_id  = $sessionId
    content     = $KnowledgeContent
    source_type = "manual"
} | ConvertTo-Json
$knowledge = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/kb/knowledge" -Body $knowledgeBody
Write-Host $knowledge

Write-Step "7. 重建索引"
$reindexBody = @{
    document_id = $documentId
} | ConvertTo-Json
$reindex = Invoke-JsonRequest -Method "POST" -Url "$BaseUrl/api/kb/reindex" -Body $reindexBody
Write-Host $reindex

Write-Step "8. 查看指标"
$metrics = Invoke-CurlText -Arguments @("-sS", "$BaseUrl/metrics")
$metrics | Select-String "knowflow_" | ForEach-Object { $_.Line } | Write-Host

Write-Step "完成"
Write-Host "document_id=$documentId"
Write-Host "session_id=$sessionId"
