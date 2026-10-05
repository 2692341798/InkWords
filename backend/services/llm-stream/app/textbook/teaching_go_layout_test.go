package textbook

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTeachingGoLayoutUsesDeclarationsNotCommentsOrStrings(t *testing.T) {
	markdown := "```go\npackage main\n// func TestExample is discussed here, not declared.\nfunc main() { println(\"func Test is only text\") }\n```\n\n```go\npackage main\nimport \"testing\"\nfunc TestExample(t *testing.T) { main() }\n```"
	require.Empty(t, validateTeachingGoSources(markdown))
}

func TestTeachingGoLayoutReportsMissingEntrypointWithoutSourceLeak(t *testing.T) {
	markdown := "```go\npackage main\n// func main() is not implemented; private-marker\nfunc helper() {}\n```\n\n```go\npackage main\nimport \"testing\"\nfunc TestExample(t *testing.T) { helper() }\n```"
	failures := validateTeachingGoSources(markdown)
	require.Contains(t, failures, "teaching_go_file_layout_invalid: missing main.go")
	require.NotContains(t, strings.Join(failures, ";"), "private-marker")
}

func TestTeachingGoLayoutRejectsMixedFilesAndMissingRealTests(t *testing.T) {
	for name, markdown := range map[string]string{
		"mixed":             "```go\npackage main\nfunc main() {}\nfunc TestExample() {}\n```\n```go\npackage main\nfunc helper() {}\n```",
		"comment test":      "```go\npackage main\nfunc main() {}\n```\n```go\npackage main\n// func TestExample() {}\nfunc helper() {}\n```",
		"method entrypoint": "```go\npackage main\ntype Router struct{}\nfunc (r Router) main() {}\n```\n```go\npackage main\nfunc TestExample() {}\n```",
	} {
		t.Run(name, func(t *testing.T) {
			require.NotEmpty(t, validateTeachingGoSources(markdown))
		})
	}
}
