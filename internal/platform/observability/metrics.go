package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	registry         *prometheus.Registry
	llmRequests      *prometheus.CounterVec
	llmLatency       *prometheus.HistogramVec
	ragHits          *prometheus.CounterVec
	ragMisses        *prometheus.CounterVec
	rerankFallbacks  *prometheus.CounterVec
	guardrailRejects *prometheus.CounterVec
	reindexTasks     *prometheus.CounterVec
	toolCalls        *prometheus.CounterVec
	toolCallFailures *prometheus.CounterVec
	knowledgeExtract *prometheus.CounterVec
	knowledgeDedupe  *prometheus.CounterVec
	knowledgeMerge   *prometheus.CounterVec
}

func NewMetrics() *Metrics {
	registry := prometheus.NewRegistry()
	metrics := &Metrics{
		registry: registry,
		llmRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_llm_request_total",
			Help: "Total llm requests",
		}, []string{"provider"}),
		llmLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "knowflow_llm_latency_seconds",
			Help:    "LLM latency in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"provider"}),
		ragHits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_rag_hit_total",
			Help: "Total RAG hits",
		}, []string{"user_id", "session_id"}),
		ragMisses: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_rag_miss_total",
			Help: "Total RAG misses",
		}, []string{"user_id", "session_id"}),
		rerankFallbacks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_rerank_fallback_total",
			Help: "Total rerank fallbacks",
		}, []string{"reason"}),
		guardrailRejects: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_guardrail_reject_total",
			Help: "Total guardrail rejections",
		}, []string{"endpoint", "reason"}),
		reindexTasks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_reindex_task_total",
			Help: "Total background reindex tasks",
		}, []string{"target_type", "result"}),
		knowledgeExtract: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_knowledge_extract_total",
			Help: "Total knowledge extraction attempts",
		}, []string{"result"}),
		knowledgeDedupe: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_knowledge_dedupe_total",
			Help: "Total knowledge dedupe detections",
		}, []string{"result"}),
		knowledgeMerge: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_knowledge_merge_total",
			Help: "Total knowledge merge operations",
		}, []string{"result"}),
		toolCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_tool_call_total",
			Help: "Total tool calls",
		}, []string{"tool_name"}),
		toolCallFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "knowflow_tool_call_fail_total",
			Help: "Total failed tool calls",
		}, []string{"tool_name"}),
	}

	registry.MustRegister(
		metrics.llmRequests,
		metrics.llmLatency,
		metrics.ragHits,
		metrics.ragMisses,
		metrics.rerankFallbacks,
		metrics.guardrailRejects,
		metrics.reindexTasks,
		metrics.knowledgeExtract,
		metrics.knowledgeDedupe,
		metrics.knowledgeMerge,
		metrics.toolCalls,
		metrics.toolCallFailures,
	)

	return metrics
}

func (m *Metrics) RecordRAGHit(userID, sessionID string) {
	m.ragHits.WithLabelValues(userID, sessionID).Inc()
}

func (m *Metrics) RecordRAGMiss(userID, sessionID string) {
	m.ragMisses.WithLabelValues(userID, sessionID).Inc()
}

func (m *Metrics) RecordRerankFallback(reason string) {
	m.rerankFallbacks.WithLabelValues(strings.ToLower(reason)).Inc()
}

func (m *Metrics) RecordGuardrailReject(endpoint, reason string) {
	m.guardrailRejects.WithLabelValues(strings.ToLower(endpoint), strings.ToLower(reason)).Inc()
}

func (m *Metrics) RecordReindexTask(targetType, result string) {
	m.reindexTasks.WithLabelValues(strings.ToLower(targetType), strings.ToLower(result)).Inc()
}

func (m *Metrics) RecordKnowledgeExtraction(result string) {
	m.knowledgeExtract.WithLabelValues(strings.ToLower(result)).Inc()
}

func (m *Metrics) RecordKnowledgeDedupe(result string) {
	m.knowledgeDedupe.WithLabelValues(strings.ToLower(result)).Inc()
}

func (m *Metrics) RecordKnowledgeMerge(result string) {
	m.knowledgeMerge.WithLabelValues(strings.ToLower(result)).Inc()
}

func (m *Metrics) RecordToolCall(toolName string, success bool) {
	m.toolCalls.WithLabelValues(toolName).Inc()
	if !success {
		m.toolCallFailures.WithLabelValues(toolName).Inc()
	}
}

func (m *Metrics) RecordLLMRequest(provider string) {
	m.llmRequests.WithLabelValues(provider).Inc()
}

func (m *Metrics) RecordLLMLatency(provider string, duration time.Duration) {
	m.llmLatency.WithLabelValues(provider).Observe(duration.Seconds())
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

func (m *Metrics) Expose() (string, error) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	m.Handler().ServeHTTP(recorder, request)
	return recorder.Body.String(), nil
}
