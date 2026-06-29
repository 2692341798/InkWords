# Code Cleanup Baseline

Captured on 2026-06-29 before the systematic cleanup. Unless noted otherwise, commands were run from the directory named in each section.

## Commit

- Repository root: `/Users/huangqijun/Documents/墨言博客助手/InkWords`
- Branch: `codex/governance-baseline`
- Base commit: `ff4c51696bbc032d0255af98ad545e8af7a3dce8`
- The untracked cleanup plan under `docs/superpowers/plans/` was intentionally left untouched and is not part of this baseline.

## Backend

Commands were run from `backend/`.

| Check | Result |
| --- | --- |
| `go test ./... -coverprofile=/tmp/inkwords-before.cover` | Passed for all packages. Packages without tests or statements were reported by Go as such. |
| `go tool cover -func=/tmp/inkwords-before.cover \| tail -n 1` | `total: (statements) 37.7%` |
| `go vet ./...` | Passed with exit code 0 and no findings. |

The first sandboxed `go test` attempt could not read the user-level Go build cache and failed with `operation not permitted`. Running the same command with the required filesystem permission produced the passing result above; this was an execution-environment restriction, not a repository test failure.

## Frontend

Commands were run from `frontend/`.

| Check | Result |
| --- | --- |
| `npm test` | Passed: 41 test files and 138 tests. Vitest reported a 1.72 s duration. |
| `npm run build` | Passed: TypeScript project build and Vite 8.0.3 production build; 6,862 modules transformed. |
| `npm run lint` | Failed at the known baseline: 5 findings, comprising 2 errors and 3 warnings. |

Lint findings:

- `src/hooks/useKnowledgeReview.test.tsx:60`: error, `react-hooks/immutability`.
- `src/hooks/usePolishStream.ts:28`: error, `react-hooks/refs`.
- `src/pages/Editor.tsx:64`: warning, `react-hooks/exhaustive-deps`.
- `src/pages/Editor.tsx:108`: warning, `react-hooks/exhaustive-deps`.
- `src/pages/Editor.tsx:156`: warning, `react-hooks/exhaustive-deps`.

Bundle observations from the successful build:

- Main application JavaScript (`dist/assets/index-B18yZ2xX.js`) was 1,805.56 kB minified and 585.91 kB gzip.
- Another generated JavaScript chunk (`dist/assets/chunk-K5T4RW27-Fc-gDzFD.js`) was 518.15 kB minified and 115.02 kB gzip.
- Vite warned that some chunks exceeded 500 kB after minification and suggested code splitting or adjusting the warning limit.
- The generated application CSS was 94.36 kB minified and 15.28 kB gzip.

Asset hashes in generated filenames may change between builds; the sizes above are the observations at the base commit.

## Known Structural Debt

### Exact duplicate production files

The following baseline scan hashes all non-test Go files under `backend/`, groups byte-for-byte identical files, prints every duplicate path, and calculates the summary totals:

```sh
rg --files backend -g '*.go' -g '!**/*_test.go' -g '!**/vendor/**' \
  | LC_ALL=C sort \
  | while IFS= read -r file; do shasum "$file"; done \
  | LC_ALL=C sort -k1,1 -k2,2 \
  | awk '
      function flush_group(    i) {
        if (group_size > 1) {
          duplicate_groups++
          duplicate_files += group_size
          printf "duplicate_group=%d sha1=%s files=%d\n", duplicate_groups, current_hash, group_size
          for (i = 1; i <= group_size; i++) print "  " group_paths[i]
        }
        delete group_paths
        group_size = 0
      }
      {
        hash = $1
        path = $0
        sub(/^[^[:space:]]+[[:space:]]+/, "", path)
        if (current_hash != "" && hash != current_hash) flush_group()
        current_hash = hash
        group_paths[++group_size] = path
      }
      END {
        flush_group()
        printf "duplicate_groups=%d\n", duplicate_groups
        printf "duplicate_files=%d\n", duplicate_files
      }
    '
```

It found **22 exact duplicate groups spanning 52 files**:

1. `internal/domain/user/handler.go` and `services/core-api/domain/user/handler.go`
2. `internal/infra/parser/git_fetcher_github.go`, `services/parser-service/infra/parser/git_fetcher_github.go`, and `shared/platform/parser/git_fetcher_github.go`
3. `internal/infra/llm/output_sanitize.go` and `shared/platform/llm/output_sanitize.go`
4. `internal/domain/project/dto.go` and `services/core-api/domain/project/dto.go`
5. `internal/domain/stream/dto.go` and `services/llm-stream/domain/stream/dto.go`
6. `internal/domain/review/handler.go` and `services/review-service/domain/review/handler.go`
7. `internal/infra/parser/git_fetcher_git.go`, `services/parser-service/infra/parser/git_fetcher_git.go`, and `shared/platform/parser/git_fetcher_git.go`
8. `internal/infra/parser/git_fetcher.go`, `services/parser-service/infra/parser/git_fetcher.go`, and `shared/platform/parser/git_fetcher.go`
9. `internal/infra/llm/deepseek.go` and `shared/platform/llm/deepseek.go`
10. `internal/infra/parser/git_fetcher_cache.go`, `services/parser-service/infra/parser/git_fetcher_cache.go`, and `shared/platform/parser/git_fetcher_cache.go`
11. `internal/infra/parser/git_fetcher_filter.go`, `services/parser-service/infra/parser/git_fetcher_filter.go`, and `shared/platform/parser/git_fetcher_filter.go`
12. `internal/domain/review/history_service.go` and `services/review-service/domain/review/history_service.go`
13. `internal/domain/blog/dto.go` and `services/core-api/domain/blog/dto.go`
14. `internal/domain/review/frontmatter.go` and `services/review-service/domain/review/frontmatter.go`
15. `internal/infra/parser/archive_parser.go`, `services/parser-service/infra/parser/archive_parser.go`, and `shared/platform/parser/archive_parser.go`
16. `internal/infra/parser/doc_parser.go`, `services/parser-service/infra/parser/doc_parser.go`, and `shared/platform/parser/doc_parser.go`
17. `internal/infra/parser/git_fetcher_types.go`, `services/parser-service/infra/parser/git_fetcher_types.go`, and `shared/platform/parser/git_fetcher_types.go`
18. `internal/domain/project/handler.go` and `services/core-api/domain/project/handler.go`
19. `internal/domain/review/picker.go` and `services/review-service/domain/review/picker.go`
20. `internal/domain/task/export_task.go` and `services/core-api/domain/task/export_task.go`
21. `internal/domain/project/source_assembler.go` and `services/core-api/domain/project/source_assembler.go`
22. `internal/domain/stream/generation_result.go` and `services/llm-stream/domain/stream/generation_result.go`

All paths in this list are relative to `backend/`. Test files were excluded so the baseline measures duplicated production implementation rather than mirrored tests.

### Service-to-monolith dependency

A direct-import scan (`rg -n '"inkwords-backend/internal/service"' backend/services --glob '*.go'`) found these service-layer files still depending on the monolithic `internal/service` package:

- `backend/services/core-api/app/bootstrap/bootstrap.go`
- `backend/services/llm-stream/app/bootstrap/bootstrap.go`
- `backend/services/llm-stream/domain/stream/service.go`

Thus, both `core-api` and `llm-stream` retain direct dependencies on `internal/service` at this baseline.
