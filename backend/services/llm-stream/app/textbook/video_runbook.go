package textbook

import (
	"fmt"
	"strings"

	sharedtextbook "inkwords-backend/shared/kernel/textbook"
)

// RunbookStep is one manually recordable action for a second-screen teaching script.
// It never claims that InkWords can drive a local IDE or verify a visual observation.
type RunbookStep = sharedtextbook.VideoRunbookStep

// BuildRunbook provides goal-specific, human-executable runbook steps. A goal
// changes the observation evidence, rather than only changing generic narration.
func BuildRunbook(stack, goal string) ([]RunbookStep, error) {
	recommendation, err := RecommendTool(stack, goal)
	if err != nil {
		return nil, err
	}
	switch recommendation.Primary {
	case "Visual Studio 2022":
		return visualStudioRunbook(goal), nil
	case "GoLand":
		return goLandRunbook(goal), nil
	case "浏览器开发者工具":
		return browserDevToolsRunbook(goal), nil
	default:
		return nil, fmt.Errorf("no runbook template for %q", recommendation.Primary)
	}
}

// BuildVideoRunbookProjection binds a human-operated recording guide to a
// manuscript revision. The projection stays unverified until the author has
// captured the listed observations with the stated tool.
func BuildVideoRunbookProjection(stack, goal string) (sharedtextbook.VideoRunbookProjection, error) {
	recommendation, err := RecommendTool(stack, goal)
	if err != nil {
		return sharedtextbook.VideoRunbookProjection{}, err
	}
	steps, err := BuildRunbook(stack, goal)
	if err != nil {
		return sharedtextbook.VideoRunbookProjection{}, err
	}
	checklist := []string{
		"录制前记录实际工具名称、完整版本和操作系统。",
		"逐步录下起始状态、输入和预期画面；保留失败恢复动作是否发生。",
		"将截图或录屏片段与每一步的 CapturePoint 对应，并区分观察到的事实与旁白中的解释。",
	}
	projection := sharedtextbook.VideoRunbookProjection{
		Format:               "inkwords.video-runbook.v1",
		Stack:                stack,
		ObservationGoal:      goal,
		Recommendation:       recommendation,
		Steps:                steps,
		CaptureChecklist:     checklist,
		ManualCapturePending: true,
		VerificationStatus:   sharedtextbook.ArtifactStatusUnverified,
	}
	if err := projection.Validate(); err != nil {
		return sharedtextbook.VideoRunbookProjection{}, err
	}
	return projection, nil
}

func visualStudioRunbook(goal string) []RunbookStep {
	observation, capture := debuggerObservation(goal, "调用堆栈", "局部变量")
	return []RunbookStep{
		{Tool: "Visual Studio 2022", ToolVersion: "17.x", StartState: "已关闭旧解决方案，主屏已开始录制。", Action: "打开教材生成的示例解决方案，并选中入口项目。", ShortcutOrMenu: "文件 → 打开 → 项目/解决方案", Input: "选择教学工件目录中的 .sln 文件", ExpectedView: "解决方案资源管理器显示入口项目和目标源文件。", Narration: "先确认我们操作的是教学工件，不是读者的真实项目。", CapturePoint: "项目树与入口项目", Recovery: "若打开了旧解决方案，使用 文件 → 关闭解决方案 后重新选择 .sln。", CompletionSignal: "入口项目以粗体显示，目标源文件已打开。"},
		{Tool: "Visual Studio 2022", ToolVersion: "17.x", StartState: "目标源文件已打开。", Action: "在关键调用处设置断点，并以调试方式启动。", ShortcutOrMenu: "F9 切换断点；F5 开始调试", Input: "在目标调用行按 F9，再按 F5", ExpectedView: "红色断点标记可见，执行在该行暂停。", Narration: "现在观察实际执行路径；不要把阅读源码时的猜测说成运行结果。", CapturePoint: "断点与当前执行行", Recovery: "若未停下，确认启动项目、调试配置和断点没有被禁用；必要时重建后再按 F5。", CompletionSignal: "黄色当前行箭头停在断点处。"},
		{Tool: "Visual Studio 2022", ToolVersion: "17.x", StartState: "调试器停在目标调用行。", Action: "单步进入关键调用，并打开与本节目标对应的观察窗口。", ShortcutOrMenu: "F11 单步进入；调试 → 窗口", Input: observation, ExpectedView: "进入被调用函数，并显示" + capture + "。", Narration: "把代码写了什么和运行时实际走到哪里放在同一画面中。", CapturePoint: capture, Recovery: "若单步进入了库代码，使用 Shift+F11 跳出，或把断点移到教学工件的调用行。", CompletionSignal: "录到目标函数、" + capture + "和本节旁白所指的状态。"},
	}
}

func goLandRunbook(goal string) []RunbookStep {
	observation, capture := debuggerObservation(goal, "调用栈", "变量")
	return []RunbookStep{
		{Tool: "GoLand", ToolVersion: "2024.1 或更新版本", StartState: "已关闭旧项目，主屏已开始录制。", Action: "打开教材示例目录，确认 Go SDK 和运行配置指向该教学工件。", ShortcutOrMenu: "File → Open；Run → Edit Configurations", Input: "选择示例目录；确认 package main 与 Go SDK", ExpectedView: "项目树、main.go 和 Go 运行配置可见。", Narration: "先固定示例和工具链，读者才能复现相同的观察。", CapturePoint: "项目树、SDK 与运行配置", Recovery: "若没有运行配置，右键 main.go 选择 Run；若 SDK 缺失，在 Settings → Go → GOROOT 选择已安装版本。", CompletionSignal: "运行配置名称与目标 main.go 一致。"},
		{Tool: "GoLand", ToolVersion: "2024.1 或更新版本", StartState: "目标文件和运行配置已确认。", Action: "在关键调用处设置断点后启动调试。", ShortcutOrMenu: "⌘F8 / Ctrl+F8 切换断点；Shift+F9 调试", Input: "在目标调用行设置断点，再按 Shift+F9", ExpectedView: "断点启用，调试工具窗口在目标行暂停。", Narration: "暂停的位置就是本段讲解要证明的实际入口。", CapturePoint: "断点与调试工具窗口", Recovery: "若未停下，确认运行的包、断点和构建标签；不要用旧终端进程替代调试配置。", CompletionSignal: "当前执行行停在教学工件。"},
		{Tool: "GoLand", ToolVersion: "2024.1 或更新版本", StartState: "调试器停在关键调用行。", Action: "进入关键调用并展示本节要求的运行时观察。", ShortcutOrMenu: "F7 / ⌘F7 单步进入；Debug 工具窗口", Input: observation, ExpectedView: "进入目标函数，并显示" + capture + "。", Narration: "屏幕上可见的是实际运行数据；底层解释要区分证据与推断。", CapturePoint: capture, Recovery: "若进入第三方实现而失去上下文，使用 Shift+F8 / ⇧F8 跳出，再把断点移动到下一处教学工件调用。", CompletionSignal: "目标函数、" + capture + "和相应旁白均已录制。"},
	}
}

func browserDevToolsRunbook(goal string) []RunbookStep {
	observation, capture := browserObservation(goal)
	return []RunbookStep{
		{Tool: "浏览器开发者工具", ToolVersion: "Chrome 或 Edge 当前稳定版", StartState: "浏览器仅打开教学示例页，主屏已开始录制。", Action: "打开开发者工具并切到本节需要的面板。", ShortcutOrMenu: "F12 或 Ctrl+Shift+I / ⌘⌥I", Input: observation, ExpectedView: "开发者工具停在" + capture + "对应面板。", Narration: "网页代码的结果要用浏览器原始观察记录，而不是凭页面猜测。", CapturePoint: capture + "面板初始状态", Recovery: "若快捷键被系统占用，从浏览器菜单 更多工具 → 开发者工具 打开。", CompletionSignal: "目标面板已打开且没有旧记录干扰。"},
		{Tool: "浏览器开发者工具", ToolVersion: "Chrome 或 Edge 当前稳定版", StartState: "目标面板已打开。", Action: "清空旧记录，执行教材中的最小交互或刷新页面。", ShortcutOrMenu: "Ctrl+L 后输入地址并回车；面板中的清除按钮", Input: "清空记录，再执行本节示例操作", ExpectedView: "只出现本次操作产生的" + capture + "记录。", Narration: "先清掉旧请求或旧日志，否则画面无法证明这次操作导致了什么。", CapturePoint: "单次操作后的" + capture, Recovery: "若记录没有出现，确认没有开启筛选条件，并用无痕窗口排除缓存或扩展影响。", CompletionSignal: "目标记录可选中并显示时间、状态或调用信息。"},
		{Tool: "浏览器开发者工具", ToolVersion: "Chrome 或 Edge 当前稳定版", StartState: "本次操作记录已出现。", Action: "展开关键记录，讲解观察到的字段及其边界。", ShortcutOrMenu: "单击记录；必要时切换 Headers/Response/Initiator", Input: "选择与教材证据一致的一条记录", ExpectedView: "关键字段和来源面板可见。", Narration: "只陈述画面直接支持的结论；性能、内存或调用关系之外的解释要标为推断。", CapturePoint: "展开后的关键字段", Recovery: "若找不到对应记录，重新清空并执行一次最小操作，不要截取历史记录。", CompletionSignal: "截图点、旁白和教材中的证据引用可以一一对应。"},
	}
}

func debuggerObservation(goal, stackLabel, variableLabel string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(goal)) {
	case "memory", "内存", "资源占用":
		return "打开诊断工具或内存/变量视图，并记录采样条件。", "内存/变量视图与采样条件"
	case "call_stack", "调用链", "调用栈":
		return "打开调用堆栈，选中当前帧并对照源码。", stackLabel
	default:
		return "打开局部变量和调用堆栈，选中当前帧。", variableLabel + "与" + stackLabel
	}
}

func browserObservation(goal string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(goal)) {
	case "performance", "性能", "资源占用":
		return "切到 Performance 面板并准备开始录制。", "性能时间线"
	case "console", "控制台":
		return "切到 Console 面板并保留日志级别筛选。", "控制台"
	default:
		return "切到 Network 面板并勾选 Preserve log（如本节需要重定向链）。", "网络请求"
	}
}
