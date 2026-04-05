package memory

import "context"

type HeuristicSummaryGenerator struct{}

func (HeuristicSummaryGenerator) Summarize(_ context.Context, messages []MessageMemory, _ int) (string, error) {
	return heuristicSummary(messages), nil
}
