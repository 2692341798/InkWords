# Review Question Entry Button Design

## Goal

Add a clear `提问开始` entry button to the knowledge review entry area so users can start a recommendation-based review session directly in `细致提问` mode.

## Current Problem

- The knowledge review entry area exposes `用这篇开始`, `再抽一篇`, and `打开候选列表`, but no explicit action labeled around asking questions.
- The product already supports `细致提问` as a review mode, yet that mode is only visible after the user moves deeper into the flow.
- Users who expect a direct "questioning" entry interpret the current UI as if the feature is missing.

## Chosen Approach

Add a third action button named `提问开始` to the existing recommendation card in the knowledge review entry area.

- `用这篇开始` keeps the current behavior and starts the session in the currently selected mode.
- `提问开始` switches the selected mode to `detailed_qa` immediately before starting the session with the current recommendation card.
- `再抽一篇` keeps the current behavior and only refreshes the recommendation card.

## UI Changes

### Recommendation Card Actions

- Keep the current recommendation card structure in `ReviewEntryCards`.
- Change the action area from a two-button layout to a three-action layout:
  - `用这篇开始`
  - `提问开始`
  - `再抽一篇`
- Keep the manual picker card unchanged to avoid widening this change beyond the requested entry point.

### Interaction Copy

- The new button label is `提问开始`.
- The button appears only on the recommendation card, not on the manual picker card.
- No additional helper text is required because the button label already makes the mode intent explicit.

## Behavior Changes

### Start In Question Mode

- When the user clicks `提问开始`, the page must:
  - ensure a recommendation card exists, reusing the current lazy-load behavior if needed
  - set `selectedMode` to `detailed_qa`
  - start the session with the current recommendation card through the existing `startSession` flow

### Existing Flow Preservation

- `用这篇开始` must remain unchanged.
- Session mode still locks after the session starts.
- No backend API or payload changes are required because the frontend already sends mode as part of session creation.

## Implementation Scope

### Files Expected To Change

- `frontend/src/components/review/ReviewEntryCards.tsx`
- `frontend/src/pages/KnowledgeReview.tsx`
- related frontend tests covering the entry card and page wiring

### Out Of Scope

- Redesigning the review page layout
- Adding a separate homepage-level `提问` button
- Changing manual picker behavior
- Changing session mode locking after a session starts
- Changing backend review APIs

## Error Handling

- If no recommendation card is currently loaded, `提问开始` follows the same fallback as `用这篇开始`: load the recommendation first, then continue only if a card exists.
- If recommendation loading fails, the existing empty-state and failure behavior remain unchanged.

## Testing

- Add or update a component test to verify the recommendation card renders `提问开始`.
- Add or update a page-level test to verify clicking `提问开始` sets the mode to `detailed_qa` before starting the session.
- Keep existing tests around mode locking intact because this change does not alter in-session behavior.
