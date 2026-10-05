package export

import (
	"encoding/json"
	"fmt"
	"strings"

	shared "inkwords-backend/shared/kernel/textbook"
)

// frozenBookRunbookFiles emits companion material from the frozen manifest,
// never from the currently approved chapter or the generation service.
func frozenBookRunbookFiles(raw json.RawMessage, book shared.CanonicalBookAST) ([]BookPackageFile, error) {
	snapshot, err := shared.ReadBookVideoRunbooks(raw)
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	out.WriteString("# 冻结视频操作教案\n\n本文件是冻结母稿的配套投影，不是独立编辑的教材正文。教案存在不代表画面已采集、程序已运行或出版审校已完成。\n\n")
	if snapshot == nil {
		out.WriteString("该历史构建未冻结视频教案，内容不可用。不会从当前章节补取；如需教案，请从已批准修订创建新的冻结构建。\n")
	} else {
		if err := snapshot.ValidateAgainstBook(book); err != nil {
			return nil, err
		}
		for _, chapter := range snapshot.Chapters {
			fmt.Fprintf(&out, "## 章节 %s\n\n批准修订：`%s`\n\n母稿哈希：`%s`\n\n教案哈希：`%s`\n\n", chapter.ChapterID, chapter.RevisionID, chapter.ContentHash, chapter.RunbookHash)
			runbook := chapter.VideoRunbook
			if runbook == nil {
				out.WriteString("该批准修订没有视频教案；保持缺失状态。\n\n")
				continue
			}
			fmt.Fprintf(&out, "观察目标：%s\n\n工具：%s；备选：%s。%s\n\n验证状态：%s；采集待完成：%t。\n\n", runbook.ObservationGoal, runbook.Recommendation.Primary, runbook.Recommendation.Alternative, runbook.Recommendation.Reason, runbook.VerificationStatus, runbook.ManualCapturePending)
			for index, step := range runbook.Steps {
				fmt.Fprintf(&out, "### %d. %s\n\n", index+1, step.Action)
				for _, field := range [][2]string{{"工具与版本", step.Tool + " · " + step.ToolVersion}, {"起始状态", step.StartState}, {"快捷键或菜单", step.ShortcutOrMenu}, {"输入", step.Input}, {"预期画面", step.ExpectedView}, {"旁白", step.Narration}, {"截图点", step.CapturePoint}, {"失败恢复", step.Recovery}, {"完成信号", step.CompletionSignal}} {
					fmt.Fprintf(&out, "%s：%s\n\n", field[0], field[1])
				}
			}
			out.WriteString("### 采集清单\n\n")
			for _, item := range runbook.CaptureChecklist {
				fmt.Fprintf(&out, "- %s\n", item)
			}
			out.WriteString("\n")
		}
	}
	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return nil, err
	}
	return []BookPackageFile{{Path: "projections/video-runbooks.json", Content: payload}, {Path: "projections/video-runbooks.md", Content: []byte(out.String())}}, nil
}
