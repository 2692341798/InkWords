# Tasks

- [x] Task 1: Update Outline Generation for Series Title
  - [x] SubTask 1.1: Modify `GenerateOutline` prompt in `backend/internal/service/decomposition.go` to return `{"series_title": "...", "chapters": [...]}` instead of a plain array.
  - [x] SubTask 1.2: Update parsing logic in `GenerateOutline` and `AnalyzeStream` to handle the new JSON structure.
  - [x] SubTask 1.3: Update `streamStore.ts` in frontend to store `seriesTitle`.
  - [x] SubTask 1.4: Update frontend `useBlogStream.ts` to parse `seriesTitle` from the SSE event payload.

- [x] Task 2: Frontend UI for Series Title and Scrollbars
  - [x] SubTask 2.1: Add an editable input field for `seriesTitle` in `Generator.tsx` above the "开始生成" button.
  - [x] SubTask 2.2: Add `max-h-[40vh] overflow-y-auto custom-scrollbar` to the outline list container in `Generator.tsx`.
  - [x] SubTask 2.3: Add `max-h-[30vh] overflow-y-auto custom-scrollbar` to the "当前生成任务" list container in `Sidebar.tsx`.
  - [x] SubTask 2.4: Update `GenerateSeries` backend API (`stream.go` and `decomposition.go`) to accept `series_title` in the request body and use it when creating the parent blog record.

- [x] Task 3: Incomplete Content Handling (Prompt + Continue API)
  - [x] SubTask 3.1: Enhance the `GenerateSeries` prompt in `decomposition.go` to strongly instruct the LLM to complete the article and not truncate.
  - [x] SubTask 3.2: Add a new backend API endpoint `POST /api/v1/blogs/:id/continue` in `blog.go` that fetches the blog, prompts the LLM to continue, streams the response, and appends to the DB.
  - [x] SubTask 3.3: Add a "继续生成" (Continue Generating) button in `Editor.tsx` (next to the Save/Delete buttons) that calls the continue API and appends the stream to the editor content.

# Task Dependencies
- Task 2 depends on Task 1
- Task 3 can be done in parallel
