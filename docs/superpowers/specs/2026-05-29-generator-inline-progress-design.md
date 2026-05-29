# InkWords Generator Inline-Progress Design

## 1. Background
- The current generator redesign introduced a dedicated `progress` stage.
- After practical testing, that model turned out to be conceptually wrong for this workflow.
- The user expectation is clearer than the current implementation:
  - `配置解析` should remain the active step while parsing/analyzing is running
  - `确认大纲` should remain the active step while writing/generating is running
- A standalone `处理进度` step makes the process feel like it jumps forward and then backward:
  - file upload may appear to jump to progress before the user finishes scene selection
  - generation may appear to leave the outline step and then return later
- The better model is to treat progress as an in-step working state, not as a separate stage.

## 2. Goal
- Replace the 4-step generator model with a clearer 3-step model.
- Keep progress visible, but embed it inside the active step rather than moving to a dedicated progress page.
- Preserve the source-first structure and the strict focus of each step.
- Make the flow match the user mental model:
  1. 选择来源
  2. 配置解析
  3. 确认大纲

## 3. Scope
### 3.1 In Scope
- Update generator step design from 4 steps to 3 steps.
- Remove `progress` as an independent page/stage concept.
- Embed parse/analyze progress inside `配置解析`.
- Embed writing/generation progress inside `确认大纲`.
- Adjust stage derivation, page composition, and progress component ownership to match this model.

### 3.2 Out of Scope
- No backend API redesign.
- No change to the SSE protocol format.
- No editor/result-page redesign.
- No unrelated style overhaul of the generator page.

## 4. Product Decision
- Chosen flow model: `3-step flow`
- Chosen progress model: `Inline progress in step 2 and step 3`
- Rejected model: `Standalone progress step`

## 5. Why This Direction
- It matches the real task structure:
  - Step 2 is not just configuration; it is configuration plus analysis.
  - Step 3 is not just outline confirmation; it is outline review plus writing progress.
- It removes the misleading sense that progress is a separate workflow milestone.
- It prevents the UI from feeling like it skips ahead and then returns.
- It keeps the user anchored in the step where the work belongs.
- It reuses the existing progress rendering logic without forcing the product to invent a fake fourth step.

## 6. Core UX Model
- Old model:
  1. 选择来源
  2. 配置解析
  3. 确认大纲
  4. 处理进度
- New model:
  1. 选择来源
  2. 配置解析
     - choose scene
     - start analyze
     - watch analyze progress inline
  3. 确认大纲
     - edit outline
     - start generate
     - watch writing progress inline

## 7. Step Responsibilities
### 7.1 Step 1: 选择来源
- Only source selection is shown.
- No scenario selector here.
- No progress panel here.
- The user chooses:
  - GitHub repository
  - local file / ZIP / courseware

### 7.2 Step 2: 配置解析
- This step owns:
  - scene selection
  - git module selection when source is `git`
  - parsed file summary when source is `file`
  - the action that starts analysis / outline generation
  - parse/analyze progress feedback
- While analysis is running, the user is still in `配置解析`.
- The step label in the strip must remain `配置解析`, not change to a separate progress step.

### 7.3 Step 3: 确认大纲
- This step owns:
  - outline review and editing
  - locked scene context
  - generation start action
  - chapter-by-chapter writing progress
- While writing is running, the user is still in `确认大纲`.
- The progress panel becomes part of the same step layout rather than replacing the entire page.

## 8. Progress Placement
### 8.1 Parsing / Analysis Progress
- Parsing and analysis progress must render inline within `配置解析`.
- The progress panel should appear below or alongside the step’s primary controls.
- The user should still understand that they are configuring and analyzing within the same step.

### 8.2 Writing / Generation Progress
- Generation progress must render inline within `确认大纲`.
- The progress panel should appear inside the outline step layout.
- The user should still feel they are inside the outline/writing step, not transported to a separate screen.

## 9. Step Strip Rules
- The top strip should have exactly 3 items:
  1. `选择来源`
  2. `配置解析`
  3. `确认大纲`
- No `处理进度` step in the strip.
- While parsing/analyzing:
  - current strip step stays on `配置解析`
- While generating:
  - current strip step stays on `确认大纲`

## 10. State Model
- The generator should derive only 3 stages:
  - `source`
  - `configure`
  - `outline`
- Progress should be represented as sub-state inside `configure` and `outline`, not as a top-level stage.

### 10.1 Configure-State Semantics
- `configure` can be idle or working.
- For `git`:
  - idle: modules selectable, scene selectable
  - working: analysis progress shown inline
- For `file`:
  - idle: file parsed, scene selectable, waiting for `生成大纲`
  - working: outline analysis progress shown inline

### 10.2 Outline-State Semantics
- `outline` can be idle or generating.
- idle:
  - editable outline
  - start generation action
- generating:
  - writing progress visible inline
  - editing controls can be restricted as needed

## 11. Component Strategy
### 11.1 Keep
- `GeneratorSourceStage`
- `GeneratorConfigureStage`
- `GeneratorOutlineStage`
- existing store, stream hooks, and async services

### 11.2 Change
- `GeneratorProgressStage` should no longer represent a separate product step.
- `GeneratorStatus` should become a reusable progress panel that can be embedded inside:
  - `GeneratorConfigureStage`
  - `GeneratorOutlineStage`
- `generatorViewState.ts` should stop deriving `progress` as a top-level stage.

### 11.3 Optional Refactor
- `GeneratorProgressStage` may be removed entirely if it only exists to support the now-rejected standalone progress step.
- If keeping it temporarily reduces migration cost, it should be repurposed or retired in a follow-up cleanup.

## 12. File-Flow Requirement
- File upload must not jump directly from `选择来源` to an independent progress step.
- Correct file flow:
  1. upload file
  2. finish parse
  3. enter `配置解析`
  4. choose scene
  5. click `生成大纲`
  6. show analysis progress inline in `配置解析`
  7. enter `确认大纲`

## 13. Git-Flow Requirement
- Correct git flow:
  1. enter repository URL
  2. scan repository
  3. enter `配置解析`
  4. choose modules and scene
  5. click analyze
  6. show analysis progress inline in `配置解析`
  7. enter `确认大纲`

## 14. Generation Requirement
- Correct writing flow:
  1. edit outline in `确认大纲`
  2. click generate
  3. stay in `确认大纲`
  4. show generation progress inline
  5. finish generation without ever switching to a separate progress step

## 15. Back Navigation
- Back behavior remains step-based:
  - from `配置解析` back to `选择来源`
  - from `确认大纲` back to `配置解析`
- Progress visibility does not create a separate navigation state.
- The user should not feel they are navigating between “outline page” and “progress page.”

## 16. Testing Strategy
### 16.1 View-State Tests
- Verify only 3 top-level stages exist.
- Verify parsing/analyzing keeps the current step at `configure`.
- Verify generating keeps the current step at `outline`.

### 16.2 Flow Tests
- Verify file upload lands on `configure` after successful parse.
- Verify file outline generation starts inside `configure` and does not create a separate top-level step.
- Verify git analysis stays inside `configure`.
- Verify writing progress stays inside `outline`.

### 16.3 UI Verification
- Verify the step strip never shows a fourth progress step.
- Verify progress panels are embedded in the active stage layout.

## 17. Success Criteria
- The generator shows 3 steps, not 4.
- Step 2 contains both configuration and analysis progress.
- Step 3 contains both outline editing and writing progress.
- File uploads no longer appear to jump to progress before scene selection.
- Writing no longer appears to leave outline confirmation for a separate page.

## 18. Risks
- If progress visibility is not clearly integrated into the active step, the page could feel crowded again.
- If top-level stage and working-state logic are not separated carefully, regressions may reintroduce skipped steps.
- Temporary coexistence of old and new progress abstractions could cause confusion in the code.

## 19. Mitigations
- Keep top-level stage count fixed at 3.
- Model working progress as embedded sub-state, not separate stage.
- Reuse one progress panel component and render it conditionally within the active step.
- Cover both file and git paths with regression tests.

## 20. References
- Earlier 4-step design: `docs/superpowers/specs/2026-05-29-generator-stage-replacement-design.md`
- Current revised direction: this document supersedes the standalone progress-step decision.

## 21. Implementation Readiness
- This design is ready for implementation planning once reviewed.
- The implementation should focus on:
  - collapsing the stage model from 4 to 3
  - embedding progress inside configure and outline
  - updating tests to reflect the new mental model
