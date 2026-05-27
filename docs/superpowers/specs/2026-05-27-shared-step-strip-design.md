# InkWords Shared Step Strip Design

## 1. Background
- `HomeEntry`, `Generator`, and `KnowledgeReview` now all present users with a step-oriented workflow.
- Each page currently contains its own inline step-strip JSX and styling.
- The pattern is visually similar but not identical:
  - `HomeEntry` uses a lighter preview strip.
  - `Generator` and `KnowledgeReview` use a stronger progress strip with current-step emphasis.
- This duplication increases maintenance cost and makes it easier for the three pages to drift apart visually.

## 2. Goal
- Extract one reusable step-strip UI component shared by:
  - `HomeEntry`
  - `Generator`
  - `KnowledgeReview`
- Share one common step item type across those pages.
- Preserve page-specific workflow logic and state ownership.
- Support two visual modes:
  - `preview`
  - `progress`

## 3. Scope
### 3.1 In Scope
- Create one shared `StepStrip` component.
- Export one shared `StepStripItem` type.
- Replace inline strip JSX in the three pages with the shared component.
- Keep the existing page-level summary, CTA, and state logic unchanged.

### 3.2 Out of Scope
- No shared workflow state machine.
- No shared summary sidebar component.
- No redesign of the surrounding page layouts.
- No attempt to merge page-specific business logic into the shared component.
- No abstraction of unrelated cards or hero sections.

## 4. Chosen Direction
- Reuse level: `UI + Shared Types`
- Visual flexibility: `Two Variants`
- Chosen implementation style: `Single Component With Variant Prop`

## 5. Why This Direction
- A single shared component reduces duplicate JSX and styling.
- Keeping business logic in each page avoids over-coupling unrelated workflows.
- Two variants are enough for current needs:
  - `HomeEntry` stays lightweight
  - `Generator` and `KnowledgeReview` keep stronger current-step emphasis
- This delivers meaningful reuse without creating a mini workflow framework.

## 6. Component Responsibilities
The shared component is responsible for:
- rendering the optional strip header
- rendering the optional strip description
- rendering the step cards
- applying visual treatment for:
  - current step
  - past step
  - future step
- adjusting layout based on step count

The shared component must not:
- compute workflow state from stores
- own page navigation
- own CTA behavior
- own page summaries
- decide which steps exist for a workflow

## 7. Shared Type
The shared type is:

```ts
type StepStripItem = {
  key: string
  title: string
  description?: string
}
```

Why:
- `key` keeps rendering stable.
- `title` covers the visible step label.
- `description` supports both lightweight preview copy and stronger progress explanations without forcing all pages to use it.

## 8. Component API
The initial API should be:

```ts
type StepStripProps = {
  title?: string
  description?: string
  steps: StepStripItem[]
  currentStepIndex?: number
  variant: 'preview' | 'progress'
  className?: string
}
```

Rules:
- `currentStepIndex` is optional because `preview` may use lighter emphasis.
- `className` is the only escape hatch in v1.
- Avoid adding many visual override props.

## 9. Variant Behavior
### 9.1 Preview Variant
Used by:
- `HomeEntry`

Behavior:
- lighter visual treatment
- all steps visible
- optional descriptions may be shown
- no heavy “completed” emphasis for past steps
- serves as a path preview, not a live task tracker

### 9.2 Progress Variant
Used by:
- `Generator`
- `KnowledgeReview`

Behavior:
- current step gets strongest emphasis
- past steps get softer completed styling
- future steps remain quieter
- optional descriptions may be shown
- serves as a live orientation layer for the current page

## 10. File Plan
### 10.1 Create
- `frontend/src/components/shared/StepStrip.tsx`
- `frontend/src/components/shared/StepStrip.test.tsx` (recommended)

### 10.2 Modify
- `frontend/src/pages/HomeEntry.tsx`
- `frontend/src/pages/Generator.tsx`
- `frontend/src/pages/KnowledgeReview.tsx`

## 11. Page Responsibilities After Extraction
### 11.1 HomeEntry
- still owns `activePath`
- still gets flow data from `homeEntryViewState`
- passes:
  - `variant="preview"`
  - selected path steps
  - section title and description

### 11.2 Generator
- still owns generator step computation through `generatorViewState`
- still owns current-step summary and next-action text
- passes:
  - `variant="progress"`
  - generator step metadata
  - current step index

### 11.3 KnowledgeReview
- still owns review step computation through `knowledgeReviewViewState`
- still owns current-step summary and next-action text
- passes:
  - `variant="progress"`
  - review step metadata
  - current step index

## 12. Rendering Rules
- The component should internally choose grid layout by step count.
- Do not force each page to manage strip column classes manually.
- Descriptions should render only when present.
- If `currentStepIndex` is omitted:
  - `preview` still renders cleanly
  - `progress` should gracefully avoid broken highlighting

## 13. Testing Strategy
### 13.1 Component Tests
Recommended coverage:
- `preview` renders all provided steps
- `progress` visually distinguishes current and past steps
- descriptions render only when provided
- step count changes do not break rendering

### 13.2 Page Validation
- `HomeEntry`, `Generator`, and `KnowledgeReview` do not need deep duplicate tests of the shared visual internals.
- Validation focus for pages should remain:
  - correct step data
  - correct current step index
  - correct surrounding workflow behavior

## 14. Success Criteria
- No duplicated step-strip card JSX remains in the three pages.
- One shared `StepStripItem` type is used across all three pages.
- `HomeEntry` still feels like a lightweight preview.
- `Generator` and `KnowledgeReview` still feel like active in-progress workflows.
- The surrounding page logic stays unchanged.

## 15. Risks
- Over-abstracting the component could make the API harder to use than the duplicated JSX it replaces.
- Under-specifying the variant behavior could cause visual drift later.
- Mixing summary logic into the shared component would create tight coupling.

## 16. Mitigations
- Keep the API narrow.
- Keep only two variants.
- Keep page-specific summaries and CTA logic outside the component.
- Export the shared type from the same file in v1 to avoid unnecessary file sprawl.

## 17. Implementation Readiness
- This design is ready for implementation.
- The implementation should proceed as a narrow frontend refactor:
  - add `StepStrip`
  - replace duplicated strip JSX in three pages
  - run tests and build
