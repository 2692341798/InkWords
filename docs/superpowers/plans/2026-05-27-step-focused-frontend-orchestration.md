# Step-Focused Frontend Orchestration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the real frontend so `Generator` and `KnowledgeReview` only show the user's current step at any moment.

**Architecture:** Keep existing data-fetching and business logic intact, and move the orchestration into page-level view-state helpers. Each page renders a compact step indicator plus one active panel, while child components remain mostly unchanged and only receive the extra callbacks needed for back/close navigation.

**Tech Stack:** React 18, TypeScript, Zustand, Vitest, Tailwind CSS

---

### Task 1: Define Step Gating Rules

**Files:**
- Modify: `frontend/src/pages/generatorViewState.ts`
- Create: `frontend/src/pages/knowledgeReviewViewState.ts`
- Test: `frontend/src/pages/generatorViewState.test.ts`
- Test: `frontend/src/pages/knowledgeReviewViewState.test.ts`

- [ ] Add explicit generator step output such as `input`, `configure`, `outline`, and `processing`.
- [ ] Add explicit knowledge review step output such as `entry`, `picker`, and `session`.
- [ ] Cover the expected transitions with unit tests before page refactors.

### Task 2: Refactor Generator Page To One Active Step

**Files:**
- Modify: `frontend/src/pages/Generator.tsx`
- Reference: `frontend/src/components/generator/GeneratorInput.tsx`
- Reference: `frontend/src/components/generator/GeneratorModules.tsx`
- Reference: `frontend/src/components/generator/GeneratorOutline.tsx`
- Reference: `frontend/src/components/generator/GeneratorStatus.tsx`

- [ ] Add a small step strip and current-step summary at the page level.
- [ ] Render only one primary generator section at a time:
  - input step
  - configure step
  - outline step
  - processing overlay
- [ ] Keep `GeneratorStatus` mounted for scanning/analyzing/generating feedback.
- [ ] Add minimal back navigation so users can return from outline to configure, and from configure to source selection.

### Task 3: Refactor Knowledge Review To One Active Step

**Files:**
- Modify: `frontend/src/pages/KnowledgeReview.tsx`
- Modify: `frontend/src/components/review/ReviewNotePicker.tsx`
- Modify: `frontend/src/components/review/ReviewSessionCard.tsx`
- Reference: `frontend/src/hooks/useKnowledgeReview.ts`
- Reference: `frontend/src/store/reviewStore.ts`

- [ ] Add page-local picker/session orchestration based on the new review view state helper.
- [ ] Show only one active review block at a time:
  - entry selection
  - manual picker
  - active session
- [ ] Hide history while picker or session is active.
- [ ] Add minimal back/close actions so users can return to the entry step after opening the picker or finishing a session.

### Task 4: Verify The Changed Frontend

**Files:**
- Test: `frontend/src/pages/generatorViewState.test.ts`
- Test: `frontend/src/pages/knowledgeReviewViewState.test.ts`
- Modify if needed: `frontend/src/pages/KnowledgeReview.tsx`

- [ ] Run focused Vitest coverage for the updated view-state helpers.
- [ ] Run focused tests for the knowledge review page if assertions need updating.
- [ ] Smoke-check the real UI flow after code changes and keep only the current step visible at page level.
