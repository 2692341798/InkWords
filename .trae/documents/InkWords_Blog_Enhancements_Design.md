# InkWords Blog Generation & Concurrency Enhancements Design

## 1. Overview
The goal of this enhancement is to improve the concurrency analysis visualization, provide a flexible and editable blog outline, and adjust the blog generation prompts to support an unlimited number of detailed technical posts with rich code examples and images.

## 2. Concurrency Animation Enhancements
- **Backend Changes**: Update `mapReduceAnalyze` in `backend/internal/service/decomposition.go`. Introduce a `worker_id` (0-4) pool. Assign an ID to each of the 5 goroutines and include `worker_id` in the `sendProgress` payload.
- **Frontend Changes**: Update `MapReduceProgress` in `store/streamStore.ts` to manage a record of `workers` (`Record<number, MapReduceProgress>`).
- **UI Updates**: In `Generator.tsx`, display 5 "Worker Slots". Active workers will have a pulsing border/background and show the currently analyzing chunk directory, index, and status.

## 3. Editable Outline
- **Frontend State**: Add `updateChapter`, `addChapter`, `removeChapter`, and `moveChapter` actions to `store/streamStore.ts`.
- **UI Updates**: Replace the static outline display in `Generator.tsx` with an inline editable list. Each chapter card will feature:
  - Editable `input` for the title.
  - Editable `textarea` for the summary.
  - Action buttons: "Up", "Down", "Delete".
  - A global "Add Chapter" button at the end of the list.

## 4. Prompt Adjustments (LLM)
- **Outline Generation (`decomposition.go`)**: Replace the "5-10 篇" limit. Instruct the LLM to create one blog post per technical point, with no upper limit on the total number of posts. The outline should be split as detailed as necessary depending on the project size.
- **Blog Generation (`generator.go`)**: Instruct the LLM to provide more code examples and images for each technical point to explain it in greater detail, maintaining the "小白友好" (beginner-friendly) requirement.

## 5. Security & Constraints
- Ensure prompt updates still respect the 300,000 character limits.
- Ensure the newly added `worker_id` logic uses thread-safe channel operations to avoid data races.
