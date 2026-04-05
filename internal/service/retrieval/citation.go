package retrieval

type Citation struct {
	DocumentID string `json:"document_id"`
	SourceName string `json:"source_name"`
	ChunkID    string `json:"chunk_id"`
	Snippet    string `json:"snippet"`
}

func BuildCitations(candidates []Candidate) []Citation {
	out := make([]Citation, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, Citation{
			DocumentID: candidate.DocumentID,
			SourceName: candidate.SourceName,
			ChunkID:    candidate.ChunkID,
			Snippet:    clipSnippet(candidate.Content),
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
