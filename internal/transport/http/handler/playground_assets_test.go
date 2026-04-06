package handler

import (
	"os"
	"strings"
	"testing"
)

func TestPlaygroundScriptStreamsByUpdatingMessageContentInPlace(t *testing.T) {
	content, err := os.ReadFile("playground_assets/playground.js")
	if err != nil {
		t.Fatalf("read playground.js: %v", err)
	}

	script := string(content)
	if !strings.Contains(script, "updateMessageContent(") {
		t.Fatal("expected in-place message update helper for streaming")
	}

	if strings.Contains(script, `assistantMessage.content += payload.content || "";
          renderMessages();`) {
		t.Fatal("expected streaming delta path to avoid full timeline rerender")
	}
}

func TestPlaygroundScriptIncludesKnowledgeAndTaskLoaders(t *testing.T) {
	content, err := os.ReadFile("playground_assets/playground.js")
	if err != nil {
		t.Fatalf("read playground.js: %v", err)
	}

	script := string(content)
	if !strings.Contains(script, "/api/kb/reindex/tasks") {
		t.Fatal("expected playground to support loading reindex tasks")
	}
	if !strings.Contains(script, "loadKnowledgeEntries") {
		t.Fatal("expected playground to support loading knowledge entries")
	}
}
