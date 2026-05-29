# InkWords StepStrip Visual Polish Design

## 1. Background
- `StepStrip` has already been extracted into a shared component used by:
  - `HomeEntry`
  - `Generator`
  - `KnowledgeReview`
- The current implementation is structurally correct and reusable.
- The main remaining problem is visual refinement:
  - `preview` and `progress` variants are still too close in appearance
  - current/past/future states work, but they do not yet feel intentional enough
  - the strip still reads a bit like generic bordered cards rather than a calm workflow orientation layer

## 2. Goal
- Polish `StepStrip` so it feels more elegant and editorial.
- Make `preview` and `progress` variants clearly different at a glance.
- Improve state hierarchy in `progress` mode:
  - completed
  - current
  - upcoming
- Keep the product tone minimal and calm.
- Preserve the existing component API and behavior.

## 3. Scope
### 3.1 In Scope
- Visual polish inside `frontend/src/components/shared/StepStrip.tsx`
- Spacing, typography, borders, surfaces, and subtle transition refinement
- Stronger visual differentiation between:
  - `preview`
  - `progress`
- Stronger visual differentiation between:
  - `preview`
  - `complete`
  - `current`
  - `upcoming`

### 3.2 Out of Scope
- No API change
- No workflow logic change
- No new page-level summary changes
- No new animation system
- No redesign of surrounding page layouts
- No new component split

## 4. Chosen Direction
- Style direction: `More Elegant`
- Motion level: `Minimal Motion`
- Visual approach: `Hierarchy-First Elegance`

## 5. Why This Direction
- The product aims for a calm, reading-friendly, process-oriented interface.
- A polished hierarchy improves clarity without adding noise.
- `StepStrip` is a support layer, not the hero of the page, so the refinement should improve confidence and scanability rather than draw too much attention.
- Minimal motion keeps the UI feeling mature and restrained.

## 6. Keep Stable
The following must remain unchanged:
- `StepStripItem`
- `StepStrip` public props
- `variant`
- `currentStepIndex`
- `className`
- page usage patterns in `HomeEntry`, `Generator`, and `KnowledgeReview`

This is a presentation-only refinement.

## 7. Visual Goal
- Make the strip feel calmer, cleaner, and more deliberate.
- Reduce the sense of “generic stacked cards”.
- Improve scannability of current workflow state.
- Keep the strip supportive rather than decorative.

## 8. Variant Behavior
### 8.1 Preview Variant
Used by:
- `HomeEntry`

Target feeling:
- calm roadmap
- lighter than progress mode
- informative, not status-heavy

Desired treatment:
- soft neutral or lightly tinted surface
- softer border contrast
- more breathing room
- smaller, quieter step index label
- muted description text
- no strong sense of completion state

### 8.2 Progress Variant
Used by:
- `Generator`
- `KnowledgeReview`

Target feeling:
- live orientation strip
- more purposeful than preview mode
- clearly indicates “where you are now”

Desired treatment:
- `current`
  - strongest border contrast
  - slightly richer surface tint
  - strongest title emphasis
  - may include a subtle accent treatment
- `complete`
  - visually resolved
  - softer than current
  - not visually dominant
- `upcoming`
  - neutral surface
  - calm and quiet
  - clearly less prominent than current

## 9. Typography
- Step index label:
  - smaller
  - lighter
  - more editorial than control-like
- Step title:
  - main semantic anchor
  - current step gets slightly stronger emphasis
- Step description:
  - compact
  - muted
  - should never dominate the card

## 10. Spacing
- Increase consistency in vertical spacing inside each card.
- Let cards breathe slightly more than the current version.
- Improve spacing between strip header and card grid.
- Preserve responsiveness and compactness for smaller screens.

## 11. Borders And Surfaces
- Prefer layered neutrals over hard contrast.
- Keep borders present but softer for non-current states.
- Use the strongest contrast only for the current step in progress mode.
- Shadows, if any, must remain extremely subtle.
- Avoid flashy gradients or bright glow effects.

## 12. Motion
- Minimal motion only:
  - gentle hover polish if already implied by the component context
  - light transition smoothing for background/border/color changes
- No shimmer
- No pulse
- No animated progress lines

## 13. Dark Mode
- Preserve the same hierarchy as light mode:
  - current remains strongest
  - complete remains softer
  - upcoming remains calm
- Avoid making dark mode cards feel luminous or overly contrasted.

## 14. File Plan
### 14.1 Modify
- `frontend/src/components/shared/StepStrip.tsx`

### 14.2 Keep Unchanged
- `frontend/src/pages/HomeEntry.tsx`
- `frontend/src/pages/Generator.tsx`
- `frontend/src/pages/KnowledgeReview.tsx`
- `frontend/src/pages/homeEntryViewState.ts`

## 15. Testing Strategy
- Keep the current component tests.
- Only update tests if rendered semantics or attributes change meaningfully.
- Avoid overfitting tests to low-level visual class names unless required.
- Focus validation on:
  - preview still renders correctly
  - progress still exposes state differences correctly
  - build still succeeds

## 16. Success Criteria
- `preview` and `progress` look clearly different at a glance
- `current` is easier to identify immediately in progress mode
- completed and upcoming states feel intentionally calmer than current
- the strip feels more polished without becoming flashy
- no page-level behavior changes are introduced

## 17. Risks
- Over-polishing could make the strip too visually loud for the rest of the product.
- Overusing tint or contrast could break the minimalist tone.
- Small visual changes could unintentionally reduce clarity in dark mode.

## 18. Mitigations
- Keep the changes surface-level and restrained.
- Treat `current` as the only strongly emphasized state.
- Keep `preview` lighter than all progress states.
- Validate both light and dark class combinations during implementation.

## 19. Implementation Readiness
- This design is ready for implementation.
- The implementation should be a narrow refactor contained within `StepStrip.tsx`, followed by focused tests and a frontend build.
