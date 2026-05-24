# InkWords 场景切换与 Prompt 分层设计

## 1. 背景与目标
- 当前系统已经支持 `article_style`（如通用技术博客、小白手把手、备考复习），但该能力本质上仍是“写作要求模板”，不足以表达“这次要解决什么任务”的业务语义。
- 现有上传与生成主链路已经较完整：`project/parse -> stream/analyze -> stream/generate`，并支持文件、ZIP 课件包和 Git 仓库三类主要来源。
- 当前问题在于，不同来源内容虽然进入了统一链路，但后端提示词还没有针对“电子书解读”“开卷考试复习资料”“小白手把手教程”做稳定切换，导致输出容易跑偏。
- 本设计的目标是在不推翻现有架构的前提下，引入一层清晰、可扩展、可配置的“场景模式（scenario mode）”，让生成结果能真正贴合任务目标。

## 2. 范围与非范围
### 2.1 本次范围（做）
- 引入一级业务概念：`scenario_mode`。
- 支持三个首批场景：
  - `ebook_interpretation`：电子书解读
  - `open_book_exam_review`：开卷考试复习资料
  - `beginner_walkthrough`：面向小白的手把手教程
- 将场景能力接入 `stream/analyze` 与 `stream/generate` 两个关键阶段。
- 设计 Prompt 分层组装机制，避免继续把所有约束堆进一段大字符串。
- 保留并兼容现有 `article_style` 与用户 Prompt 覆盖能力。
- 给出前端交互、后端数据结构、兼容策略、验证方案。

### 2.2 非范围（不做）
- 本次不新增第四类及以上场景。
- 本次不设计复杂的自动分类器，不做模型级自动识别。
- 本次不直接改造数据库表结构；若后续需要持久化用户级场景模板，再作为第二阶段演进。
- 本次不扩展新的文件解析格式（如 `epub/mobi/azw`）。
- 本次不设计完整视觉稿，仅定义交互结构与字段语义。

## 3. 设计原则
- **任务目标优先**：场景首先表达“要产出什么类型的内容”，而不是“上传了什么文件”。
- **复用现有链路**：解析、SSE、Map-Reduce、章节并发生成继续沿用，不另起新流程。
- **分层注入 Prompt**：把固定系统约束、场景约束、风格约束、用户覆盖解耦。
- **兼容旧接口**：旧请求未传 `scenario_mode` 时仍可正常生成。
- **用户显式决策优先**：场景以用户手动选择为主，系统默认仅作兜底或推荐。

## 4. 方案选择
- 采用 **方案 B：引入独立 `scenario_mode`，并与 `article_style` 并存**。
- 不采用“只增加几个 `article_style` 枚举”的原因：
  - 场景差异不仅是文风不同，还包括大纲拆解逻辑、章节组织方式、内容重点和读者假设。
  - 如果继续复用 `article_style` 承载业务语义，会让后续扩展越来越难维护。
- 不采用“完全自动推断场景”的原因：
  - 场景本质上是创作目标，而不是文件扩展名。
  - 同一份素材可能可以被写成不同形态的内容，最终应由用户决定。

## 5. 核心概念设计
### 5.1 概念边界
- `scenario_mode`：决定“这次生成任务属于哪一种内容场景”。
- `article_style`：决定“在该场景下，内容采用什么写法和细节偏好呈现”。
- `source_type`：决定“内容来源于文件还是 Git 仓库”，继续承担链路分流职责。

### 5.2 概念关系
- `source_type` 管输入来源。
- `scenario_mode` 管任务目标。
- `article_style` 管表达风格。

推荐理解方式：
- 文件上传 + `ebook_interpretation` + `general`
- ZIP 课件 + `open_book_exam_review` + `exam_review`
- Git 仓库 + `beginner_walkthrough` + `beginner_tutorial`

### 5.3 首批场景定义
#### 5.3.1 `ebook_interpretation`
- 适用输入：电子书 PDF、文史类长文档、概念性材料。
- 目标：帮助读者读懂原文结构、核心观点、上下文关系与现实意义。
- 强调输出：
  - 篇章结构拆解
  - 关键观点与概念解释
  - 原文摘录与白话解释
  - 现实案例或现代映射
- 明确避免：
  - 硬套“技术教程”口吻
  - 不必要的代码化表达
  - 过度工程化的模块分层叙述

#### 5.3.2 `open_book_exam_review`
- 适用输入：系列课件、实验指导、课程讲义、考试范围资料。
- 目标：生成适合开卷考试快速查阅的复习资料，聚焦“会做题、会操作、会作答”。
- 强调输出：
  - 高频考点清单
  - 操作步骤模板
  - 答题抓手
  - 易错点与对比表
  - 速查表与最短定位路径
- 明确弱化：
  - 大段理论推导
  - 长篇背景介绍
  - 与考试无关的延伸展开

#### 5.3.3 `beginner_walkthrough`
- 适用输入：大型 Git 仓库、源码目录、官方教程、项目文档。
- 目标：让零基础或初学者能跟着操作、跑通环境、理解结构、看懂关键链路。
- 强调输出：
  - 环境准备与启动路径
  - 目录结构与模块职责
  - 核心调用链与关键代码位置
  - 动手步骤、命令、排错提示
  - 抽象概念的生活化类比
- 明确要求：
  - 讲清楚“为什么这样设计”
  - 但解释必须落到具体路径、具体步骤、具体文件

## 6. Prompt 分层设计
### 6.1 问题
- 当前实现中，默认模板与用户覆盖虽然已经抽离，但生成阶段仍然以“单段 requirements 文本 + 临时拼接”的方式使用。
- 这种方式很难表达“系统固定约束”“场景专属约束”“风格偏好”“用户个性化覆盖”之间的边界。

### 6.2 分层方案
- 将 Prompt 组装拆成以下四层：
  1. **System Base Layer**
  2. **Scenario Layer**
  3. **Style Layer**
  4. **User Override Layer**

### 6.3 各层职责
#### 6.3.1 System Base Layer
- 放置不会被用户覆盖的硬约束，例如：
  - 输出语言为中文
  - Mermaid 禁止自定义样式
  - 必须结构化、逻辑清晰
  - 不得捏造不存在的实现细节
  - 对源码场景优先引用真实文件与路径信息

#### 6.3.2 Scenario Layer
- 放置由 `scenario_mode` 决定的约束，例如：
  - 电子书解读强调原文结构和现实映射
  - 开卷复习强调步骤、答题模板、速查性
  - 小白教程强调环境、路径、命令与排错
- 场景层应同时作用于：
  - 大纲生成
  - 单篇生成
  - 系列章节生成
  - 导读生成

#### 6.3.3 Style Layer
- 继续复用现有 `article_style` 体系，但降级为“写作表现层”。
- 示例：
  - `general`：偏通用、均衡
  - `beginner_tutorial`：更口语化、更细步骤
  - `exam_review`：更强调关键词、表格、清单

#### 6.3.4 User Override Layer
- 保留当前用户级 Prompt 覆盖逻辑。
- 第一阶段沿用当前结构，对 `article_style` 做覆盖。
- 第二阶段可升级为：
  - `scenario_defaults`
  - `style_overrides`

### 6.4 组装顺序
- 建议后端统一通过 `PromptRequirementsService` 或其演进版本进行组装：

```text
finalRequirements =
  SystemBase
  + ScenarioInstruction(scenario_mode)
  + StyleInstruction(article_style)
  + UserOverride(user_id, scenario_mode, article_style)
```

- 最终在生成器中再与任务上下文拼接：

```text
finalPrompt =
  SourceContext
  + OutlineContext
  + ChapterContext
  + finalRequirements
```

## 7. 后端设计
### 7.1 枚举建议
- 新增 `internal/prompt/scenario_mode.go`：

```go
package prompt

type ScenarioMode string

const (
    ScenarioModeEbookInterpretation ScenarioMode = "ebook_interpretation"
    ScenarioModeOpenBookExamReview  ScenarioMode = "open_book_exam_review"
    ScenarioModeBeginnerWalkthrough ScenarioMode = "beginner_walkthrough"
)
```

### 7.2 Prompt 包职责扩展
- `internal/prompt/article_style.go`：继续承载文章风格枚举。
- `internal/prompt/default_requirements.go`：可拆分为：
  - `default_style_requirements.go`
  - `default_scenario_requirements.go`
- 新增统一组装入口，例如：
  - `BuildRequirements(scenario ScenarioMode, style ArticleStyle) string`

### 7.3 Service 层改造
- `PromptRequirementsService.Resolve` 现状只接收 `style`。
- 建议演进为：

```go
Resolve(ctx context.Context, userID uuid.UUID, scenario prompt.ScenarioMode, style prompt.ArticleStyle) (string, error)
```

- 为什么这样做：
  - 让调用方只关心业务输入，不关心 Prompt 拼装细节。
  - 避免在 `generator.go`、`decomposition_generate.go`、`decomposition_generate_outline.go` 中各自拼一套场景逻辑。

### 7.4 Analyze 阶段接入点
- `stream/analyze` 不应只负责“按来源切分”，还应根据 `scenario_mode` 改变大纲拆解方式。
- 建议策略：
  - `ebook_interpretation`：按篇章、思想主线、概念主题拆解
  - `open_book_exam_review`：按考点、题型、实验步骤、知识块拆解
  - `beginner_walkthrough`：按学习路径、运行路径、模块职责拆解

### 7.5 Generate 阶段接入点
- 单篇生成：`GeneratorService.GenerateBlogStream`
- 系列章节生成：`DecompositionService.GenerateSeries`
- 系列导读生成：`decomposition_generate_intro.go`
- 三处都需要接入 `scenario_mode`，否则会出现“大纲像复习资料，正文却像通用博客”的割裂。

### 7.6 DTO 与接口契约
- 建议在 `backend/internal/domain/stream/dto.go` 的请求模型中新增字段：

```go
ScenarioMode string `json:"scenario_mode"`
```

- 涉及接口：
  - `POST /api/v1/stream/analyze`
  - `POST /api/v1/stream/generate`

## 8. 前端交互设计
### 8.1 交互原则
- 场景必须显式可见、可理解、可切换。
- 所有展示文案使用中文。
- 不增加复杂流程，保持在现有生成器页完成选择。

### 8.2 推荐交互结构
- 在现有生成器输入区域增加“创作场景”单选卡片组。
- 三张卡片建议文案：
  - **电子书解读**
    - 适合经典著作、理论文本、长文档解读
  - **开卷复习**
    - 适合系列课件、实验指导、考试资料整理
  - **小白教程**
    - 适合源码仓库、项目教程、官方文档拆解

### 8.3 默认值策略
- 以用户手动选择为主。
- 可选增强：根据输入来源预选默认值，但不锁死用户选择。
- 默认映射建议：
  - 上传普通文档文件：默认 `ebook_interpretation`
  - 上传 ZIP：默认 `open_book_exam_review`
  - Git 仓库：默认 `beginner_walkthrough`

### 8.4 状态流转
- `parse` 之后保留 `scenario_mode` 到前端状态。
- `analyze` 请求显式发送 `scenario_mode`。
- `generate` 请求继续沿用同一值，避免两阶段不一致。

## 9. 兼容策略
### 9.1 请求兼容
- 若前端新版本传入 `scenario_mode`，后端按新逻辑执行。
- 若旧前端未传入，则后端兜底推断默认值。

### 9.2 老字段兼容
- `article_style` 保留，不做破坏性移除。
- 老的用户 Prompt 覆盖结构继续有效。
- 第一阶段禁止让 `scenario_mode` 成为必填，以避免旧客户端报错。

### 9.3 渐进迁移
- 阶段一：仅引入后端枚举、默认 Prompt、DTO 字段和前端选择器。
- 阶段二：大纲生成按场景差异化。
- 阶段三：用户中心支持按场景配置模板。
- 阶段四：评测与推荐逻辑增强。

## 10. 示例 Prompt 策略
### 10.1 电子书解读
- 典型要求：
  - 先交代原文主题与上下文位置
  - 用白话解释关键观点
  - 提取代表性原文并进行现代映射
  - 输出结构要适合连续阅读，而不是答题速查

### 10.2 开卷复习
- 典型要求：
  - 优先抽取考点、步骤、公式、定义、实验流程
  - 用表格、清单、口诀、答题模板组织内容
  - 面向“考试时快速翻找”的阅读场景
  - 避免无关背景与长篇理论展开

### 10.3 小白教程
- 典型要求：
  - 假设读者基础薄弱
  - 按“准备环境 -> 跑通项目 -> 看结构 -> 抓主链路 -> 深入模块”的路径展开
  - 对关键代码和命令提供上下文解释
  - 对常见报错给出定位和排查建议

## 11. 风险与边界
- 风险一：`scenario_mode` 与 `article_style` 边界不清，导致前后端重复配置。
  - 应对：在接口和文档中明确“场景管任务目标，风格管呈现方式”。
- 风险二：只在生成阶段切场景，不在分析阶段切场景，会导致大纲与正文不一致。
  - 应对：Analyze 与 Generate 同步接入。
- 风险三：场景默认值过于强势，用户误以为不可修改。
  - 应对：UI 上明确“默认推荐，可手动切换”。
- 风险四：场景提示词写得过硬，导致跨场景素材适配能力变差。
  - 应对：Prompt 中保留“若素材不完全匹配，则输出最接近目标的结构化结果”这一柔性约束。

## 12. 测试与验收
### 12.1 单元测试
- `ScenarioMode` 默认值与非法值兜底逻辑正确。
- `PromptRequirementsService.Resolve` 能正确合并场景层与风格层。
- 未传 `scenario_mode` 时，默认值策略符合预期。

### 12.2 集成验证
- 上传《孙子兵法》类 PDF，选择“电子书解读”，输出不再偏技术博客。
- 上传 ZIP 课件包，选择“开卷复习”，输出明显偏向步骤、答题模板、速查表。
- 输入大型 Git 仓库，选择“小白教程”，输出明显偏向环境、结构、主链路、排错。

### 12.3 对比验证
- 对同一份输入分别使用三个场景生成，结果差异应清晰可见：
  - 标题与章节结构不同
  - 内容重点不同
  - 组织形式不同
  - 读者假设不同

### 12.4 验收标准
- 用户能明确选择场景，且理解三者差异。
- 生成结果能稳定体现所选场景，而不是只改几句措辞。
- 旧接口与旧客户端不回归。
- 现有 `parse -> analyze -> generate` 主链路保持稳定。

## 13. 实施顺序建议
1. 后端新增 `scenario_mode` 枚举与默认 Prompt。
2. 扩展 DTO、Analyze/Generate 请求契约。
3. 改造 `PromptRequirementsService` 为统一组装入口。
4. 让大纲生成与正文生成同时接入场景。
5. 前端生成器增加“创作场景”选择器并透传字段。
6. 增加三组回归样例进行人工验收与日志对比。

## 14. 回滚策略
- 保留现有 `article_style` 主路径不删除。
- 新逻辑通过 `scenario_mode` 缺省兜底，不阻断旧客户端。
- 若首版上线后质量不稳定，可临时关闭前端场景入口，仅保留后端默认映射。

## 15. 关联约束与引用
- 参考：`wiki/sources/孙子兵法`
- 参考：`wiki/sources/muduo网络库源码深度解析系列`
- 参考：`wiki/sources/机器学习入门：从Python基础到模型部署`
- 参考：`wiki/concepts/项目整体架构与设计理念`
- 参考：`.trae/documents/InkWords_PRD.md`
- 参考：`.trae/documents/InkWords_Architecture.md`
- 参考：`.trae/documents/InkWords_API.md`
