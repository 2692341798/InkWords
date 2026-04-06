# Tasks
- [x] Task 1: Fix Generator.tsx UI issues
  - [x] SubTask 1.1: Change the hardcoded worker ID array `[0,1,2,3,4]` to `Object.keys(store.workers).map(Number)` for rendering.
  - [x] SubTask 1.2: Adjust grid classes to auto-wrap smoothly (e.g. `grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-3`).
  - [x] SubTask 1.3: Update Step 2 default label to "并发分析代码分块...".
- [x] Task 2: Enhance LLM Prompts in `decomposition.go`
  - [x] SubTask 2.1: Add `gitURL` to the series generation prompt template if it is provided.
  - [x] SubTask 2.2: Pass the next chapter's title to the prompt to fix the mismatched preview at the end of the article.

# Task Dependencies
- [Task 2] depends on [Task 1]
