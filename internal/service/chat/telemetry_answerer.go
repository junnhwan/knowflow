package chat

import (
	"context"
	"time"
)

type TelemetryObserver interface {
	RecordLLMRequest(provider string)
	RecordLLMLatency(provider string, duration time.Duration)
}

type TelemetryAnswerer struct {
	provider string
	next     Answerer
	observer TelemetryObserver
}

func NewTelemetryAnswerer(provider string, next Answerer, observer TelemetryObserver) *TelemetryAnswerer {
	return &TelemetryAnswerer{
		provider: provider,
		next:     next,
		observer: observer,
	}
}

func (a *TelemetryAnswerer) Generate(ctx context.Context, req PromptRequest) (PromptResult, error) {
	start := time.Now()
	result, err := a.next.Generate(ctx, req)
	if a.observer != nil {
		a.observer.RecordLLMRequest(a.provider)
		a.observer.RecordLLMLatency(a.provider, time.Since(start))
	}
	return result, err
}

func (a *TelemetryAnswerer) Stream(ctx context.Context, req PromptRequest, onDelta func(string) error) (PromptResult, error) {
	start := time.Now()
	result, err := a.next.Stream(ctx, req, onDelta)
	if a.observer != nil {
		a.observer.RecordLLMRequest(a.provider)
		a.observer.RecordLLMLatency(a.provider, time.Since(start))
	}
	return result, err
}
