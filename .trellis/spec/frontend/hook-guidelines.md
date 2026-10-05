# Hook Guidelines

> Custom hooks, SSE streaming, cancellation, and the long-task UX contract.

## Conventions

- **Hooks are named `useXxx`.** They are named exports and return an object of state and actions.
- **Existing hooks:**
  - `useBlogStream`, a facade over `hooks/generator/{useProjectScanner, useProjectAnalyzer, useFileParser, useSeriesGenerator}`
  - `useContinueStream`, `usePolishStream`
  - `useEditorAutosave` (2 s debounce, saves on unmount), `useDebounce`
  - `useKnowledgeReview`, `useSidebarBatchSelection`, `useSpeechRecognition`, `useSyncedScroll`, `useBatchExportZip`

  Extend an existing hook before adding a parallel one.
- **Hooks call services and stores; components call hooks.** Hooks read store actions with `useXxxStore.getState()` inside async callbacks to avoid stale closures. This pattern appears 32 times.
- **Every effect that subscribes, times, or listens returns a cleanup.** See `useDebounce`, `useSyncedScroll`, `useEditorAutosave` and `markdown/MermaidBlock.tsx`.
- **Data loading** is an imperative `useEffect` or event handler that calls a store action or service. There is no React Query or SWR; don't add one in a feature change.

## SSE streaming

All streams go through `fetchEventSourceWithAuth` in `src/services/sse.ts`. It:

- adds the Bearer token;
- runs `assertApiResponse` and checks for an `text/event-stream` content type in `onopen`;
- **always rethrows in `onerror`**, so streams never auto-retry. The source comment explains why: implicit replays of task mutations are unsafe, so retry must be an explicit user action with a new idempotency key.

Stream lifecycle:

1. Create the task: `createGenerationTask` (`services/generationTasks.ts`), or the domain equivalent.
2. Subscribe to `task.stream_url` with `fetchEventSourceWithAuth`.
3. Handle the events `chunk`, `progress`, `chapter`, `done` and `error`. Batch high-frequency `chunk` writes into the store with `lib/streamFlushBuffer.ts`.
4. End in an explicit terminal state on both `done` and `error`.

**Rules for new streaming code:**

- Pass `openWhenHidden: true`. Without it, fetch-event-source closes the stream when the tab is hidden and re-requests it when the tab is shown again.
- Own an `AbortController`. Keep it in the relevant store (`streamStore.abortController`) or in a `useRef` (`usePolishStream`), and abort it on stop **and** on unmount.
- To cancel a backend task, also call the cancel endpoint (best effort), as `useBlogStream.stopGenerating` does with `cancelGenerationTask`.
- Don't redefine `class StopStreamError` or match AbortError by message string inside the hook. If you need these, add one shared helper next to `services/sse.ts` or in `lib/` (`lib/polishStreamStop.ts` is the closest existing helper), and use it from the new code.
- Clean up any `setTimeout` you schedule after `done`.

## Long-task UX contract

`AGENTS.md` §8 says every long task must show progress, offer cancel and retry, show terminal failure, and keep refresh-safe state.

| Requirement | Existing reference |
|-------------|--------------------|
| Progress | `streamStore` `chapterStatus` / `chapterPhases`, `components/generator/GeneratorStatus.tsx` |
| Cancel | `useBlogStream.stopGenerating` (abort and backend cancel), `usePolishStream` stop |
| Terminal failure | `streamStore.chapterErrors` per chapter, `role="alert"` in `pages/HomeEntry.tsx` |
| Explicit retry | "重试加载" in `pages/HomeEntry.tsx`, covered by `e2e/review-failure.spec.ts` |
| Refresh-safe state | Review sessions only: `useKnowledgeReview` stores the active session ID under `inkwords-active-review-session` and resumes it |

New long-task UI must cover all five rows. Today, generation, course and export tasks lose `currentTaskId` and progress on refresh. A new long task should persist its task ID, as the review session does, and re-attach on load.

## Known debt (do not copy)

- `StopStreamError` is duplicated in `hooks/useContinueStream.ts`, `usePolishStream.ts` and `hooks/generator/{useSeriesGenerator,useProjectAnalyzer,useFileParser}.ts`.
- `useContinueStream` has no `AbortController`.
- No streaming hook aborts on unmount.
- `openWhenHidden: true` is missing in `useProjectScanner.ts`, `useContinueStream.ts`, and `streamProjectCourseTask` in `services/projectCourse.ts`.
- `useSeriesGenerator.ts` calls `setTimeout(() => reset(), 2000)` after `done` and never cleans it up.
