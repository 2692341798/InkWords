# InkWords Hub Homepage Process-Oriented Design

## 1. Background
- The current homepage experience feels too flat because multiple functional modules are presented at the same visual level.
- Users can see many options, but the interface does not clearly answer what they should do first, what happens next, or which path is recommended.
- InkWords already has two major product directions:
  - `Generate Blog`: turn repositories or local documents into structured technical blog output.
  - `Knowledge Review`: turn stored knowledge into active recall and guided review.
- The new homepage should not behave like a feature catalog. It should behave like a decision center that guides users into one focused workflow.

## 2. Goal
- Redesign the homepage into a `decision center + guided workspace`.
- Make `Generate Blog` the default recommended primary path.
- Keep `Knowledge Review` visible as the secondary path.
- Reduce the flat feeling by ensuring that only one major workflow is expanded at a time.
- Produce a standalone HTML mockup that looks like a realistic product homepage and is suitable for discussing direction before implementation.

## 3. Scope
### 3.1 In Scope
- One standalone HTML homepage mockup.
- A unified hub page covering both:
  - `生成博客`
  - `知识复习`
- Clickable mock interactions for switching paths and changing key choices.
- Chinese UI copy throughout the mockup.
- A process-oriented layout that shows:
  - decision
  - flow preview
  - active workspace
  - summary and primary CTA
  - compact support/history area

### 3.2 Out of Scope
- No real backend integration.
- No real file upload, repository scan, or API requests.
- No production frontend component wiring yet.
- No attempt to merge all secondary features into the main hero area.
- No multi-page prototype in this step.

## 4. Product Positioning
- The hub homepage should feel like the real entry page of InkWords, not a planning artifact.
- Product hierarchy:
  1. First, help the user choose the right path.
  2. Second, explain the process in a compact and visual way.
  3. Third, present only the next relevant interaction area.
- `生成博客` remains the recommended starting action because it reflects the primary product entry.
- `知识复习` is framed as the continuation path that helps users internalize what they have already produced or stored.

## 5. Chosen Approach
- Chosen direction: `Guided Command Center`.
- Rejected alternatives:
  - `Split Decision Hero`: clearer than today, but still too easy to fall back into two equal cards with content below.
  - `Timeline Workflow Hub`: good for system storytelling, but weaker for first-use clarity.
- Why `Guided Command Center`:
  - It preserves the homepage nature.
  - It gives one recommended primary direction.
  - It supports both product paths without fully expanding both.
  - It turns the page into `choose -> understand -> act`.

## 6. Core UX Shift
- Old feeling: `many functions are available`.
- New feeling: `the system is guiding me through one chosen path`.

This shift is implemented through three rules:
- Only one major workflow panel is expanded at a time.
- Every selection updates a visible summary of what the user chose and what happens next.
- History and supporting modules move lower on the page so they support action instead of competing with it.

## 7. Information Architecture
- `Top`: hero and decision center.
- `Middle`: selected flow preview and active workspace.
- `Bottom`: compact support area with recent work and review history.

The page contains five main blocks:
1. `Hero`
2. `Decision Center`
3. `Flow Preview Strip`
4. `Active Workspace`
5. `Support Area`

## 8. Layout Design
### 8.1 Page Proportions
- Top 20%: hero + decision center
- Middle 50%: active workflow workspace
- Bottom 30%: support/history panels

### 8.2 Visual Tone
- Light, calm, minimal, and reading-friendly.
- Closer to Notion-style productivity than a dense dashboard.
- Strong hierarchy with one primary area at any time.
- Avoid large equal-weight card grids.

### 8.3 Workspace Structure
- Use a two-column workspace layout:
  - left: interactive decisions
  - right: summary, outcome preview, and primary CTA
- Why: the left side lets the user make choices, while the right side explains the consequence of those choices. This makes the interface feel process-oriented.

## 9. Hero Section
- Show product name and short positioning statement.
- Suggested value line:
  - `从资料到博客，从博客到复习，把知识变成可输出的成果`
- Add a helper sentence introducing the hub behavior:
  - `今天你想先完成哪一种任务？`

## 10. Decision Center
- Two large selectable path cards shown side by side on desktop.
- Card A: `生成博客`
  - marked with `推荐`
  - short copy: `从 GitHub 仓库或本地文档开始，生成结构化技术博客`
- Card B: `知识复习`
  - short copy: `从知识库中抽取重点内容，进行复述与追问练习`
- `生成博客` is selected by default.
- Selected state uses stronger contrast, border, and shadow.
- Unselected state remains visible but visually quieter.
- Add a small recommendation hint near the selected card:
  - `推荐先从博客生成开始，再进入知识复习做内化`

## 11. Flow Preview Strip
- Show a horizontal 4-step strip directly under the decision center.
- The strip changes based on the selected path.

### 11.1 Blog Flow
1. `选择来源`
2. `完成解析`
3. `选择写作场景`
4. `开始生成`

### 11.2 Review Flow
1. `选择入口`
2. `选择模式`
3. `开始复述`
4. `查看反馈`

### 11.3 Interaction Rule
- The strip is instructional, not a full wizard.
- Current step is highlighted.
- Future steps stay visible but lighter.
- The purpose is to reduce uncertainty and teach the process visually.

## 12. Active Workspace
Only one workflow is expanded at a time.

### 12.1 Blog Mode Workspace
Left side blocks:
- `选择内容来源`
  - segmented choices:
    - `GitHub 仓库`
    - `本地文档`
- if GitHub is selected:
  - repository URL input
  - scan button
- if local document is selected:
  - drag-and-drop style area
  - supported file format hint
- `选择写作目标`
  - scenario cards:
    - `电子书解读`
    - `开箱复习`
    - `小白教程`

Right side summary panel:
- `本次任务摘要`
- `内容来源`
- `创作目标`
- `预计输出`
- primary CTA:
  - `开始解析并生成`

### 12.2 Review Mode Workspace
Left side blocks:
- `选择复习入口`
  - `今日推荐`
  - `随机抽一篇`
  - `手动选择文章`
- `选择复习方式`
  - `轻提示复述`
  - `细致提问`
- short preview of selected entry/article

Right side summary panel:
- `本次复习摘要`
- `开始方式`
- `复习模式`
- `预计耗时`
- primary CTA:
  - `开始本次复习`

## 13. Support Area
- Place support modules below the main workspace.
- Keep them compact and secondary.

### 13.1 Recent Blog Work
- up to 3 items
- each item shows:
  - title
  - status
  - resume action

### 13.2 Recent Review Records
- up to 3 items
- each item shows:
  - topic
  - mode
  - suggested next action

### 13.3 Resume Card
- If there is unfinished work, show `继续上次任务` above or inside the support area.
- This should be visible, but it should not overpower the current chosen flow.

## 14. Page States
### 14.1 First Visit
- Default to `生成博客`.
- Show a short note recommending the first path for new users.

### 14.2 Blog Selected
- Blog workspace expands.
- Review path stays available as a quiet secondary option.

### 14.3 Review Selected
- Review workspace expands.
- Blog path remains visible as the switch-back option.

### 14.4 Ready To Start
- Summary panel clearly reflects all current selections.
- Primary CTA becomes the obvious next action.

### 14.5 Resume Available
- Show a compact resume affordance if unfinished work exists.

## 15. Interaction Principles
- Every click should reduce uncertainty.
- The page should answer three questions after each selection:
  - `What did I choose?`
  - `What happens next?`
  - `What is the main action now?`
- The mockup should never show both workflows fully expanded at once.

## 16. Content Rules
- All UI text in the HTML must be Chinese.
- The mockup should look close to a real homepage, not a wireframe board.
- Use realistic content labels and examples, but keep the structure simple and readable.
- Avoid overloading the page with extra options unrelated to the chosen workflow.

## 17. HTML Mockup Requirements
- Must be a standalone HTML file.
- Must support clickable switching between:
  - `生成博客`
  - `知识复习`
- Must support clickable changes for key choices such as:
  - source type
  - scenario mode
  - review entry
  - review mode
- Must update the visible flow preview and summary panel based on selection.
- Must include compact mock history/support panels below the main workspace.

## 18. Success Criteria
- A reviewer can understand the homepage structure within a few seconds.
- The page clearly communicates that `生成博客` is recommended.
- The page feels more guided than the current flat multi-card layout.
- The mockup helps compare the future product direction without requiring backend functionality.

## 19. Risks
- If both paths receive too much visual weight, the page may still feel flat.
- If the support/history area is too tall, it may steal attention from the main workflow.
- If the flow strip looks too much like a dense navigation component, it may add complexity instead of clarity.

## 20. Mitigations
- Keep one active workflow only.
- Use stronger emphasis on the selected lane and current CTA.
- Keep support modules compact and visually quieter.
- Use the flow strip as a teaching layer, not as a second dashboard.

## 21. Implementation Readiness
- This spec is ready for a standalone HTML mockup.
- The first deliverable should focus on layout, path switching, content hierarchy, and lightweight client-side interaction.
- Production component refactoring or frontend integration should happen only after the mockup direction is validated.
