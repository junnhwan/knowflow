package tools

import (
	"context"
	"fmt"
	"time"
)

type Output struct {
	Status string         `json:"status"`
	Data   any            `json:"data,omitempty"`
	Error  string         `json:"error,omitempty"`
	Meta   map[string]any `json:"meta,omitempty"`
}

type Trace struct {
	ToolName   string `json:"tool_name"`
	Status     string `json:"status"`
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

type Tool interface {
	Execute(ctx context.Context, input map[string]any) (Output, error)
}

type ToolFunc func(ctx context.Context, input map[string]any) (Output, error)

func (f ToolFunc) Execute(ctx context.Context, input map[string]any) (Output, error) {
	return f(ctx, input)
}

type ServiceConfig struct {
	Timeout time.Duration
}

type Registry struct {
	config ServiceConfig
	tools  map[string]Tool
	observer ToolObserver
}

type ToolObserver interface {
	RecordToolCall(toolName string, success bool)
}

func NewRegistry(cfg ServiceConfig) *Registry {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 3 * time.Second
	}
	return &Registry{
		config: cfg,
		tools:  map[string]Tool{},
	}
}

func (r *Registry) SetObserver(observer ToolObserver) {
	r.observer = observer
}

func (r *Registry) Register(name string, tool Tool) error {
	if name == "" {
		return fmt.Errorf("tool name is required")
	}
	if tool == nil {
		return fmt.Errorf("tool is required")
	}
	r.tools[name] = tool
	return nil
}

func (r *Registry) Execute(ctx context.Context, name string, input map[string]any) (Output, error) {
	tool, ok := r.tools[name]
	if !ok {
		if r.observer != nil {
			r.observer.RecordToolCall(name, false)
		}
		return Output{Status: "error", Error: "tool not found"}, fmt.Errorf("tool not found: %s", name)
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()
	result, err := tool.Execute(timeoutCtx, input)
	if r.observer != nil {
		r.observer.RecordToolCall(name, err == nil)
	}
	return result, err
}
