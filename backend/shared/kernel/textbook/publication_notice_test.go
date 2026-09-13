package textbook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPublicationNoticesFreezeExactTextAndRejectDrift(t *testing.T) {
	draft := PublicationNoticeDraft{Title: "MIT 许可", Text: "Copyright fixture\nPermission fixture.\n", SourceURL: "https://example.org/LICENSE", PreparedBy: "用户委托 AI", SubjectRefs: []string{"code-artifact:00000000-0000-4000-8000-000000000001"}}
	notices, err := FreezePublicationNotices([]PublicationNoticeDraft{draft})
	require.NoError(t, err)
	require.Equal(t, draft.Text, notices[0].Text)
	require.NoError(t, notices[0].Validate())
	changed := notices[0]
	changed.Text += "changed"
	require.Error(t, changed.Validate())
	retry, err := FreezePublicationNotices([]PublicationNoticeDraft{draft})
	require.NoError(t, err)
	require.Equal(t, notices, retry)
	for _, mutate := range []func(*PublicationNoticeDraft){
		func(d *PublicationNoticeDraft) { d.SubjectRefs = []string{"code-artifact:../../outside"} },
		func(d *PublicationNoticeDraft) { d.SourceURL = "file:///private/data" },
		func(d *PublicationNoticeDraft) { d.SourceURL = "https://example.org/><script>" },
		func(d *PublicationNoticeDraft) { d.PreparedBy = "reviewer\x00hidden" },
		func(d *PublicationNoticeDraft) { d.Text = strings.Repeat("x", 24001) },
		func(d *PublicationNoticeDraft) { d.PreparedBy = "" },
	} {
		copy := draft
		mutate(&copy)
		_, err := FreezePublicationNotices([]PublicationNoticeDraft{copy})
		require.Error(t, err)
	}
}
