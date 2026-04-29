package service

import "testing"

func TestBuildObsidianBody_RemovesLeadingH1(t *testing.T) {
	t.Parallel()

	body := "# 原标题\n\n正文内容\n"
	got := buildObsidianBody(body)
	want := "正文内容\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildObsidianBody_DemotesRemainingH1(t *testing.T) {
	t.Parallel()

	body := "# 原标题\n\n# 副标题\n\n正文内容\n"
	got := buildObsidianBody(body)
	want := "## 副标题\n\n正文内容\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestBuildObsidianBody_LeavesCodeFenceH1Untouched(t *testing.T) {
	t.Parallel()

	body := "# 原标题\n\n```md\n# inside fence\n```\n\n# 副标题\n\n正文\n"
	got := buildObsidianBody(body)
	want := "```md\n# inside fence\n```\n\n## 副标题\n\n正文\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

