# Enhance Generation and UI Spec

## Why
The user reported three issues during the generation of the "GSAP plugin ecosystem" series:
1. **Incomplete Content**: The generated content for the second part was incomplete, either due to token limits or the LLM stopping early.
2. **Hardcoded Series Title**: The series title was hardcoded as "Git源码解析系列" instead of being dynamically named based on the actual topic (e.g., "GSAP插件生态系列").
3. **UI Stretching**: The generated blog outline list in both the main Generator view and the Sidebar stretches the page vertically, causing a poor user experience.

## What Changes
- **Topic Naming (AI Extract + User Edit)**:
  - Update the `GenerateOutline` prompt in the backend to return a JSON object containing both `series_title` and the `chapters` array.
  - Update the frontend `streamStore` to capture and store the `seriesTitle`.
  - Add an input field in `Generator.tsx` allowing the user to edit the `seriesTitle` before starting the generation.
  - Pass the custom `seriesTitle` to the backend `/api/v1/stream/generate` endpoint to replace the hardcoded "Git 源码解析系列".
- **Scrollbar UI Fixes**:
  - Add `max-h-[30vh]` and `overflow-y-auto` (with custom scrollbar styling if needed) to the "当前生成任务" (Current generating tasks) list in `Sidebar.tsx`.
  - Add `max-h-[40vh]` and `overflow-y-auto` to the outline list in `Generator.tsx`.
- **Incomplete Content Fix (Prompt + Continue Button)**:
  - **Backend Optimization**: Update the prompt in `GenerateSeries` to emphasize completeness, comprehensive coverage, and ensuring a proper conclusion.
  - **Frontend "Continue" Feature**: Add a "继续生成" (Continue Generating) button in the `Editor.tsx` toolbar.
  - **Backend Continue API**: Create a new API endpoint `/api/v1/blog/:id/continue` that reads the current blog content, prompts the LLM with "请继续完成上文未写完的内容", streams the output, and appends it to the existing blog content in the database.

## Impact
- Affected specs: Outline Generation, Series Generation, Blog Editing.
- Affected code:
  - `backend/internal/service/decomposition.go` (GenerateOutline, GenerateSeries)
  - `backend/internal/api/stream.go` & `backend/internal/api/blog.go` (New endpoint, parsing payload)
  - `frontend/src/store/streamStore.ts` (State for series title)
  - `frontend/src/components/Generator.tsx` (Title input, Scrollbar)
  - `frontend/src/components/Sidebar.tsx` (Scrollbar)
  - `frontend/src/components/Editor.tsx` (Continue button)

## ADDED Requirements
### Requirement: Dynamic Series Title
The system SHALL automatically extract a series title during analysis and allow the user to edit it before generating the chapters.

### Requirement: Continue Generating
The system SHALL provide a "Continue Generating" action for existing blogs to append missing content if the LLM output was truncated.

## MODIFIED Requirements
### Requirement: UI Layout
The outline lists in the Sidebar and Generator SHALL have a fixed maximum height and vertical scrollbars to prevent page stretching.
