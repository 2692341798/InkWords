package api

import (
	streamdomain "inkwords-backend/internal/domain/stream"
)

// Deprecated: llm-stream now owns its primary route registration in services/llm-stream.
// StreamAPI remains as a transitional adapter while generation and decomposition use cases are being migrated.
type StreamAPI struct {
	streamDomainHandler *streamdomain.Handler
}

// NewStreamAPIWithDeps creates a StreamAPI with explicitly injected dependencies.
func NewStreamAPIWithDeps(streamDomainHandler *streamdomain.Handler) *StreamAPI {
	return &StreamAPI{
		streamDomainHandler: streamDomainHandler,
	}
}
