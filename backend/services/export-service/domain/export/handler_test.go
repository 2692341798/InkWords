package export

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWriteSeriesZip_CreatesValidArchive(t *testing.T) {
	parentID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	blogs := []Blog{
		{Title: "系列入门", Content: "欢迎阅读", ParentID: nil},
		{Title: "第一节", Content: "正文 1", ParentID: &parentID, ChapterSort: 1},
		{Title: "第二节", Content: "正文 2", ParentID: &parentID, ChapterSort: 2},
	}

	var buf bytes.Buffer
	err := writeSeriesZip(&buf, blogs)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Equal(t, 3, len(reader.File))

	names := make([]string, len(reader.File))
	for i, f := range reader.File {
		names[i] = f.Name
	}
	require.Equal(t, "系列入门.md", names[0])
	require.Equal(t, "01-第一节.md", names[1])
	require.Equal(t, "02-第二节.md", names[2])

	rc, err := reader.File[0].Open()
	require.NoError(t, err)
	defer rc.Close()
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Contains(t, string(body), "# 系列入门")
	require.Contains(t, string(body), "欢迎阅读")
}

func TestWriteSeriesZip_UsesPlaceholderForEmptyTitle(t *testing.T) {
	blogs := []Blog{
		{Title: "", Content: "无标题内容", ParentID: nil},
	}

	var buf bytes.Buffer
	err := writeSeriesZip(&buf, blogs)
	require.NoError(t, err)

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	require.NoError(t, err)
	require.Equal(t, 1, len(reader.File))
	require.Equal(t, "未命名_0.md", reader.File[0].Name)
}

func TestSeriesParentTitle_UsesFirstBlogTitle(t *testing.T) {
	blogs := []Blog{
		{Title: "Go 入门指南"},
		{Title: "第一节"},
	}
	require.Equal(t, "Go 入门指南", seriesParentTitle(blogs))
}

func TestSeriesParentTitle_FallsBackToSeries(t *testing.T) {
	require.Equal(t, "series", seriesParentTitle(nil))
	require.Equal(t, "series", seriesParentTitle([]Blog{{Title: ""}}))
}
