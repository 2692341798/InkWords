# Quality Guidelines

> Lint, tests, e2e, accessibility, and the commands that define "done" for frontend changes.

## Commands

Run in `frontend/`:

```bash
npm test
```

```bash
npm run lint
```

```bash
npm run build
```

Optional checks:
- `npm run deadcode` runs knip.
- `npm run check:bundle` checks the bundle budget.
- `npm run test:coverage` runs coverage.
- `npm run test:e2e` runs Playwright `@core`.

## Lint and static checks

- **ESLint** (`frontend/eslint.config.js`) uses `js.recommended`, `typescript-eslint` recommended, `react-hooks` flat recommended and `react-refresh` vite. `npm run lint` runs with `--max-warnings=0`, so a warning fails the build. There is no a11y lint plugin, which means accessibility has to be checked by hand (see below).
- **knip** (`frontend/knip.json`) ignores only `src/components/ui/**`. Unused exports, files or dependencies elsewhere fail it.
- **Bundle budget** (`frontend/scripts/check-bundle-budget.mjs`): there must be exactly one `index-*.js` chunk, at most 1,160,000 bytes raw and 355,000 bytes gzip. Load heavy, optional renderers with `React.lazy`, as `MermaidBlock` and `CodeBlock` already are.

## Unit and component tests (vitest)

- **Test placement:** tests sit next to the file they cover (`Foo.tsx` → `Foo.test.tsx`).
- **Environment:** vitest is configured in `vite.config.ts` with no `setupFiles` and no global DOM. A DOM test opts in with `// @vitest-environment jsdom` on line 1 (e.g. `pages/HomeEntry.test.tsx`, `hooks/useKnowledgeReview.test.tsx`).
- **Pure logic first:** view-state helpers, `lib/*`, stores and services get plain unit tests. This is most of the 46 test files.
- **Component tests (majority pattern):** `renderToStaticMarkup` plus assertions on the HTML string. See `components/review/ReviewSessionCard.test.tsx` and `GeneratorStageViews.test.tsx`.
- **Interactive behaviour:** `@testing-library/react` (`render`, `screen`, `waitFor`) under jsdom, as in `pages/HomeEntry.test.tsx`. `user-event` is not installed, and no test simulates events yet. If you need one, use `fireEvent` from `@testing-library/react` rather than adding a dependency.
- **Mocking:**
  - `vi.hoisted` plus `vi.mock('@/services/...')` / `vi.mock('@/store/...')`.
  - `vi.stubGlobal('fetch' | 'localStorage', ...)`.
  - The `fetchImpl` option for API helpers (`services/apiClient.test.ts`).
- **Coverage thresholds** (`vite.config.ts`): statements 38, branches 65, functions 54, lines 38. `ui/**` is excluded.

## Guardrail tests

Repo-level tests scan the source and fail on forbidden patterns. Keep them passing; don't weaken them.

- `src/task4Guardrails.test.ts` forbids `alert`, `confirm`, `location.reload`, and `location.href` except for the OAuth jump. Use `ConfirmDialog` and toasts instead.
- `src/gatewayRouting.test.ts` checks that every API prefix is routed by nginx and the Vite proxy.
- `src/vitePermissionCompatibility.test.ts` and `src/indexCssMarkdownProse.test.ts` guard build and CSS invariants.

When a new rule can be checked mechanically, add a guardrail test rather than relying only on prose.

## E2E (Playwright)

- **Config:** `frontend/playwright.config.ts` defines the projects `chromium`, `mobile-chromium` (Pixel 7), `firefox` and `webkit`. Locale is `zh-CN`, and the web server runs `npm run dev` with `VITE_AUTH_BYPASS=true`.
- **Fixture:** `e2e/fixtures/app.ts` `appPage` stubs every `/api/v1/**` request with envelopes, returns 404 for anything not stubbed, and **fails the test on any console error or page error**.
- **Tags:**
  - `@core` runs with `npm run test:e2e`.
  - `@cross-browser` runs with `npm run test:e2e:smoke`.
  - `@deepseek`, `@obsidian` and `@oauth` (`e2e/real-integrations.spec.ts`) need `E2E_EXTERNAL_MODE=real` and `E2E_TEST_TOKEN`. They are opt-in; never report them as passed when they were skipped.
- **Selectors:** use roles and Chinese accessible names (`getByRole('button', { name: '重试加载' })`), not CSS classes.
- **Expectation:** meaningful UI changes get a component test, plus a browser flow when one is available.

## Accessibility

Required for new UI (`AGENTS.md` §8): keyboard access, visible focus, semantic labels, and alt text.

Patterns to copy:
- **Review views:** `aria-labelledby`, `aria-live="polite"`, `aria-current="step"`, and `aria-busy` on a disabled `fieldset` while pending (`review/ReviewSessionCard.tsx`).
- **HomeEntry:** `role="alert"` for terminal errors and `aria-pressed` for toggles (`pages/HomeEntry.tsx`).
- **BlueprintWorkspace:** `sr-only` labels on form controls (`project-course/BlueprintWorkspace.tsx`).

Rules:
- Interactive elements are `<button>` / `<a>` / form controls, never a bare `<div onClick>`.
- Dialogs need `role="dialog"`, `aria-modal`, a labelled title, Escape to close, and focus that moves in and is restored afterwards.
- `alt` text is Chinese and describes the content.

**Known debt (do not copy):**
- `sidebar/BlogTreeDisplay.tsx` uses clickable `<div>`s with no role or keyboard support.
- `ui/confirm-dialog.tsx` has no dialog role, focus trap or Escape handling.
- `pages/Dashboard.tsx` uses `alt="Avatar"`.
- The whole app has only one `onKeyDown`.

## Manuscript UI requirements

Manuscript and revision UI must show revision status, lock status, evidence, verification state ("generated" is not "verified"), and a diff against the candidate. None of this exists yet. The closest pieces are the blueprint `expected_version` save (`projectCourseStore`) and the polish preview tab in `components/editor/EditorBody.tsx` ("应用润色结果"), which shows the draft without a diff. New manuscript work must add these and must not imply that a candidate has been applied or verified.
