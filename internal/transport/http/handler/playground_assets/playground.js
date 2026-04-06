const storageKey = "knowflow.playground";

const state = {
  baseUrl: window.location.origin,
  userId: "demo-user",
  sessionId: "",
  documentId: "",
  knowledgeEntryId: "",
  debugTab: "raw",
  sessions: [],
  messages: [],
  citations: [],
  lastRaw: null,
  lastRetrieval: null,
  lastTools: [],
  lastMetrics: "",
  streamAbortController: null,
  nextMessageId: 1,
};

const refs = {
  baseUrlInput: document.getElementById("baseUrlInput"),
  userIdInput: document.getElementById("userIdInput"),
  sessionIdInput: document.getElementById("sessionIdInput"),
  documentIdInput: document.getElementById("documentIdInput"),
  knowledgeEntryIdInput: document.getElementById("knowledgeEntryIdInput"),
  sourceNameInput: document.getElementById("sourceNameInput"),
  documentContentInput: document.getElementById("documentContentInput"),
  documentFileInput: document.getElementById("documentFileInput"),
  knowledgeContentInput: document.getElementById("knowledgeContentInput"),
  knowledgeSourceTypeInput: document.getElementById("knowledgeSourceTypeInput"),
  knowledgeSessionInput: document.getElementById("knowledgeSessionInput"),
  questionInput: document.getElementById("questionInput"),
  connectionBadge: document.getElementById("connectionBadge"),
  documentSummary: document.getElementById("documentSummary"),
  knowledgeSummary: document.getElementById("knowledgeSummary"),
  sessionList: document.getElementById("sessionList"),
  messageTimeline: document.getElementById("messageTimeline"),
  citationList: document.getElementById("citationList"),
  debugOutput: document.getElementById("debugOutput"),
  sessionLabel: document.getElementById("sessionLabel"),
  documentLabel: document.getElementById("documentLabel"),
  activityFeed: document.getElementById("activityFeed"),
};

function init() {
  hydrateFromStorage();
  seedInputs();
  bindEvents();
  updateContextLabels();
  renderSessions();
  renderMessages();
  renderCitations();
  renderDebug();
  pushActivity("Playground 已加载，建议先做健康检查，再上传面试文档。", "success");
}

function hydrateFromStorage() {
  const raw = window.localStorage.getItem(storageKey);
  if (!raw) {
    return;
  }
  try {
    const saved = JSON.parse(raw);
    state.baseUrl = saved.baseUrl || state.baseUrl;
    state.userId = saved.userId || state.userId;
    state.sessionId = saved.sessionId || "";
    state.documentId = saved.documentId || "";
    state.knowledgeEntryId = saved.knowledgeEntryId || "";
  } catch (error) {
    console.warn("failed to parse playground state", error);
  }
}

function persistStorage() {
  window.localStorage.setItem(
    storageKey,
    JSON.stringify({
      baseUrl: state.baseUrl,
      userId: state.userId,
      sessionId: state.sessionId,
      documentId: state.documentId,
      knowledgeEntryId: state.knowledgeEntryId,
    }),
  );
}

function seedInputs() {
  refs.baseUrlInput.value = state.baseUrl;
  refs.userIdInput.value = state.userId;
  refs.sessionIdInput.value = state.sessionId;
  refs.documentIdInput.value = state.documentId;
  refs.knowledgeEntryIdInput.value = state.knowledgeEntryId;
  refs.knowledgeSessionInput.value = state.sessionId;
  refs.documentContentInput.value =
    refs.documentContentInput.value ||
    [
      "# Redis 会话记忆",
      "",
      "- 最近窗口保留当前多轮上下文，保证回答连贯。",
      "- 历史摘要在消息过长时做压缩，降低 token 成本。",
      "- 摘要失败时退化为启发式摘要，避免链路中断。",
      "",
      "# 索引热更新",
      "",
      "- 文档摄取和知识反写落库后，重建受影响范围的索引。",
      "- 混合检索包含 PgVector 语义召回与 pg_trgm 关键词召回。",
    ].join("\n");
  refs.questionInput.value =
    refs.questionInput.value ||
    "请结合 Redis 双层记忆和知识反写热更新，解释 KnowFlow 在后端面试中的亮点。";
}

function bindEvents() {
  document.getElementById("syncContextButton").addEventListener("click", syncContextFromInputs);
  document.getElementById("healthButton").addEventListener("click", onHealthCheck);
  document.getElementById("metricsButton").addEventListener("click", onLoadMetrics);
  document.getElementById("uploadTextButton").addEventListener("click", () => onUploadDocument(false));
  document.getElementById("uploadFileButton").addEventListener("click", () => onUploadDocument(true));
  document.getElementById("upsertKnowledgeButton").addEventListener("click", onUpsertKnowledge);
  document.getElementById("reindexButton").addEventListener("click", onReindexDocument);
  document.getElementById("loadKnowledgeButton").addEventListener("click", loadKnowledgeEntries);
  document.getElementById("loadReindexTasksButton").addEventListener("click", onLoadReindexTasks);
  document.getElementById("refreshSessionsButton").addEventListener("click", onRefreshSessions);
  document.getElementById("queryButton").addEventListener("click", onQuery);
  document.getElementById("streamButton").addEventListener("click", onStreamQuery);
  document.getElementById("clearTimelineButton").addEventListener("click", clearWorkspace);

  document.querySelectorAll(".chip").forEach((chip) => {
    chip.addEventListener("click", () => {
      refs.questionInput.value = chip.dataset.prompt || "";
      refs.questionInput.focus();
    });
  });

  document.querySelectorAll(".debug-tab").forEach((tab) => {
    tab.addEventListener("click", () => {
      state.debugTab = tab.dataset.tab;
      renderDebug();
    });
  });
}

function syncContextFromInputs() {
  state.baseUrl = normalizeBaseUrl(refs.baseUrlInput.value);
  state.userId = (refs.userIdInput.value || "demo-user").trim();
  state.sessionId = refs.sessionIdInput.value.trim();
  state.documentId = refs.documentIdInput.value.trim();
  state.knowledgeEntryId = refs.knowledgeEntryIdInput.value.trim();
  refs.knowledgeSessionInput.value = refs.knowledgeSessionInput.value.trim() || state.sessionId;
  persistStorage();
  updateContextLabels();
  setConnectionBadge("已同步当前上下文。", "success");
}

function normalizeBaseUrl(value) {
  const trimmed = (value || "").trim();
  if (!trimmed) {
    return window.location.origin;
  }
  return trimmed.endsWith("/") ? trimmed.slice(0, -1) : trimmed;
}

function resolveUrl(path) {
  return new URL(path, `${state.baseUrl}/`).toString();
}

function jsonHeaders() {
  return {
    "Content-Type": "application/json",
    "X-User-ID": state.userId,
  };
}

async function onHealthCheck() {
  syncContextFromInputs();
  try {
    const response = await fetch(resolveUrl("/healthz"), {
      headers: {
        "X-User-ID": state.userId,
      },
    });
    const data = await response.json();
    state.lastRaw = data;
    renderDebug();
    if (!response.ok) {
      throw new Error(`healthz failed: ${response.status}`);
    }
    setConnectionBadge(`服务可用，status=${data.status || "ok"}`, "success");
    pushActivity("健康检查通过。", "success");
  } catch (error) {
    handleError("健康检查失败", error);
  }
}

async function onLoadMetrics() {
  syncContextFromInputs();
  try {
    const response = await fetch(resolveUrl("/metrics"), {
      headers: {
        "X-User-ID": state.userId,
      },
    });
    const text = await response.text();
    state.lastMetrics = text;
    state.lastRaw = { metrics_loaded: text.length > 0 };
    state.debugTab = "metrics";
    renderDebug();
    if (!response.ok) {
      throw new Error(`metrics failed: ${response.status}`);
    }
    pushActivity("指标快照已刷新。", "success");
  } catch (error) {
    handleError("读取指标失败", error);
  }
}

async function onUploadDocument(useFile) {
  syncContextFromInputs();
  const formData = new FormData();
  if (useFile) {
    const file = refs.documentFileInput.files[0];
    if (!file) {
      handleError("上传文件失败", new Error("请先选择 .md 或 .txt 文件"));
      return;
    }
    formData.append("file", file);
  } else {
    formData.append("source_name", refs.sourceNameInput.value.trim());
    formData.append("content", refs.documentContentInput.value);
  }

  try {
    const response = await fetch(resolveUrl("/api/kb/documents"), {
      method: "POST",
      headers: {
        "X-User-ID": state.userId,
      },
      body: formData,
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    const documentId = data.document_id || data.DocumentID || "";
    if (documentId) {
      state.documentId = documentId;
      refs.documentIdInput.value = documentId;
      persistStorage();
      updateContextLabels();
    }
    refs.documentSummary.textContent = safePretty(data);
    renderDebug();
    pushActivity(`文档摄取完成，document_id=${documentId || "未知"}`, "success");
  } catch (error) {
    handleError("文档摄取失败", error);
  }
}

async function onUpsertKnowledge() {
  syncContextFromInputs();
  const content = refs.knowledgeContentInput.value.trim();
  if (!content) {
    handleError("知识反写失败", new Error("请先填写要沉淀的知识内容"));
    return;
  }

  const payload = {
    user_id: state.userId,
    session_id: (refs.knowledgeSessionInput.value || state.sessionId).trim(),
    source_type: (refs.knowledgeSourceTypeInput.value || "manual").trim(),
    content,
  };

  try {
    const response = await fetch(resolveUrl("/api/kb/knowledge"), {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify(payload),
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    const knowledgeEntryId = data.result?.data?.id || data.data?.id || data.id || "";
    if (knowledgeEntryId) {
      state.knowledgeEntryId = knowledgeEntryId;
      refs.knowledgeEntryIdInput.value = knowledgeEntryId;
      persistStorage();
    }
    refs.knowledgeSummary.textContent = safePretty(data);
    renderDebug();
    pushActivity(`知识条目已写入，knowledge_entry_id=${knowledgeEntryId || "未知"}`, "success");
  } catch (error) {
    handleError("知识反写失败", error);
  }
}

async function onReindexDocument() {
  syncContextFromInputs();
  if (!state.documentId && !state.knowledgeEntryId) {
    handleError("重建索引失败", new Error("当前 document_id 和 knowledge_entry_id 都为空"));
    return;
  }

  try {
    const payload = state.knowledgeEntryId
      ? { knowledge_entry_id: state.knowledgeEntryId }
      : { document_id: state.documentId };
    const response = await fetch(resolveUrl("/api/kb/reindex"), {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify(payload),
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    refs.knowledgeSummary.textContent = safePretty(data);
    renderDebug();
    const label = state.knowledgeEntryId
      ? `knowledge_entry_id=${state.knowledgeEntryId}`
      : `document_id=${state.documentId}`;
    pushActivity(`索引已重建，${label}`, "warn");
  } catch (error) {
    handleError("重建索引失败", error);
  }
}

async function loadKnowledgeEntries() {
  syncContextFromInputs();
  try {
    const response = await fetch(resolveUrl("/api/kb/knowledge"), {
      headers: {
        "X-User-ID": state.userId,
      },
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    refs.knowledgeSummary.textContent = safePretty(data);
    renderDebug();
    const size = Array.isArray(data) ? data.length : 0;
    pushActivity(`知识列表已刷新，共 ${size} 条。`, "success");
  } catch (error) {
    handleError("加载知识列表失败", error);
  }
}

async function onLoadReindexTasks() {
  syncContextFromInputs();
  try {
    const response = await fetch(resolveUrl("/api/kb/reindex/tasks"), {
      headers: {
        "X-User-ID": state.userId,
      },
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    refs.knowledgeSummary.textContent = safePretty(data);
    renderDebug();
    const size = Array.isArray(data) ? data.length : 0;
    pushActivity(`重建任务列表已刷新，共 ${size} 条。`, "warn");
  } catch (error) {
    handleError("加载重建任务列表失败", error);
  }
}

async function onRefreshSessions() {
  syncContextFromInputs();
  try {
    const response = await fetch(resolveUrl("/api/chat/sessions"), {
      headers: {
        "X-User-ID": state.userId,
      },
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    state.sessions = Array.isArray(data) ? data.map(normalizeSession) : [];
    renderSessions();
    renderDebug();
    pushActivity(`会话列表已刷新，共 ${state.sessions.length} 条。`, "success");
  } catch (error) {
    handleError("加载会话列表失败", error);
  }
}

async function onQuery() {
  syncContextFromInputs();
  const question = refs.questionInput.value.trim();
  if (!question) {
    handleError("普通问答失败", new Error("问题不能为空"));
    return;
  }

  addMessage("user", question);

  try {
    const response = await fetch(resolveUrl("/api/chat/query"), {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({
        session_id: state.sessionId,
        message: question,
      }),
    });
    const data = await parseJsonResponse(response);
    applyQueryResponse(data, false);
    pushActivity("普通问答完成。", "success");
  } catch (error) {
    handleError("普通问答失败", error);
  }
}

async function onStreamQuery() {
  syncContextFromInputs();
  const question = refs.questionInput.value.trim();
  if (!question) {
    handleError("SSE 问答失败", new Error("问题不能为空"));
    return;
  }

  if (state.streamAbortController) {
    state.streamAbortController.abort();
  }

  addMessage("user", question);
  const assistantMessage = addMessage("assistant", "等待流式返回...");
  state.streamAbortController = new AbortController();
  setConnectionBadge("SSE 流式响应中...", "warn");

  try {
    const response = await fetch(resolveUrl("/api/chat/query/stream"), {
      method: "POST",
      headers: jsonHeaders(),
      body: JSON.stringify({
        session_id: state.sessionId,
        message: question,
      }),
      signal: state.streamAbortController.signal,
    });
    if (!response.ok || !response.body) {
      const text = await response.text();
      throw new Error(text || `stream failed: ${response.status}`);
    }

    const reader = response.body.getReader();
    const decoder = new TextDecoder("utf-8");
    let buffer = "";
    assistantMessage.content = "";
    updateMessageContent(assistantMessage.id, assistantMessage.content);

    while (true) {
      const { value, done } = await reader.read();
      if (done) {
        break;
      }
      buffer += decoder.decode(value, { stream: true });
      buffer = consumeSSE(buffer, (eventName, payload) => {
        if (eventName === "delta") {
          assistantMessage.content += payload.content || "";
          updateMessageContent(assistantMessage.id, assistantMessage.content);
          return;
        }
        if (eventName === "done") {
          applyQueryResponse(payload, true);
        }
      });
    }

    buffer += decoder.decode();
    consumeSSE(buffer, (eventName, payload) => {
      if (eventName === "done") {
        applyQueryResponse(payload, true);
      }
    });
    pushActivity("SSE 问答完成。", "success");
  } catch (error) {
    if (error.name === "AbortError") {
      pushActivity("上一次 SSE 已中止。", "warn");
      return;
    }
    assistantMessage.content = `流式请求失败：${error.message}`;
    updateMessageContent(assistantMessage.id, assistantMessage.content);
    handleError("SSE 问答失败", error);
  } finally {
    state.streamAbortController = null;
  }
}

function consumeSSE(buffer, onEvent) {
  let normalized = buffer;
  let divider = normalized.indexOf("\n\n");
  while (divider !== -1) {
    const block = normalized.slice(0, divider).trim();
    normalized = normalized.slice(divider + 2);
    if (block) {
      const parsed = parseSSEBlock(block);
      if (parsed) {
        onEvent(parsed.event, parsed.data);
      }
    }
    divider = normalized.indexOf("\n\n");
  }
  return normalized;
}

function parseSSEBlock(block) {
  const lines = block.split(/\r?\n/);
  let eventName = "message";
  const dataLines = [];
  lines.forEach((line) => {
    if (line.startsWith("event:")) {
      eventName = line.slice(6).trim();
      return;
    }
    if (line.startsWith("data:")) {
      dataLines.push(line.slice(5).trim());
    }
  });
  if (dataLines.length === 0) {
    return null;
  }
  const joined = dataLines.join("\n");
  try {
    return {
      event: eventName,
      data: JSON.parse(joined),
    };
  } catch (error) {
    return {
      event: eventName,
      data: { content: joined },
    };
  }
}

async function parseJsonResponse(response) {
  const text = await response.text();
  let data = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch (error) {
      data = { raw: text };
    }
  }
  if (!response.ok) {
    throw new Error(data.error || data.raw || `request failed: ${response.status}`);
  }
  return data;
}

function applyQueryResponse(data, isStreaming) {
  state.lastRaw = data;
  state.lastRetrieval = data.retrieval_meta || data.RetrievalMeta || null;
  state.lastTools = data.tool_traces || data.ToolTraces || [];
  state.citations = Array.isArray(data.citations || data.Citations)
    ? (data.citations || data.Citations).map(normalizeCitation)
    : [];

  const sessionId = data.session_id || data.SessionID || "";
  if (sessionId) {
    state.sessionId = sessionId;
    refs.sessionIdInput.value = sessionId;
    refs.knowledgeSessionInput.value = sessionId;
    persistStorage();
    updateContextLabels();
  }

  const answer = data.answer || data.Answer || "";
  if (isStreaming && state.messages.length > 0) {
    const lastMessage = state.messages[state.messages.length - 1];
    lastMessage.content = answer || lastMessage.content;
    updateMessageContent(lastMessage.id, lastMessage.content);
  } else {
    addMessage("assistant", answer || "未返回回答内容");
  }

  if (!isStreaming) {
    renderMessages();
  }
  renderCitations();
  renderDebug();
}

function normalizeSession(item) {
  return {
    id: item.id || item.ID || "",
    title: item.title || item.Title || "未命名会话",
    lastActiveAt: item.last_active_at || item.LastActiveAt || "",
  };
}

function normalizeCitation(item) {
  return {
    sourceName: item.source_name || item.SourceName || "unknown",
    chunkId: item.chunk_id || item.ChunkID || "",
    snippet: item.snippet || item.Snippet || "",
    documentId: item.document_id || item.DocumentID || "",
    knowledgeEntryId: item.knowledge_entry_id || item.KnowledgeEntryID || "",
    sourceKind: item.source_kind || item.SourceKind || "",
  };
}

function newMessageId() {
  const id = `msg-${state.nextMessageId}`;
  state.nextMessageId += 1;
  return id;
}

function addMessage(role, content) {
  const message = {
    id: newMessageId(),
    role,
    content,
    createdAt: new Date().toISOString(),
  };
  state.messages.push(message);
  renderMessages();
  return message;
}

function updateMessageContent(messageId, content) {
  const contentNode = refs.messageTimeline.querySelector(`[data-message-id="${messageId}"] .message-content`);
  if (!contentNode) {
    renderMessages();
    return;
  }
  contentNode.textContent = content || "";
}

function renderMessages() {
  if (state.messages.length === 0) {
    refs.messageTimeline.textContent = "还没有消息，先上传一份面试资料再提问吧。";
    refs.messageTimeline.className = "message-timeline empty-state";
    return;
  }

  refs.messageTimeline.className = "message-timeline";
  refs.messageTimeline.innerHTML = state.messages
    .map((message) => {
      const roleLabel = message.role === "user" ? "候选人提问" : "KnowFlow 回答";
      return `
        <article class="message-card ${message.role}" data-message-id="${escapeHtml(message.id)}">
          <div class="message-meta">
            <strong>${escapeHtml(roleLabel)}</strong>
            <span>${escapeHtml(formatTime(message.createdAt))}</span>
          </div>
          <div class="message-content">${escapeHtml(message.content || "")}</div>
        </article>
      `;
    })
    .join("");
}

function renderCitations() {
  if (!state.citations.length) {
    refs.citationList.textContent = "引用会显示在这里。";
    refs.citationList.className = "citation-list empty-state";
    return;
  }

  refs.citationList.className = "citation-list";
  refs.citationList.innerHTML = state.citations
    .map((citation) => {
      return `
        <article class="citation-item">
          <strong>${escapeHtml(citation.sourceName)}</strong>
          <p>${escapeHtml(citation.snippet)}</p>
          <code>${escapeHtml(citation.chunkId || citation.documentId || citation.knowledgeEntryId)}</code>
        </article>
      `;
    })
    .join("");
}

function renderSessions() {
  if (!state.sessions.length) {
    refs.sessionList.textContent = "暂无会话。";
    refs.sessionList.className = "session-list empty-state";
    return;
  }

  refs.sessionList.className = "session-list";
  refs.sessionList.innerHTML = "";
  state.sessions.forEach((session) => {
    const button = document.createElement("button");
    button.type = "button";
    button.className = `session-item${session.id === state.sessionId ? " active" : ""}`;
    button.innerHTML = `<strong>${escapeHtml(session.title)}</strong><span>${escapeHtml(session.id)}</span><span>${escapeHtml(formatTime(session.lastActiveAt))}</span>`;
    button.addEventListener("click", () => onLoadMessages(session.id));
    refs.sessionList.appendChild(button);
  });
}

async function onLoadMessages(sessionId) {
  syncContextFromInputs();
  try {
    const response = await fetch(resolveUrl(`/api/chat/sessions/${encodeURIComponent(sessionId)}/messages`), {
      headers: {
        "X-User-ID": state.userId,
      },
    });
    const data = await parseJsonResponse(response);
    state.lastRaw = data;
    state.sessionId = sessionId;
    refs.sessionIdInput.value = sessionId;
    refs.knowledgeSessionInput.value = sessionId;
    state.messages = Array.isArray(data)
      ? data.map((message) => ({
          role: (message.role || message.Role || "assistant").toLowerCase(),
          content: message.content || message.Content || "",
          createdAt: message.created_at || message.CreatedAt || new Date().toISOString(),
        }))
      : [];
    persistStorage();
    updateContextLabels();
    renderMessages();
    renderSessions();
    renderDebug();
    pushActivity(`已加载会话 ${sessionId} 的消息历史。`, "success");
  } catch (error) {
    handleError("加载消息历史失败", error);
  }
}

function renderDebug() {
  document.querySelectorAll(".debug-tab").forEach((tab) => {
    tab.classList.toggle("active", tab.dataset.tab === state.debugTab);
  });

  let payload = "等待请求...";
  if (state.debugTab === "raw") {
    payload = safePretty(state.lastRaw);
  }
  if (state.debugTab === "retrieval") {
    payload = safePretty(state.lastRetrieval);
  }
  if (state.debugTab === "tools") {
    payload = safePretty(state.lastTools);
  }
  if (state.debugTab === "metrics") {
    payload = state.lastMetrics || "尚未抓取 metrics。";
  }
  refs.debugOutput.textContent = payload;
}

function updateContextLabels() {
  refs.sessionLabel.textContent = state.sessionId || "未建立";
  refs.documentLabel.textContent = state.documentId || "未上传";
}

function setConnectionBadge(message, tone) {
  refs.connectionBadge.textContent = message;
  refs.connectionBadge.className = `status-banner ${tone || "neutral"}`;
}

function pushActivity(message, tone) {
  const item = document.createElement("div");
  item.className = `activity-item ${tone || "neutral"}`;
  item.textContent = `${formatTime(new Date().toISOString())} · ${message}`;
  refs.activityFeed.prepend(item);
}

function clearWorkspace() {
  state.messages = [];
  state.citations = [];
  state.lastRaw = null;
  state.lastRetrieval = null;
  state.lastTools = [];
  renderMessages();
  renderCitations();
  renderDebug();
  pushActivity("右侧展示区已清空。", "warn");
}

function handleError(prefix, error) {
  const message = `${prefix}：${error.message || error}`;
  setConnectionBadge(message, "error");
  pushActivity(message, "error");
  state.lastRaw = { error: message };
  renderDebug();
}

function safePretty(value) {
  if (value === null || value === undefined || value === "") {
    return "暂无数据";
  }
  if (typeof value === "string") {
    return value;
  }
  try {
    return JSON.stringify(value, null, 2);
  } catch (error) {
    return String(value);
  }
}

function formatTime(value) {
  if (!value) {
    return "just now";
  }
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) {
    return String(value);
  }
  return parsed.toLocaleString("zh-CN", {
    hour12: false,
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function escapeHtml(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");
}

window.addEventListener("DOMContentLoaded", init);
