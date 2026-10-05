package textbook

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

func TestSampleGenerationRequestRejectsInvalidOrMismatchedAudience(t *testing.T) {
	request := sampleGenerationRequest(t)
	request.Audience = sharedtextbook.AudienceLevel("unknown")
	require.ErrorContains(t, request.Validate(), "audience level")
	request.Audience = sharedtextbook.AudienceProgramming
	require.ErrorContains(t, request.Validate(), "book contract does not match")
}

func TestFixedGinFixtureUsesAudienceSpecificProseInsteadOfRelabeling(t *testing.T) {
	markers := map[sharedtextbook.AudienceLevel]string{
		sharedtextbook.AudienceProgramming:   "假设你已经会读 Go 函数、map 和单元测试",
		sharedtextbook.AudienceStackFamiliar: "假设你已经能创建 Gin 路由并使用调试器",
	}
	for audience, marker := range markers {
		request := sampleGenerationRequest(t)
		request.Audience = audience
		request.BookContract.Reader.Audience = audience
		request.GenerationTarget = sharedtextbook.SampleGenerationTarget{ProviderName: sharedtextbook.SampleFixtureProviderName, ModelName: sharedtextbook.SampleFixtureModelName}
		generation, err := (FakeSampleGenerator{}).Generate(context.Background(), request)
		require.NoError(t, err)
		require.Equal(t, audience, generation.Chapter.Audience)
		require.Contains(t, generation.Chapter.Markdown, marker)
		require.NotContains(t, generation.Chapter.Markdown, "不要求你先会 Go 或 Gin")
	}
}

func TestSampleGenerationRequestRejectsUnresolvedSourceConflicts(t *testing.T) {
	request := sampleGenerationRequest(t)
	request.ClaimCandidates = []ClaimCandidate{
		{Key: "route-stage", Text: "启动登记", SourceRole: sharedtextbook.SourceRolePrimary, EvidenceID: "primary"},
		{Key: "route-stage", Text: "每个请求登记", SourceRole: sharedtextbook.SourceRoleOfficial, EvidenceID: "official"},
	}

	require.ErrorContains(t, request.Validate(), "resolve source conflicts before generation: route-stage")
}
