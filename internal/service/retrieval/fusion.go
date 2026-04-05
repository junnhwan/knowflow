package retrieval

import "sort"

func FuseWithRRF(vectorCandidates []Candidate, keywordCandidates []Candidate, k int) []Candidate {
	if k <= 0 {
		k = 60
	}

	fused := make(map[string]Candidate)
	apply := func(candidates []Candidate, kind string) {
		for index, candidate := range candidates {
			current := fused[candidate.ChunkID]
			if current.ChunkID == "" {
				current = candidate
			}
			rrf := 1.0 / float64(k+index+1)
			if kind == "vector" {
				current.VectorScore = candidate.VectorScore
			}
			if kind == "keyword" {
				current.KeywordScore = candidate.KeywordScore
			}
			current.FusionScore += rrf
			fused[candidate.ChunkID] = current
		}
	}

	apply(vectorCandidates, "vector")
	apply(keywordCandidates, "keyword")

	out := make([]Candidate, 0, len(fused))
	for _, candidate := range fused {
		out = append(out, candidate)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].FusionScore == out[j].FusionScore {
			return out[i].ChunkID < out[j].ChunkID
		}
		return out[i].FusionScore > out[j].FusionScore
	})
	return out
}
