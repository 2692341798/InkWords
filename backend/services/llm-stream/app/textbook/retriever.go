package textbook

import "context"

// Retriever decouples chapter generation from lexical or future embedding implementations.
type Retriever interface {
	Retrieve(context.Context, string, []RetrievalCandidate, int) (RetrievalPlan, error)
}
type LexicalRetriever struct{}

func (LexicalRetriever) Retrieve(ctx context.Context, query string, candidates []RetrievalCandidate, limit int) (RetrievalPlan, error) {
	if err := ctx.Err(); err != nil {
		return RetrievalPlan{}, err
	}
	return Retrieve(query, candidates, limit)
}
