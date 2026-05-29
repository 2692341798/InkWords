# Review Entry Merge Design

## Goal

Fix the broken review-entry experience on the knowledge review page by:

- merging the duplicated `Start today review` and `Randomly pick one` cards into a single recommendation card
- making refresh actually switch to a different article when alternatives exist
- fixing the backend random picker so it is truly random instead of deterministically returning the first eligible note

## Current Problems

- The page exposes two entry cards that overlap in purpose, which makes the decision unclear.
- The backend `PickRandom` implementation is deterministic, so repeated picks can feel broken.
- The `today` recommendation endpoint is intentionally stable, so using it for refresh does not satisfy the user's expectation of seeing a different article.

## Chosen Approach

Use a single merged recommendation card plus the existing manual picker card.

- Initial load uses the `today` recommendation as the default displayed article.
- Refresh switches to a different article by using the random-pick path while excluding the currently displayed note when possible.
- Start action launches a session from whichever article is currently shown on the merged recommendation card.

## Frontend Changes

### Entry Card Layout

- Remove the separate random card from `ReviewEntryCards`.
- Rename the remaining automatic entry to a single recommendation-oriented card.
- Keep the manual picker card unchanged.

### State Model

- Replace the dual-card UI dependency with one displayed recommendation card in the review store.
- Track whether the displayed card came from the stable `today` load or from a refreshed random pick only if needed for button copy or analytics.

### Refresh Behavior

- First page load: show `today` recommendation.
- Refresh button: request a different article than the one currently displayed.
- If no alternative exists, keep the current article and avoid implying that rotation happened.

## Backend Changes

### Random Picker

- Update `PickRandom` to choose randomly from eligible notes that are not in the recent set.
- If every note is recent, choose randomly from the full candidate pool.
- Keep the behavior deterministic in tests by injecting or stubbing the random source where needed.

## Error Handling

- If recommendation loading fails, preserve current fallback empty-state copy.
- If refresh cannot find a different article because only one candidate exists, keep the current card and avoid a misleading state change.

## Testing

- Add backend tests to verify `PickRandom` can choose from multiple valid candidates instead of always returning the first one.
- Add frontend/store tests to verify:
  - the merged recommendation card is the only automatic entry card
  - refresh replaces the displayed article when an alternative exists
  - start action uses the currently displayed article

## Out Of Scope

- Redesigning the manual note picker
- Changing session/question flow
- Adding a brand-new recommendation ranking API
