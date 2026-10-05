# Type Safety

> TypeScript settings, where types live, and how untrusted data is handled.

## Compiler settings (`frontend/tsconfig.app.json`)

- `strict`, `noUnusedLocals`, `noUnusedParameters`, `noFallthroughCasesInSwitch` and `noUncheckedSideEffectImports` are on.
- `verbatimModuleSyntax` is on, so type-only imports must say so: `import type { X }` or inline `import { type X, value }` (pattern: `store/reviewStore.ts`).
- `erasableSyntaxOnly` is on, so there are **no `enum` and no `namespace`**. Use string-literal unions (`type ReviewMode = 'light_recall' | 'detailed_qa'`) and `as const` objects (`apiRoutes.ts`).
- The path alias `@/*` maps to `src/*`.
- `noUncheckedIndexedAccess` is **off**. Guard array and record lookups yourself.

## Where types live

There is no global `types/` folder. Put a type next to its owner:

| Type | Location | Example |
|------|----------|---------|
| API request/response DTOs | The service module that calls the endpoint | `services/review.ts` (`ReviewCardResponse`, `ReviewSessionResponse`, ...) |
| Store-owned domain shapes | The store | `store/streamStore.ts` (`Chapter`, `ModuleCard`) |
| Pure domain helpers and their types | `lib/` | `lib/projectCourse.ts` |
| Component props | The component file (`interface XxxProps`) | |

DTO naming:
- `XxxResponse` / `XxxInput` / `XxxRequest`.
- Fields stay snake_case to match the backend JSON (`note_path`, `estimated_minutes`). Don't add a camelCase mapping layer.

Before writing a new type, search for an existing one. Import the service DTO instead of redeclaring it.

## `any`, `unknown`, and runtime validation

- **No `any`.** It appears only in `MarkdownEngine.tsx` (remark/rehype nodes, with an eslint-disable) and in `vite.config.ts`.
- **Catch blocks** use `catch (err: unknown)` and narrow with `err instanceof Error ? err.message : '<中文兜底>'`.
- **No runtime validation library.** There is no zod, and responses are cast with `as T` inside the API helpers. For data that drives persistence or a security decision (stream payloads, scanned modules, model output), write a small `unknown`-based guard. Patterns: `normalizeModules` in `hooks/generator/useProjectScanner.ts` and `extractTaskChunkContent` in `services/generationTasks.ts`. Don't add zod in a feature change.

## Known debt (do not copy)

- `BlogServiceNode` (`services/blog.ts`) duplicates `BlogNode` (`store/blogStore.ts`), bridged with `as BlogNode[]`.
- `SeriesChapter` (`services/generationTasks.ts`) duplicates `Chapter` (`store/streamStore.ts`).
- `UserStats` / `UserProfile` in `pages/Dashboard.tsx` duplicate `UserStatsResponse` / `UserProfileResponse` in `services/user.ts`.
- The `currentView` union is written out twice in `store/blogStore.ts`.
- `evidence_ids` is typed in `lib/projectCourse.ts` but never rendered. Evidence UI is still missing (`AGENTS.md` §8).
