package retrieval

type Citation struct {
	DocumentID       string `json:"document_id"`
	KnowledgeEntryID string `json:"knowledge_entry_id,omitempty"`
	SourceName       string `json:"source_name"`
	SourceKind       string `json:"source_kind,omitempty"`
	ChunkID          string `json:"chunk_id"`
	Snippet          string `json:"snippet"`
}

func BuildCitations(candidates []Candidate) []Citation {
	out := make([]Citation, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, Citation{
			DocumentID:       candidate.DocumentID,
			KnowledgeEntryID: candidate.KnowledgeEntryID,
			SourceName:       candidate.SourceName,
			SourceKind:       candidate.SourceKind,
			ChunkID:          candidate.ChunkID,
			Snippet:          clipSnippet(candidate.Content),
		})
	}
	return out
}

func clipSnippet(content string) string {
	runes := []rune(content)
	if len(runes) <= 120 {
		return content
	}
	return string(runes[:120])
}
