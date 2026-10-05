# Directory Structure

> Where frontend code lives in `frontend/src/`.

## Layout (as-is)

```text
src/
├── App.tsx                 # auth gate + view switch (no router)
├── main.tsx
├── index.css               # Tailwind 4 entry, theme tokens, semantic component classes
├── pages/                  # one component per view, plus pure view-state helpers
│   ├── HomeEntry.tsx  Generator.tsx  Editor.tsx  Dashboard.tsx
│   ├── KnowledgeReview.tsx  ProjectCourse.tsx  Login.tsx
│   └── generatorViewState.ts  homeEntryViewState.ts  knowledgeReviewViewState.ts (+ .test.ts)
├── components/
│   ├── ui/                 # design-system primitives (see below)
│   ├── editor/ generator/ markdown/ project-course/ review/ shared/ sidebar/
│   ├── Sidebar.tsx         # legacy top-level component
│   └── MarkdownEngine.tsx  # legacy top-level component
├── hooks/                  # useXxx hooks; hooks/generator/ holds generator sub-hooks
├── services/               # API layer: apiClient, apiRoutes, sse, one module per domain
├── store/                  # zustand stores
├── lib/                    # pure utilities + authTokenStore
└── *.test.ts               # repo-level guardrail tests (task4Guardrails, gatewayRouting, ...)
```

There is no `src/types/`. Types live next to their owner (see [Type Safety](./type-safety.md)).

## Routing: view switching, not URLs

There is no router; `react-router` is not a dependency. `src/App.tsx` chooses what to render:

1. If `authTokenStore` (read with `useSyncExternalStore`) has no token, it renders `<Login/>`. `App.tsx` reads `?token=` after OAuth and removes it with `history.replaceState`.
2. If `useBlogStore().selectedBlog` is set, it renders `<Editor key={selectedBlog.id}/>`.
3. Otherwise `currentView` (`'home-entry' | 'generator' | 'dashboard' | 'knowledge-review' | 'project-course'`) picks the page.

To add a page, add a `currentView` member in `store/blogStore.ts` and a branch in `App.tsx`. Don't add a router library in a feature change. Views aren't deep-linkable and a reload returns to `home-entry`, so any state that must survive a refresh has to be restored explicitly (see [State Management](./state-management.md#persistence)).

## `components/ui/`

It contains only `button.tsx` (shadcn, cva), `dropdown-menu.tsx`, `sonner.tsx`, and two hand-written files:
- `confirm-dialog.tsx`
- `workspace.tsx`, which exports `PageShell`, `PageHeader`, `Panel`, `SectionHeader` and `StatusPill`.

There are no Input, Dialog, Card or Tabs primitives yet. Before adding one, check whether `workspace.tsx` or a semantic class in `index.css` already covers it. If you add a primitive, put it in `ui/` and name the file in kebab-case (shadcn convention).

## Placement rules

| Code | Goes in |
|------|---------|
| A full screen selected by `currentView` | `pages/` |
| Pure function deriving view state from store data | `pages/<page>ViewState.ts` + `.test.ts`, or `components/<feature>/<name>.ts` (e.g. `review/reviewPhase.ts`) |
| Feature UI | `components/<feature>/` |
| Reusable stateful behavior | `hooks/` |
| HTTP/SSE calls | `services/` |
| Cross-page state | `store/` |
| Pure helpers without React | `lib/` |
| Tests | Next to the file under test: `Foo.tsx` → `Foo.test.tsx`. No `__tests__/` folders. |
| e2e specs | `frontend/e2e/*.spec.ts`, fixtures in `frontend/e2e/fixtures/` |

Import from `src` with the `@/` alias (`@/services/review`, `@/lib/utils`).
