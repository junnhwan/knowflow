package observability

import (
	"net/http"
	"net/http/httptest"
	"strings"

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
	toolCalls        *prometheus.CounterVec
	toolCallFailures *prometheus.CounterVec
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

func (m *Metrics) RecordToolCall(toolName string, success bool) {
	m.toolCalls.WithLabelValues(toolName).Inc()
	if !success {
		m.toolCallFailures.WithLabelValues(toolName).Inc()
	}
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
