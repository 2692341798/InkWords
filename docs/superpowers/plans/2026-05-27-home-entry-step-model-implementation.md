# Home Entry Step-Model Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a guided home entry page so authenticated users without a selected blog choose a path before entering the real generator or review pages.

**Architecture:** Introduce one new `home-entry` view in the frontend store, implement a lightweight `HomeEntry` page with local path selection and a pure view-state helper, and wire `App` plus `Sidebar` so the new page becomes the default work landing surface. Keep generator, review, and dashboard business logic unchanged.

**Tech Stack:** React 18, TypeScript, Zustand, Vitest, Tailwind CSS

---

### Task 1: Define Home Entry View State

**Files:**
- Create: `frontend/src/pages/homeEntryViewState.ts`
- Create: `frontend/src/pages/homeEntryViewState.test.ts`

- [ ] **Step 1: Write the failing tests for home entry path data**
- [ ] **Step 2: Run the view-state tests to verify they fail**
- [ ] **Step 3: Implement the minimal pure helper**
- [ ] **Step 4: Re-run the tests and make them pass**

### Task 2: Add The New Top-Level App View

**Files:**
- Modify: `frontend/src/store/blogStore.ts`
- Modify: `frontend/src/App.tsx`

- [ ] **Step 1: Extend the store view union with `home-entry`**
- [ ] **Step 2: Change the default non-editor landing to `HomeEntry`**
- [ ] **Step 3: Keep dashboard, generator, and review routes intact**
- [ ] **Step 4: Run a focused build or tests to verify the new view wiring**

### Task 3: Build The Home Entry Page

**Files:**
- Create: `frontend/src/pages/HomeEntry.tsx`
- Reference: `frontend/src/pages/Generator.tsx`
- Reference: `frontend/src/pages/KnowledgeReview.tsx`

- [ ] **Step 1: Render the hero, decision cards, and flow preview**
- [ ] **Step 2: Render the selected path summary and CTA**
- [ ] **Step 3: Render compact support sections for resume, recent blogs, and recent review activity**
- [ ] **Step 4: Keep the page shallow and avoid embedding generator/review logic**

### Task 4: Wire Sidebar Entry Actions

**Files:**
- Modify: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: Point `新工作区` to `home-entry`**
- [ ] **Step 2: Keep direct access to `知识漫游复习` and `个人中心`**
- [ ] **Step 3: Preserve current interrupt/reset behavior when a stream is active**

### Task 5: Validate The New Entry Flow

**Files:**
- Test: `frontend/src/pages/homeEntryViewState.test.ts`
- Modify if needed: `frontend/src/App.tsx`
- Modify if needed: `frontend/src/components/Sidebar.tsx`

- [ ] **Step 1: Run focused Vitest coverage for the new helper**
- [ ] **Step 2: Run the frontend production build**
- [ ] **Step 3: Check that no-selected-blog users land on the new entry page and CTA transitions still reach the real target pages**
