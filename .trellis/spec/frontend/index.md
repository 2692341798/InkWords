# Frontend Development Guidelines

> How the InkWords React frontend (`frontend/`) is actually built, and the rules new code must follow.

**Stack:**
- React 19, Vite 8, TypeScript 5.9 (strict)
- Tailwind CSS 4 (CSS-first) with shadcn `base-nova` / `@base-ui/react`
- zustand 5 for state
- `@microsoft/fetch-event-source` for SSE
- vitest and Testing Library for unit/component tests, Playwright for e2e
- ESLint 9 and knip

Product invariants, the writing contract, and safety rules live in the root `AGENTS.md`. The files below turn its §8 (Frontend) into concrete guidance and record where the current code doesn't match yet.

## How to read these files

- **As-is:** today's pattern, with file paths. Follow it so new code looks native.
- **Rules for new code:** the `AGENTS.md` target that new or touched code must meet.
- **Known debt (do not copy):** existing deviations. Don't spread them, and don't fix them inside an unrelated change.

## Guidelines Index

| Guide | Description |
|-------|-------------|
| [Directory Structure](./directory-structure.md) | `src/` layout, view switching instead of a router, co-located tests |
| [Component Guidelines](./component-guidelines.md) | Exports, props, naming, `cn()`, design-system primitives, styling tokens |
| [Hook Guidelines](./hook-guidelines.md) | Custom hooks, SSE streaming, abort/cancel, long-task UX contract |
| [State Management](./state-management.md) | zustand stores, selectors, persistence, local vs global state |
| [API Layer](./api-layer.md) | `apiClient`, `apiRoutes`, envelopes, errors, auth token |
| [Type Safety](./type-safety.md) | tsconfig flags, type placement, DTO naming, runtime validation |
| [Quality Guidelines](./quality-guidelines.md) | Lint, unit/component tests, guardrail tests, e2e, accessibility, bundle budget |

## Pre-development checklist

1. Find the page in `src/pages/` and the store or hook it already uses. Extend those before creating new ones.
2. Need an API call? Add the route to `src/services/apiRoutes.ts` and a function in the owning `src/services/*.ts`.
3. Building a long task (generation, export, course, verification)? Read [Hook Guidelines](./hook-guidelines.md#long-task-ux-contract).
4. Building manuscript or revision UI? It must show revision status, lock status, evidence, verification, and candidate diff (`AGENTS.md` §8).
5. Run `npm test`, `npm run lint`, and `npm run build` in `frontend/` before declaring done.
