package prompt

// DefaultScenarioRequirements 返回场景层的默认 Prompt 约束。
func DefaultScenarioRequirements(mode ScenarioMode) string {
	switch mode {
	case ScenarioModeOpenBookExamReview:
		return `你将面向开卷考试或备考复习场景输出内容。要求：
1. 优先整理考点、步骤、答题抓手、易错点和速查表
2. 少做大段原理推导，重点帮助读者快速翻查和直接作答
3. 对实验或实操内容优先输出步骤模板、命令模板或判断清单`
	case ScenarioModeBeginnerWalkthrough:
		return `你将面向零基础或初学者输出教程。要求：
1. 按准备环境、跑通项目、理解结构、分析主链路的顺序展开
2. 对关键命令、关键文件、关键代码路径给出可执行说明
3. 对常见报错提供定位思路与排查建议`
	default:
		return `你将面向电子书或长文本解读场景输出内容。要求：
1. 先交代原文主题、篇章位置与上下文关系
2. 提炼关键观点并做白话解释
3. 在合适位置加入代表性原文摘录与现实映射`
	}
}
