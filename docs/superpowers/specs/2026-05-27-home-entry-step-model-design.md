# InkWords Home Entry Step-Model Design

## 1. Background
- The current real app entry does not behave like a guided home page.
- In the current structure:
  - `App.tsx` switches between `Dashboard`, `KnowledgeReview`, `Generator`, and `Editor`.
  - When there is no selected blog, the user is effectively dropped into a work page instead of first being guided through a top-level decision.
- After the recent step-focused refactor, `Generator` and `KnowledgeReview` now each show one current step at a time.
- The remaining gap is the top-level entry itself: users still need a clearer first question before entering a tool page.

## 2. Goal
- Add the same process-oriented step model to the real home entry page.
- Make the first interaction answer: `你现在想先做什么？`
- Stop defaulting the no-selection state to a tool page.
- Create a guided entry surface that:
  - lets users choose between `生成博客` and `知识复习`
  - previews the selected flow
  - provides one clear CTA into the real target page
  - keeps recent activity and resume information compact and secondary

## 3. Scope
### 3.1 In Scope
- Add one dedicated real home entry page in the frontend.
- Use a `Hub First` entry model when there is no selected blog.
- Use a `Hybrid` transition model:
  - choose a path on the home entry page
  - show inline flow preview and summary
  - click CTA to enter the real `Generator` or `KnowledgeReview` page
- Include compact support sections:
  - `继续上次任务`
  - `最近博客记录`
  - `最近复习记录`
- Reuse existing generator and review pages instead of duplicating their full logic on the entry page.

### 3.2 Out of Scope
- No backend API redesign.
- No merger of `Dashboard` profile/statistics into the home entry page.
- No full inline embedding of generator or review workflows into the home entry page.
- No heavy resume engine in the first pass.
- No routing framework migration.

## 4. Product Decision
- Chosen target: `Real Home Entry`.
- Chosen entry behavior: `Hub First`.
- Chosen transition model: `Hybrid`.
- Chosen secondary content strategy: `Resume + Recent History`.

## 5. Why This Direction
- A guided home entry page fixes the top-level decision problem without duplicating tool pages.
- `Generator` remains the recommended default path because it is still the main production flow.
- `KnowledgeReview` remains visible as the secondary path for knowledge internalization.
- This approach fits the existing app architecture and minimizes change risk:
  - top-level decision stays in one new page
  - existing page-level business logic stays in `Generator` and `KnowledgeReview`

## 6. Core UX Model
- Old top-level behavior: user lands inside a tool page.
- New top-level behavior:
  1. choose a path
  2. understand the path
  3. enter the selected tool

This top-level rule matches the new internal rule already applied to `Generator` and `KnowledgeReview`, where users only see the current step.

## 7. Entry Behavior
- When the user is authenticated and there is no selected blog:
  - show the new home entry page first
- The home entry page becomes the operational landing surface for starting work.
- The existing `Dashboard` remains a separate `个人中心 / 数据统计` page.

## 8. Page Responsibilities
The new home entry page has only three responsibilities:
1. Help the user choose a work path
2. Explain the selected path with a short step model
3. Hand the user off to the real page with a single CTA

It must not execute generation or review logic directly.

## 9. Information Architecture
The page contains five main blocks:
1. `Hero`
2. `Decision Cards`
3. `Flow Preview Strip`
4. `Current Summary + Primary CTA`
5. `Support Area`

## 10. Hero
- Show product name and short value statement.
- Explain that the page is for starting work, not for account management.
- Suggested helper question:
  - `今天你想先完成哪一种任务？`

## 11. Decision Cards
- Two main cards:
  - `生成博客（推荐）`
  - `知识复习`
- `生成博客` is selected by default.
- Only one selected path can be active at a time.
- The selected card should have stronger emphasis.
- The unselected card remains visible but quieter.

## 12. Flow Preview Strip
- Show the selected path as a compact step strip.
- The strip is informational, not a full wizard.

### 12.1 Blog Flow
1. `选择来源`
2. `完成解析`
3. `确认大纲`
4. `开始生成`

### 12.2 Review Flow
1. `选择入口`
2. `开始会话`
3. `获得反馈`

### 12.3 Strip Behavior
- Updates immediately when the user switches paths.
- Keeps the home entry page instructional and process-oriented.

## 13. Current Summary And CTA
- The center action area should explain:
  - what this path is for
  - what the user will do next
  - where the CTA will take them

### 13.1 Blog Summary
- Explain that the user will enter the real blog generation page.
- Emphasize that the next step is to choose source and scenario, then continue into the existing gated generator flow.
- CTA:
  - `进入博客生成`

### 13.2 Review Summary
- Explain that the user will enter the real review page.
- Emphasize that the next step is to choose entry mode and continue into the existing gated review flow.
- CTA:
  - `进入知识复习`

## 14. Support Area
The lower section should remain compact and secondary.

### 14.1 Continue Last Task
- Show a small resume card when the app can infer recent work.
- First pass can stay light:
  - use available local/frontend state if present
  - if strong resume data is unavailable, show a generic continue suggestion

### 14.2 Recent Blog Work
- Show compact recent blog items from existing blog data.
- Each item should be short and scannable.
- No full history tree here.

### 14.3 Recent Review Activity
- Show compact recent review summaries.
- No full review workspace here.

### 14.4 Explicit Constraint
- Do not include profile editing, charts, or full dashboard analytics on this page.

## 15. App Integration
### 15.1 `App.tsx`
- Add a new home entry page component.
- When authenticated and no blog is selected:
  - the new home entry page becomes the default landing surface
- Existing editor behavior remains unchanged when a blog is selected.

### 15.2 `Sidebar.tsx`
- `新工作区` should open the new home entry page instead of jumping directly into the generator page.
- `知识漫游复习` can still directly open the review page from the sidebar.
- `个人中心` remains separate and still points to the current dashboard page.

## 16. State Design
### 16.1 Local Page State
- Use a small page-local state:
  - `activePath: 'blog' | 'review'`
- Default:
  - `blog`

### 16.2 Derived View State
Use a pure helper to provide:
- title
- description
- steps
- CTA label
- target view
- recommendation text

Why: the page should stay declarative and easy to test.

## 17. File Plan
### 17.1 Create
- `frontend/src/pages/HomeEntry.tsx`
- `frontend/src/pages/homeEntryViewState.ts`
- `frontend/src/pages/homeEntryViewState.test.ts`

### 17.2 Modify
- `frontend/src/App.tsx`
- `frontend/src/components/Sidebar.tsx`

## 18. Visual Rules
- Only one primary action area at a time.
- The page must feel like an operational start page, not a dashboard grid.
- Keep the visual rhythm:
  - choose
  - preview
  - enter
- Support cards must remain smaller and quieter than the main action area.

## 19. Testing Strategy
### 19.1 View State Tests
- Verify `blog` and `review` produce the correct:
  - title
  - steps
  - CTA label
  - target view

### 19.2 App Integration Tests
- Verify the no-selected-blog landing uses the new home entry page.
- Verify path CTA transitions to the correct existing page.

### 19.3 Sidebar Interaction
- Verify `新工作区` routes to the new home entry page.

## 20. Success Criteria
- Users no longer land directly in the generator page by default.
- The first screen clearly asks what they want to do next.
- Only one selected top-level path is emphasized at a time.
- The home entry page explains the next action before the user enters the tool page.
- The page feels like a guided start surface instead of a mixed dashboard.

## 21. Risks
- If the support area becomes too large, the page may lose its process-oriented hierarchy.
- If the home entry page starts duplicating too much tool logic, it will become expensive to maintain.
- If path switching and actual target pages drift apart, the summary could become misleading.

## 22. Mitigations
- Keep the home entry page intentionally shallow.
- Reuse the real `Generator` and `KnowledgeReview` pages for actual work.
- Put path definitions into a small pure helper so labels, steps, and CTA targets stay consistent.

## 23. Implementation Readiness
- This design is ready for a dedicated implementation plan.
- The implementation should be done as a focused frontend change:
  - add the new page
  - wire it into `App`
  - update `Sidebar`
  - keep `Dashboard` untouched for now
