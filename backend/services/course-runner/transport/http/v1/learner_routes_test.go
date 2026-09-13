package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	domain "inkwords-backend/services/course-runner/domain/learnerverification"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

type rejectingResolver struct{ calls int }

func (resolver *rejectingResolver) Resolve(context.Context, sharedtextbook.LearnerVerificationReference) (sharedtextbook.LearnerVerificationInput, error) {
	resolver.calls++
	return sharedtextbook.LearnerVerificationInput{}, context.Canceled
}

func TestLearnerRunRouteAcceptsOnlyABoundedReference(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resolver := &rejectingResolver{}
	router := gin.New()
	RegisterLearnerVerificationRoutes(router, domain.Runner{Resolver: resolver})
	reference := sharedtextbook.LearnerVerificationReference{Format: sharedtextbook.LearnerVerificationReferenceFormat, RunID: uuid.NewString(), WorkspaceID: uuid.NewString(), ObjectiveID: uuid.NewString(), AttemptID: uuid.NewString(), InputHash: "sha256:" + strings.Repeat("a", 64), ClaimToken: strings.Repeat("A", 43)}
	body, _ := json.Marshal(reference)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/internal/v1/learner-verification/runs", bytes.NewReader(body)))
	require.Equal(t, 200, response.Code)
	require.Equal(t, 1, resolver.calls)
	require.NotContains(t, response.Body.String(), reference.ClaimToken)

	malformed := httptest.NewRecorder()
	router.ServeHTTP(malformed, httptest.NewRequest(http.MethodPost, "/internal/v1/learner-verification/runs", strings.NewReader(`{"command":"go test"}`)))
	require.Equal(t, 400, malformed.Code)
	require.Equal(t, 1, resolver.calls)
}
