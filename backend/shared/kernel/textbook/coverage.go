package textbook

type CoverageItem struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Label      string   `json:"label"`
	ChapterIDs []string `json:"chapter_ids"`
	Covered    bool     `json:"covered"`
}

type CoverageMatrix struct {
	Concepts  []CoverageItem `json:"concepts"`
	Evidence  []CoverageItem `json:"evidence"`
	Exercises []CoverageItem `json:"exercises"`
}

func (matrix CoverageMatrix) CoveredRate(items []CoverageItem) float64 {
	if len(items) == 0 {
		return 1
	}
	covered := 0
	for _, item := range items {
		if item.Covered {
			covered++
		}
	}
	return float64(covered) / float64(len(items))
}
