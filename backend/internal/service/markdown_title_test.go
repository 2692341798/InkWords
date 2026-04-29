package service

import "testing"

func TestApplyMarkdownTitle_ReplacesLeadingH1(t *testing.T) {
	t.Parallel()

	body := "# 原标题\n\n正文内容\n"
	got := applyMarkdownTitle("新标题", body)
	want := "# 新标题\n\n正文内容\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestApplyMarkdownTitle_ReplacesLeadingH1_AfterBlankLines(t *testing.T) {
	t.Parallel()

	body := "\n\n# 原标题\n\n正文内容\n"
	got := applyMarkdownTitle("新标题", body)
	want := "# 新标题\n\n正文内容\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestApplyMarkdownTitle_InsertsH1WhenMissing(t *testing.T) {
	t.Parallel()

	body := "## 二级标题\n\n正文内容\n"
	got := applyMarkdownTitle("新标题", body)
	want := "# 新标题\n\n## 二级标题\n\n正文内容\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestApplyMarkdownTitle_DemotesExtraH1(t *testing.T) {
	t.Parallel()

	body := "# 原标题\n\n# 副标题\n\n正文内容\n"
	got := applyMarkdownTitle("新标题", body)
	want := "# 新标题\n\n## 副标题\n\n正文内容\n"
	if got != want {
		t.Fatalf("unexpected markdown:\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}
