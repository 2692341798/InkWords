package service

import (
	"fmt"

	"github.com/google/uuid"
)

// StoreGenerateSeriesTaskResult caches the final generate_series task result
// until the stream layer marks the task as succeeded and writes result_json.
func (s *DecompositionService) StoreGenerateSeriesTaskResult(parentID uuid.UUID, resultJSON []byte) {
	if s == nil {
		return
	}

	s.seriesTaskResultsMu.Lock()
	defer s.seriesTaskResultsMu.Unlock()

	s.seriesTaskResults[parentID.String()] = append([]byte(nil), resultJSON...)
}

// TakeGenerateSeriesTaskResult returns and clears the cached generate_series
// task result so one task completion cannot be replayed across later requests.
func (s *DecompositionService) TakeGenerateSeriesTaskResult(parentID uuid.UUID) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("decomposition service is not configured")
	}

	s.seriesTaskResultsMu.Lock()
	defer s.seriesTaskResultsMu.Unlock()

	resultJSON, ok := s.seriesTaskResults[parentID.String()]
	if !ok {
		return nil, fmt.Errorf("generate_series task result not found for parent %s", parentID)
	}

	delete(s.seriesTaskResults, parentID.String())
	return append([]byte(nil), resultJSON...), nil
}
