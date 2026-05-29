# InkWords Generator Stage-Replacement Design

## 1. Background
- The current generator page already tries to reduce clutter by showing one main step at a time.
- However, the user still reads the page as a mixed workspace instead of a true guided flow.
- The main remaining UX problem is not only visibility, but stage ownership:
  - source selection still shares attention with scenario setup
  - progress is shown through a modal overlay instead of a dedicated stage surface
  - the page still feels like one long tool page rather than a sequence of focused pages
- The requested target experience is stricter:
  1. select source first
  2. replace the page with the next required stage
  3. replace the page again with a dedicated processing surface

## 2. Goal
- Turn `Generator` into a strict stage-replacement flow.
- Make each stage own the full main workspace.
- Ensure the user always sees only the current task.
- Move processing feedback from modal overlay behavior to a dedicated progress page.
- Keep the existing generator business logic, stream orchestration, and store structure intact where possible.

## 3. Scope
### 3.1 In Scope
- Frontend-only generator flow redesign.
- Restructure the generator page into full-stage page-like views.
- Separate `source`, `configure`, `outline`, and `progress` into distinct stage surfaces.
- Move progress rendering away from the modal overlay experience into a dedicated processing page.
- Tighten back-navigation rules so that processing is non-reversible from the UI.

### 3.2 Out of Scope
- No backend API change.
- No SSE protocol redesign.
- No new generation capability.
- No new editor/result-page redesign in this pass.
- No app-wide route migration to real per-step URLs.

## 4. Product Decisions
- Chosen stage transition model: `Full page replace`
- Chosen processing surface: `Full dedicated progress page`
- Chosen back behavior: `No back during processing`
- Recommended implementation strategy: `Route-like stage replacement inside the same page`

## 5. Why This Direction
- It gives the user the “page refreshes into the next stage” feeling without forcing a routing rewrite.
- It fits the current container plus Zustand plus stream-hook architecture.
- It minimizes implementation risk because async orchestration can stay in place while the page composition changes.
- It aligns with the existing project direction toward step-focused workflows instead of multi-panel mixed workspaces.

## 6. Current Constraints
- The frontend must keep all user-facing text in Chinese.
- Existing business orchestration should continue to flow through:
  - `useBlogStream`
  - `useStreamStore`
  - `getGeneratorViewState` or a stricter replacement helper
- The generator flow must continue to support both:
  - `git`
  - `file`
- The change should remain a focused frontend refactor, not a platform rewrite.

## 7. Core UX Model
- Old mental model:
  - one generator page
  - multiple related blocks
  - progress appears as an overlay
- New mental model:
  1. choose source
  2. configure only what is needed next
  3. confirm outline only
  4. enter a dedicated progress page

The user should never feel that earlier and later actions are competing for attention.

## 8. Stage Model
The generator flow becomes four explicit stages:
1. `source`
2. `configure`
3. `outline`
4. `progress`

Each stage replaces the previous stage in the main workspace.

## 9. Stage Responsibilities
### 9.1 Source Stage
- Show only source selection.
- Source selection remains the first question the page asks.
- The stage should not show scenario selection anymore.
- Supported source entry points:
  - GitHub repository URL
  - local file upload / drag-and-drop
- Once a valid source action succeeds, the main workspace should advance to the next required stage.

### 9.2 Configure Stage
- This stage is responsible for setup that depends on the chosen source.
- For `git`:
  - show scanned modules
  - allow module selection
  - show scenario selection
  - allow the user to start deep analysis
- For `file`:
  - show parsed source summary
  - show scenario selection
  - allow the user to continue toward outline generation or the equivalent next step in the current file flow
- This stage should feel like a dedicated setup page, not a leftover part of the source page.

### 9.3 Outline Stage
- Show only outline review and outline editing.
- Show the locked scenario label as context, not as an editable upstream control.
- Hide source selection and configuration controls entirely.
- Provide the primary generation action here.
- Back is still allowed from this stage.

### 9.4 Progress Stage
- Replace the full workspace with a dedicated progress page.
- This stage is the only visible surface while:
  - scanning
  - analyzing
  - generating
- The user should not be able to navigate backward from this stage.
- The progress page should communicate:
  - current operation
  - current progress text
  - step or chapter status
  - streaming feedback

## 10. Transition Rules
### 10.1 Forward Transitions
- `source -> configure`
  - after a source is successfully selected and the next setup step becomes available
- `configure -> progress`
  - immediately after analysis/parsing work starts
- `progress -> outline`
  - when outline generation finishes and no processing is active
- `outline -> progress`
  - immediately after blog generation starts

### 10.2 Return Transitions
- `configure -> source`
  - allowed through an explicit back action
- `outline -> configure`
  - allowed through an explicit back action
- `progress -> previous stage`
  - not allowed from the UI once processing starts

### 10.3 Step Strip Behavior
- The step strip remains a read-oriented orientation layer.
- During processing:
  - no clickable back navigation through completed steps
- Outside processing:
  - clickable steps are optional, but not required for the first pass
- The first-pass requirement is satisfied by explicit back controls on eligible stages.

## 11. Page Composition Rules
- The generator page should render one page-level stage component at a time.
- Shared chrome may remain:
  - page title
  - progress strip
  - lightweight context header
- Main stage content must not be mixed across stages.
- The old right summary sidebar should not remain visible on the dedicated progress page.

## 12. Progress Page Design
- The current `GeneratorStatus` overlay should be converted into dedicated page content.
- The progress page should inherit the useful content already present in the existing status component:
  - parsing status message
  - analysis step bar
  - analysis history list
  - chapter-by-chapter generation status
  - live content snippets where appropriate
- The page should no longer behave like a blocking modal.
- The stage itself becomes the blocking focus.

## 13. Scenario Selection Rule
- Scenario selection moves out of the source stage.
- Scenario selection belongs to the configure stage.
- Once the outline exists, the scenario is treated as locked context and shown as a read-only label in the outline stage.

Why:
- Choosing the source is the user’s first decision.
- Choosing the writing scenario is the user’s second decision.
- Keeping them separate produces a clearer mental model and a stronger sense of progression.

## 14. Back Navigation Rule
- Provide an explicit back button in eligible stages:
  - configure
  - outline
- Do not allow back interaction while processing is active.
- If cancel actions exist for analysis or generation, they stop the current process but do not restore unrestricted backward navigation while the system is still in a processing state.

## 15. State Design
### 15.1 Stage Derivation
- Replace the current loose visibility flags with a stricter stage machine or equivalent derived helper.
- The helper should determine:
  - current stage key
  - current step index
  - whether back is allowed
  - whether the scenario selector is editable
  - which page-level component should render

### 15.2 Input Signals
- The stage helper can continue to derive from existing store signals such as:
  - `sourceType`
  - `modules`
  - `outline`
  - `scenarioMode`
  - `isScanning`
  - `isAnalyzing`
  - `isGenerating`

### 15.3 Design Principle
- The UI must remain state-driven.
- The stage shown to the user should come from data state, not from ad hoc local toggle chains.

## 16. Component Strategy
### 16.1 Keep
- `useBlogStream`
- `useStreamStore`
- existing source handlers
- existing async generation handlers

### 16.2 Refactor
- `frontend/src/pages/Generator.tsx`
- `frontend/src/pages/generatorViewState.ts`
- `frontend/src/components/generator/GeneratorStatus.tsx`

### 16.3 Add
- `GeneratorSourceStage`
- `GeneratorConfigureStage`
- `GeneratorOutlineStage`
- `GeneratorProgressStage`

The exact filenames may vary, but the page should be composed from stage-level components rather than mixed inline sections.

## 17. File Plan
### 17.1 Modify
- `frontend/src/pages/Generator.tsx`
- `frontend/src/pages/generatorViewState.ts`
- `frontend/src/components/generator/GeneratorStatus.tsx`
- `frontend/src/components/shared/StepStrip.tsx` only if needed for non-clickable processing behavior

### 17.2 Create
- stage-level generator page components under `frontend/src/components/generator/` or `frontend/src/pages/`
- focused tests for the stricter stage derivation helper

## 18. Testing Strategy
### 18.1 View-State Tests
- Verify the derived stage for:
  - initial empty state
  - git source selected and modules available
  - file source parsed and ready for configuration
  - outline ready
  - scanning state
  - analyzing state
  - generating state

### 18.2 Page Rendering Tests
- Verify only one stage component renders at a time.
- Verify scenario selection is absent from the source stage.
- Verify the progress stage replaces the main workspace during processing.

### 18.3 Interaction Tests
- Verify back is available in configure and outline.
- Verify back is not available in progress.
- Verify generation start transitions into the dedicated progress page.

### 18.4 Verification During Implementation
- Run focused frontend tests for the stage helper and generator page.
- Run a frontend build.
- If visual behavior needs end-to-end confirmation, verify through the real app flow.

## 19. Success Criteria
- The generator no longer feels like a page with multiple simultaneous task zones.
- The first screen asks only for source selection.
- Scenario selection happens only after source selection.
- Outline editing appears as its own page-like stage.
- Processing uses a dedicated progress page instead of a modal overlay.
- Back navigation is available before processing and disabled during processing.

## 20. Risks
- If the stage helper is not strict enough, mixed-stage rendering could reappear.
- File-source behavior may expose edge cases because its current flow is shorter than the git flow.
- Converting the overlay into a page may reveal assumptions in existing progress layout logic.
- If back behavior is implemented inconsistently, users may lose confidence in the staged model.

## 21. Mitigations
- Centralize stage derivation in one pure helper.
- Keep stage responsibilities explicit and non-overlapping.
- Reuse the existing progress content model instead of redesigning it from scratch.
- Validate both `git` and `file` paths before closing implementation.

## 22. References
- Current page container: `frontend/src/pages/Generator.tsx`
- Current derived step logic: `frontend/src/pages/generatorViewState.ts`
- Current progress overlay: `frontend/src/components/generator/GeneratorStatus.tsx`
- Knowledge base:
  - `[[concepts/前端组件体系：Generator 系列]]`
  - `[[concepts/前端自定义 Hooks：useBlogStream 与 useDebounce]]`

## 23. Implementation Readiness
- This design is ready for implementation planning.
- The implementation should be a focused frontend refactor:
  - introduce stage-level page components
  - tighten stage derivation
  - convert progress from overlay behavior to dedicated stage content
  - preserve the current business and stream orchestration
