# Component Guidelines

> How components are written and styled.

## Shape

- **Function components with named exports:** `export function ReviewSessionCard(props) { ... }`. The only default exports are in `App.tsx` and in files loaded with `React.lazy` (`components/markdown/MermaidBlock.tsx`, `CodeBlock.tsx`, imported from `MarkdownEngine.tsx` inside `Suspense`).
- **File names:**
  - Feature components: PascalCase `.tsx`.
  - `ui/` primitives: kebab-case.
  - Non-component helpers: camelCase `.ts`.
- **Code style:** single quotes and no semicolons outside the shadcn-generated `ui/` files. There is no Prettier, so match the file you are editing.
- **Props:** declare an `interface XxxProps` above the component and destructure it in the signature, with defaults inline. This is the majority pattern, used in about 17 files, e.g. `components/review/ReviewSessionCard.tsx` and `components/generator/GeneratorInput.tsx`. Use `type XxxProps = HTMLAttributes<...> & {...}` only when you extend native element props (`ui/workspace.tsx`). Don't use inline object types for props in new components.

```tsx
interface ReviewSessionCardProps {
  session: ReviewSessionResponse | null
  pending?: boolean
  onRespond: (answer: string) => Promise<void> | void
}

export function ReviewSessionCard({ session, pending = false, onRespond }: ReviewSessionCardProps) { ... }
```

## Presentational vs connected components

- **Presentational components get data and callbacks through props.** Example: `review/ReviewSessionCard.tsx`, where the page owns the state and the card only renders it and calls `onRespond` / `onFinish`. Prefer this for anything you want to unit-test.
- **Connected components read a store directly.** Examples: `project-course/BlueprintWorkspace.tsx` (`useProjectCourseStore`) and `components/Sidebar.tsx`. Use a selector (see [State Management](./state-management.md#selectors)).
- **Pull branching UI logic into a pure helper:** `getReviewPhase` in `review/reviewPhase.ts`, `generatorViewState.ts`. Test the helper; keep the component thin.
- **No side effects during render.** Fetch in `useEffect` or in event handlers, never in the component body.

## Design system and styling

- **Tailwind 4 is CSS-first** and there is no `tailwind.config`. Tokens live in `src/index.css`:
  - `@theme inline` holds the token map.
  - `:root` / `.dark` hold the palettes.
  - `@layer components` holds semantic classes: `.surface-panel`, `.page-title`, `.choice-tile`, `.status-pill`, `.brand-pill`.
- **Combine classes with `cn()`** from `@/lib/utils` (clsx + tailwind-merge). Use `cva` for real variants, as in `ui/button.tsx`.
- **Reuse before inventing:** `Button` from `ui/button.tsx`; layout from `ui/workspace.tsx` (`PageShell`, `PageHeader`, `Panel`, `SectionHeader`, `StatusPill`); `ConfirmDialog` from `ui/confirm-dialog.tsx`; toasts from `sonner`.
- **Colors come from tokens** (`bg-background`, `text-muted-foreground`, `var(--brand)`). Don't use raw palette classes such as `zinc-*`, `indigo-*` or `bg-white`.
- **Icons** come from `lucide-react`.
- **UI copy is Chinese** and reuses existing vocabulary. Check how the same concept is already labelled before inventing a new term (e.g. "重试加载", "应用润色结果"). Keep code, commands and identifiers as they are.

## Known debt (do not copy)

- Hard-coded palette classes in `pages/Login.tsx` (43), `sidebar/BlogTreeDisplay.tsx` (22), `sidebar/StreamOutlineSection.tsx` (18) and `ui/confirm-dialog.tsx` (7).
- The `--brand*`, `--success*` and `--warning*` tokens exist only in `:root`. They have no `.dark` values and aren't registered in `@theme`. Dark mode is not wired up: `next-themes` is used only in `ui/sonner.tsx`, nothing toggles `.dark`, and there is no `ThemeProvider`.
- `ui/confirm-dialog.tsx` builds classes with template strings instead of `cn()`, and it has no dialog semantics (see [Quality Guidelines](./quality-guidelines.md#accessibility)).
- Inline prop types in `review/ReviewReadingView.tsx`, `ReviewProgress.tsx` and `ReviewRecallView.tsx`.
