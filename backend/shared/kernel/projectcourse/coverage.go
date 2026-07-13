package projectcourse

type CoverageItem struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	Label      string   `json:"label"`
	ChapterIDs []string `json:"chapter_ids"`
	Covered    bool     `json:"covered"`
}

type CoverageMatrix struct {
	Modules      []CoverageItem `json:"modules"`
	MainFlows    []CoverageItem `json:"main_flows"`
	Technologies []CoverageItem `json:"technologies"`
	Files        []CoverageItem `json:"files"`
}

func (m CoverageMatrix) CoveredRate(items []CoverageItem) float64 {
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
