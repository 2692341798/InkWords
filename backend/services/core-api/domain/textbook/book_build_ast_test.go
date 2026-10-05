package textbook

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestFrozenBookBuildASTPinsApprovedContentAndAssets(t *testing.T) {
	revisionOne, revisionTwo := uuid.New(), uuid.New()
	build, err := newFrozenBookBuildAST("Gin 自学教材", time.Unix(1, 0), []bookBuildChapterSource{
		{ChapterID: uuid.New(), RevisionID: revisionTwo, SortOrder: 2, Title: "实践", Markdown: "# 实践", ContentHash: strings.Repeat("b", 64)},
		{ChapterID: uuid.New(), RevisionID: revisionOne, SortOrder: 1, Title: "概念", Markdown: "# 概念", ContentHash: strings.Repeat("a", 64)},
	}, []bookBuildAssetSource{{ID: uuid.New(), RevisionID: revisionOne, StableRef: "inkwords-asset:route", ContentHash: "sha256:" + strings.Repeat("c", 64), AltText: "路由图"}})

	require.NoError(t, err)
	require.Len(t, build.Chapters, 2)
	require.Equal(t, "概念", build.Chapters[0].Title)
	require.Equal(t, "# 概念", build.Chapters[0].Markdown)
	require.Equal(t, "sha256:"+strings.Repeat("a", 64), build.Chapters[0].ContentHash)
	require.Equal(t, "inkwords-asset:route", build.Chapters[0].Assets[0].StableRef)
	require.Empty(t, build.Chapters[1].Assets)
}
