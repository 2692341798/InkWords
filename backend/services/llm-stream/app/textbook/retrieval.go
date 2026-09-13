package textbook

import sharedtextbook "inkwords-backend/shared/kernel/textbook"

// These aliases keep the worker's future Retriever port stable while the
// deterministic ranking contract lives in shared/kernel. Core-api and the
// worker therefore cannot drift in score explanations or evidence packing.
type RetrievalCandidate = sharedtextbook.RetrievalCandidate
type RetrievalPlan = sharedtextbook.RetrievalPlan

func Retrieve(query string, candidates []RetrievalCandidate, limit int) (RetrievalPlan, error) {
	return sharedtextbook.RetrieveEvidence(query, candidates, limit)
}

func BuildEvidencePack(plan RetrievalPlan) (EvidencePack, error) {
	return sharedtextbook.BuildGenerationEvidencePack(plan)
}
