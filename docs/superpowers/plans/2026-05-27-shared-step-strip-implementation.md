# Shared Step Strip Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract a reusable `StepStrip` component and shared `StepStripItem` type for `HomeEntry`, `Generator`, and `KnowledgeReview`.

**Architecture:** Create one shared presentational component with `preview` and `progress` variants, keep all workflow state in the existing pages and view-state helpers, and replace the duplicated strip JSX in three pages with the shared API. Use lightweight Vitest coverage for the shared type shape and rendered markup without introducing a new testing stack.

**Tech Stack:** React 19, TypeScript, Vitest, react-dom/server, Tailwind CSS

---

### Task 1: Lock Shared Step Data Shape

**Files:**
- Modify: `frontend/src/pages/homeEntryViewState.ts`
- Modify: `frontend/src/pages/homeEntryViewState.test.ts`

- [ ] **Step 1: Write the failing test for `StepStripItem[]` home entry steps**
- [ ] **Step 2: Run the test to verify it fails for the expected reason**
- [ ] **Step 3: Update the helper to return shared step items**
- [ ] **Step 4: Re-run the test to verify it passes**

### Task 2: Add The Shared StepStrip Component

**Files:**
- Create: `frontend/src/components/shared/StepStrip.tsx`
- Create: `frontend/src/components/shared/StepStrip.test.tsx`

- [ ] **Step 1: Write the failing component test for `preview` and `progress` rendering**
- [ ] **Step 2: Run the test to verify it fails because the component does not exist**
- [ ] **Step 3: Implement the minimal `StepStrip` component and exported `StepStripItem` type**
- [ ] **Step 4: Re-run the component test to verify it passes**

### Task 3: Adopt StepStrip In The Three Pages

**Files:**
- Modify: `frontend/src/pages/HomeEntry.tsx`
- Modify: `frontend/src/pages/Generator.tsx`
- Modify: `frontend/src/pages/KnowledgeReview.tsx`

- [ ] **Step 1: Replace the home entry preview strip**
- [ ] **Step 2: Replace the generator progress strip**
- [ ] **Step 3: Replace the knowledge review progress strip**
- [ ] **Step 4: Keep summaries, CTA logic, and current-step ownership in the pages**

### Task 4: Validate The Refactor

**Files:**
- Test: `frontend/src/pages/homeEntryViewState.test.ts`
- Test: `frontend/src/components/shared/StepStrip.test.tsx`

- [ ] **Step 1: Run the focused Vitest suite**
- [ ] **Step 2: Run the frontend production build**
- [ ] **Step 3: Confirm the three pages still compile and share the same strip component**
