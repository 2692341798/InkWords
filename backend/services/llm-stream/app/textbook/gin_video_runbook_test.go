package textbook

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func runbookManuscript() string {
	return "**代码来源：教学实现（main.go）。** 非生产用途，省略并发处理。\n\n```go\npackage main\n\ntype routeNode struct{}\nfunc main() {}\n```\n\n**代码来源：教学实现测试（main_test.go）。**\n\n```go\npackage main\nimport \"testing\"\n// TestImaginary is not a test declaration.\nfunc TestMethodSpecific(t *testing.T) {}\nfunc TestSharedAPINode(t *testing.T) {}\n```\n"
}

func TestGinRunbookBindsExactTeachingFilesAndDeclarations(t *testing.T) {
	markdown := runbookManuscript()
	book, err := BuildGinSampleVideoRunbookProjection(markdown, digest(markdown))
	require.NoError(t, err)
	require.Contains(t, book.Steps[0].Input, fmt.Sprintf("main.go sha256:%x", sha256.Sum256([]byte("package main\n\ntype routeNode struct{}\nfunc main() {}\n"))))
	require.NotContains(t, book.Steps[0].Input, "sha256:sha256:")
	require.Contains(t, book.Steps[0].Input, "routeNode（第 3 行）")
	require.Contains(t, book.Steps[1].Input, "TestMethodSpecific（第 4 行）")
	require.Contains(t, book.Steps[1].Input, "TestSharedAPINode（第 5 行）")
	require.NotContains(t, book.Steps[1].Input, "TestImaginary")
	require.NotContains(t, book.Steps[1].Input, "TestPathOnlyBaseline")
	require.True(t, book.ManualCapturePending)
	changed := strings.ReplaceAll(markdown, "TestMethodSpecific", "TestNewMethodBoundary")
	updated, err := BuildGinSampleVideoRunbookProjection(changed, digest(changed))
	require.NoError(t, err)
	require.Contains(t, updated.Steps[1].Input, "TestNewMethodBoundary")
	require.NotContains(t, updated.Steps[1].Input, "TestMethodSpecific")
}

func TestGinRunbookRejectsHashMismatchWithoutReturningDirections(t *testing.T) {
	book, err := BuildGinSampleVideoRunbookProjection(runbookManuscript(), strings.Repeat("0", 64))
	require.Error(t, err)
	require.Empty(t, book.Steps)
}

func TestGinRunbookPreservesCandidateWithUnusableTeachingProjection(t *testing.T) {
	for _, markdown := range []string{
		"No teaching files yet.",
		runbookManuscript() + runbookManuscript(),
		strings.ReplaceAll(runbookManuscript(), "func main()", "func main("),
	} {
		book, err := BuildGinSampleVideoRunbookProjection(markdown, digest(markdown))
		require.NoError(t, err)
		require.NoError(t, book.Validate())
		require.Len(t, book.Steps, 1)
		require.Equal(t, "source-review-pending", book.Steps[0].CapturePoint)
		require.True(t, book.ManualCapturePending)
		require.NotContains(t, book.Steps[0].Input, "TestMethodSpecific")
	}
}
