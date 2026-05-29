# StepStrip Visual Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Polish the shared `StepStrip` so the `preview` and `progress` variants feel more elegant and more clearly differentiated without changing API or behavior.

**Architecture:** Keep the refactor contained to `StepStrip.tsx` and preserve the current public props and page integrations. Strengthen the visual hierarchy through class refinements for header spacing, card surfaces, and step-state treatment, while keeping the existing test contract lightweight and behavior-focused.

**Tech Stack:** React 19, TypeScript, Vitest, Tailwind CSS

---

### Task 1: Lock The New Visual Semantics

**Files:**
- Modify: `frontend/src/components/shared/StepStrip.test.tsx`

- [ ] **Step 1: Write the failing test for refined preview and progress state markers**
- [ ] **Step 2: Run the test to verify it fails for the expected reason**

### Task 2: Polish StepStrip

**Files:**
- Modify: `frontend/src/components/shared/StepStrip.tsx`

- [ ] **Step 1: Refine wrapper spacing and header rhythm**
- [ ] **Step 2: Refine preview card styling to feel lighter and calmer**
- [ ] **Step 3: Refine progress current/complete/upcoming hierarchy**
- [ ] **Step 4: Keep motion minimal and preserve dark-mode balance**

### Task 3: Validate The Visual Refactor

**Files:**
- Test: `frontend/src/components/shared/StepStrip.test.tsx`

- [ ] **Step 1: Re-run the focused StepStrip and workflow tests**
- [ ] **Step 2: Run the frontend production build**
- [ ] **Step 3: Confirm the refactor stays API-compatible with the three existing page usages**
