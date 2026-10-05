package textbook

import (
	"fmt"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// ToolRecommendation selects a human-operated demonstration tool without claiming control over it.
type ToolRecommendation = sharedtextbook.DemonstrationToolRecommendation

// RecommendTool chooses based on the technology and observation goal, not user preference alone.
func RecommendTool(stack, goal string) (ToolRecommendation, error) {
	switch stack {
	case "go":
		return ToolRecommendation{Primary: "GoLand", Alternative: "VS Code + Go 扩展", Reason: "GoLand 适合展示跳转、调用层级和调试变量；VS Code 是轻量替代。", ManualCaptureRequired: true}, nil
	case "csharp", "cpp":
		return ToolRecommendation{Primary: "Visual Studio 2022", Alternative: "VS Code", Reason: "需要展示断点、调用栈或内存窗口时，Visual Studio 2022 提供更完整的本地调试视图。", ManualCaptureRequired: true}, nil
	case "javascript", "typescript", "web":
		return ToolRecommendation{Primary: "浏览器开发者工具", Alternative: "VS Code", Reason: "网络、DOM、性能和控制台证据应优先来自浏览器开发者工具；源码编辑使用 VS Code。", ManualCaptureRequired: false}, nil
	default:
		return ToolRecommendation{}, fmt.Errorf("no supported demonstration tool for stack %q", stack)
	}
}
