# State Management

> zustand stores, what belongs where, and how state survives refresh.

## Stores (as-is)

| Store | File | Holds |
|-------|------|-------|
| `useBlogStore` | `src/store/blogStore.ts` | Blog tree, `selectedBlog`, `currentView` (the view switch) |
| `useStreamStore` | `src/store/streamStore.ts` | Generator workflow: chapters, `chapterStatus` / `chapterPhases` / `chapterErrors`, `abortController`, `currentTaskId`, chunk buffers |
| `useReviewStore` | `src/store/reviewStore.ts` | Knowledge-review recommendation, notes, current session, feedback |
| `useProjectCourseStore` | `src/store/projectCourseStore.ts` | Project course, blueprint, versioned save |

## Store shape

Follow `src/store/reviewStore.ts`:

- One `interface XxxState` that lists data fields first, then actions.
- `export const useXxxStore = create<XxxState>((set, get) => ({ ... }))`, with no middleware and no slices.
- Async actions live in the store and call `src/services/*`. Loading flags are named `isLoadingXxx`.
- A short `// Why:` comment explains why the store exists (see the comment above `useReviewStore`).
- Provide `reset()` / `clearXxxState()` actions for teardown.

## What goes where

| State | Location |
|-------|----------|
| Shared across pages or between a page and a distant component | zustand store |
| One page's form inputs, open/closed toggles, hover | `useState` in the component |
| Derived values | Compute during render or in a pure helper (`pages/*ViewState.ts`). Don't store them. |
| Server data used by one component only | A hook or local state. Add to a store only when a second consumer appears. |

Don't create a new store for a feature that fits an existing one. Don't add more fields to `streamStore`, which already has about 40, for something that isn't part of the generator workflow.

## Selectors

Select only what you render, using `useShallow` from `zustand/react/shallow` (pattern: `pages/Generator.tsx`, `components/generator/*`):

```ts
const store = useStreamStore(
  useShallow((state) => ({
    sourceType: state.sourceType,
    isGenerating: state.isGenerating,
    // ...only the fields this component renders
  })),
)
```

**Known debt:** whole-store destructuring `const { ... } = useXxxStore()` re-renders on every change. It appears in `App.tsx`, `components/Sidebar.tsx`, `pages/Editor.tsx`, `pages/HomeEntry.tsx`, `pages/KnowledgeReview.tsx`, `project-course/BlueprintWorkspace.tsx` and `hooks/useKnowledgeReview.ts`. Don't copy it.

## Optimistic updates and versioned writes

- **Revision, blueprint or approval writes send the expected version.** On conflict, reload instead of overwriting. Reference: `saveBlueprint` in `projectCourseStore` sends `blueprint_version` as `expected_version`, and the backend's CAS path returns 409.
- **Optimistic updates must roll back on failure.** **Known debt:** `blogStore.updateBlog` updates optimistically, never reverts, and only `console.error`s on failure.

## Persistence

There is no zustand `persist` middleware. Persistence is explicit and narrow:

- **Auth token:** localStorage key `token`, through `src/lib/authTokenStore.ts`. It is a `useSyncExternalStore` source with cross-tab `storage` sync. Always read and write the token through `authTokenStore`, never through `localStorage` directly.
- **Active review session:** localStorage key `inkwords-active-review-session` (`hooks/useKnowledgeReview.ts`).

Persist only IDs that let you re-fetch from the backend, never content. The manuscript is the single source of truth on the server. Wrap storage access so a missing or blocked `localStorage` doesn't crash the page.
