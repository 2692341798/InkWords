# InkWords Project Introduction

Audience: you have Spring Boot experience, are new to Go, and have previous React experience that needs refreshing.

## 1. Big Picture

InkWords is an AI-assisted technical blog generation platform.

The product flow is:

1. User logs in.
2. User uploads files, ZIP courseware, or enters a Git repository URL.
3. Backend parses the source material.
4. Backend asks DeepSeek to analyze it and produce an outline.
5. User reviews or edits the outline.
6. Backend streams generated blog content through SSE.
7. Blogs are saved into PostgreSQL and shown in the editor/sidebar.
8. User can edit, polish, export Markdown/PDF, or export to Obsidian.

The repository is a monorepo:

- `backend/`: Go + Gin API server
- `frontend/`: React + TypeScript + Vite UI
- `docker-compose.yml`: Postgres, Redis, backend, frontend/Nginx, Obsidian bridge
- `.trae/documents/`: product, architecture, API, and database documentation

## 2. Backend Mental Model

Backend entry point:

- `backend/cmd/server/main.go`

If you come from Spring Boot, map the concepts like this:

| Spring Boot | This Go Project |
| --- | --- |
| `@SpringBootApplication` | `cmd/server/main.go` |
| `@RestController` | Gin handlers in `internal/domain/*/handler.go` and `internal/transport/http/v1/api` |
| `@Service` | `internal/domain/*/service.go` and legacy `internal/service/*` |
| `@Repository` | `internal/domain/*/repository.go` |
| JPA Entity | GORM models in `internal/model` |
| Filter / Interceptor | Gin middleware in `internal/transport/http/middleware` |
| `application.yml` | environment variables / `.env` |
| Maven/Gradle module | Go module in `backend/go.mod` |

Important backend directories:

- `backend/cmd/server/main.go`: bootstraps DB, Redis, services, handlers, routes
- `backend/internal/transport/http/v1/routes.go`: all HTTP routes
- `backend/internal/domain/`: newer vertical slices: auth, user, blog, stream, project
- `backend/internal/service/`: core generation, decomposition, export logic
- `backend/internal/infra/`: DB, Redis, parser, DeepSeek client
- `backend/internal/model/`: GORM database models
- `backend/internal/prompt/`: prompt templates and scenario modes
- `backend/pkg/`: reusable helper packages

Core backend technologies:

- Go `1.25.4`
- Gin for HTTP routing
- GORM for PostgreSQL ORM
- Redis for cache/state support
- JWT and GitHub OAuth
- Server-Sent Events for streaming
- Goroutines, channels, and context cancellation for concurrent generation
- DeepSeek API client in `backend/internal/infra/llm/deepseek.go`

## 3. Frontend Mental Model

Frontend entry point:

- `frontend/src/App.tsx`

Important frontend directories:

- `frontend/src/pages/`: page-level views: Login, Generator, Editor, Dashboard
- `frontend/src/components/`: reusable UI components
- `frontend/src/hooks/`: workflow logic
- `frontend/src/store/`: Zustand global state
- `frontend/src/services/sse.ts`: authenticated SSE helper
- `frontend/src/lib/`: pure utilities and business helpers

Core frontend technologies:

- React `19.2.4`
- TypeScript `5.9`
- Vite `8`
- Tailwind CSS `4`
- Zustand for global state
- Shadcn-style UI primitives
- `@microsoft/fetch-event-source` for SSE
- `react-markdown`, Mermaid, Recharts, JSZip

Main frontend stores:

- `frontend/src/store/blogStore.ts`: blog tree, selected blog, CRUD
- `frontend/src/store/streamStore.ts`: generation workflow, outline, progress, SSE state

## 4. Most Important Request Flow

Generation flow:

1. User interacts with `frontend/src/pages/Generator.tsx`.
2. Frontend calls `/api/v1/stream/analyze` via `frontend/src/hooks/generator/useProjectAnalyzer.ts`.
3. Backend route is registered in `backend/internal/transport/http/v1/routes.go`.
4. Backend stream handler receives the request in `backend/internal/domain/stream/handler.go`.
5. Stream service calls decomposition/generation services.
6. DeepSeek client streams output.
7. Backend emits SSE chunks.
8. Frontend updates `frontend/src/store/streamStore.ts`.
9. Generated blogs are persisted in PostgreSQL via GORM models like `backend/internal/model/blog.go`.

## 5. What You Need To Master

For Go:

1. Go module/package rules: `go.mod`, package imports, `internal/`
2. Structs, methods, and interfaces
3. Error-first style: `if err != nil`
4. Context cancellation: `context.Context`
5. Goroutines and channels
6. GORM model tags and queries
7. Gin handlers and middleware
8. Testing with `go test`

For React/TypeScript:

1. Component state vs global Zustand state
2. Hooks: `useCallback`, custom hooks, side effects
3. TypeScript interfaces/types
4. Fetch and SSE request handling
5. Controlled forms and file upload
6. Markdown rendering and editor-preview sync
7. Tailwind utility styling

For this project specifically:

1. Read `README.md`.
2. Read `.trae/documents/InkWords_Architecture.md`.
3. Read `.trae/documents/InkWords_API.md`.
4. Trace `/api/v1/stream/generate` from frontend hook to backend handler to service to LLM client.
5. Trace `/api/v1/blogs` from sidebar/editor to backend domain repository.
6. Run backend tests with `cd backend && go test ./...`.
7. Run frontend tests with `cd frontend && npm test`.

## 6. Recommended Learning Order

Start with backend first because Spring Boot experience transfers well:

1. `backend/cmd/server/main.go`
2. `backend/internal/transport/http/v1/routes.go`
3. `backend/internal/domain/blog`
4. `backend/internal/domain/stream`
5. `backend/internal/service/decomposition_*`
6. `backend/internal/infra/llm/deepseek.go`

Then frontend:

1. `frontend/src/App.tsx`
2. `frontend/src/store/blogStore.ts`
3. `frontend/src/store/streamStore.ts`
4. `frontend/src/pages/Generator.tsx`
5. `frontend/src/hooks/generator/*`
6. `frontend/src/pages/Editor.tsx`

## 7. Practical Mastery Path

The fastest way to master this project is to trace one real workflow end to end:

```text
login -> parse source -> analyze outline -> generate series -> edit blog -> export
```

That path covers almost every important technology in the project:

- React pages and components
- Zustand state
- HTTP requests
- SSE streaming
- Gin routing
- Go service layering
- Goroutines and channels
- DeepSeek API integration
- PostgreSQL persistence
- Blog tree rendering
- Markdown editing/exporting

