package ingestion

import "strings"

type Splitter struct {
	ChunkSize    int
	ChunkOverlap int
	Separators   []string
}

func NewSplitter(chunkSize, chunkOverlap int) Splitter {
	if chunkSize <= 0 {
		chunkSize = 700
	}
	if chunkOverlap < 0 {
		chunkOverlap = 0
	}
	return Splitter{
		ChunkSize:    chunkSize,
		ChunkOverlap: chunkOverlap,
		Separators:   []string{"\n# ", "\n## ", "\n\n", "\n", "。", "！", "？", ". ", " "},
	}
}

func (s Splitter) Split(text string) []string {
	chunks := s.splitRecursive(text, 0)
	out := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunk = strings.TrimSpace(chunk)
		if chunk != "" {
			out = append(out, chunk)
		}
	}
	return out
}

func (s Splitter) splitRecursive(text string, level int) []string {
	if len([]rune(text)) <= s.ChunkSize {
		return []string{text}
	}

	if level >= len(s.Separators) {
		return s.splitByLength(text)
	}

	separator := s.Separators[level]
	parts := strings.Split(text, separator)
	if len(parts) == 1 {
		return s.splitRecursive(text, level+1)
	}

	var result []string
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		candidate := part
		if builder.Len() > 0 {
			candidate = builder.String() + separator + part
		}

		if len([]rune(candidate)) <= s.ChunkSize {
			builder.Reset()
			builder.WriteString(candidate)
			continue
		}

		if builder.Len() > 0 {
			result = append(result, builder.String())
		}

		if len([]rune(part)) > s.ChunkSize {
			result = append(result, s.splitRecursive(part, level+1)...)
			builder.Reset()
			continue
		}

		builder.Reset()
		builder.WriteString(part)
	}

	if builder.Len() > 0 {
		result = append(result, builder.String())
	}

	return s.addOverlap(result)
}

func (s Splitter) splitByLength(text string) []string {
	runes := []rune(text)
	var result []string
	for start := 0; start < len(runes); start += s.ChunkSize {
		end := start + s.ChunkSize
		if end > len(runes) {
			end = len(runes)
		}
		result = append(result, string(runes[start:end]))
	}
	return s.addOverlap(result)
}

func (s Splitter) addOverlap(chunks []string) []string {
	if s.ChunkOverlap == 0 || len(chunks) <= 1 {
		return chunks
	}

	out := make([]string, 0, len(chunks))
	for index, chunk := range chunks {
		if index == 0 {
			out = append(out, chunk)
			continue
		}
		prevRunes := []rune(chunks[index-1])
		start := len(prevRunes) - s.ChunkOverlap
		if start < 0 {
			start = 0
		}
		out = append(out, string(prevRunes[start:])+chunk)
	}
	return out
}
