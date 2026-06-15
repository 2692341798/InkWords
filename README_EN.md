# InkWords Trainer

Turn raw materials into knowledge, then turn knowledge into capability.

[中文版 README](./README.md)

## Overview

InkWords Trainer is a local knowledge workbench focused on knowledge ingestion, structured learning, review, and optional content output. It is no longer just a "blog generator". The product is designed around a full learning loop:

`Source ingestion -> Knowledge structuring -> Obsidian persistence -> Knowledge review -> Optional export to blog / PDF / Obsidian`

The project is Chinese-first in its product UI, but the codebase and architecture are organized for a full-stack engineering workflow.

## What It Does

- Import Git repositories, technical docs, PDF, DOCX, Markdown, TXT, and ZIP courseware packages
- Convert source materials into structured knowledge content suitable for long-term maintenance
- Persist content into an Obsidian-style knowledge graph using the LLM Wiki pattern
- Run a dedicated knowledge review workflow with recall prompts and structured follow-up questions
- Generate blog series, continue drafts, polish content, export Markdown / ZIP / PDF, and send content to Obsidian

## Core Capabilities

- **Ingestion**: Git repository scanning, local file parsing, ZIP extraction, whitelist filtering, and aggregated analysis
- **Knowledge Persistence**: `sources/`, `concepts/`, `entities/`, index pages, and hot cache pages aligned with the Obsidian LLM Wiki pattern
- **Knowledge Review**: Separate review workspace with random pick, manual selection, light recall, and detailed QA
- **Content Generation**: Single article generation, series generation, continuation, polishing, and automatic series intro generation
- **Task Center**: Generation, parsing, and export are gradually unified under `job_tasks + RabbitMQ + SSE`
- **Async Export**: Markdown / ZIP export, async PDF export, and export to Obsidian Vault
- **Prompt Profile Locking**: File Analyze detects document type and locks a `prompt_profile`
- **Quality Pipeline**: Series chapter generation exposes `understanding -> drafting -> reviewing -> revising -> final output`

## Architecture

This repository is a frontend/backend monorepo:

- `frontend/`: React + Vite + Tailwind CSS + shadcn/ui + Zustand
- `backend/`: Go + Gin + GORM + PostgreSQL + RabbitMQ + Redis
- `docker-compose.yml`: the single container orchestration entrypoint

### Production Shape

The standard runtime shape is "single frontend entrypoint + multiple backend services":

- `frontend`: Nginx static site and API gateway
- `core-api`: core business API, task creation/query, SSE replay, user and blog ownership writes
- `llm-stream`: streaming generation execution and generation worker
- `parser-service`: file / ZIP parsing and parse worker
- `export-service`: PDF export and export worker
- `review-service`: knowledge review service
- `db`: PostgreSQL
- `redis`: cache and runtime state helpers
- `rabbitmq`: task queue
- `obsidian-bridge`: bridge service for reaching the host Obsidian Local REST API from containers

### Public Entry

- The default public entry is always `http://localhost`
- Page access should go through the frontend gateway, not directly to backend ports
- If host port `:80` is occupied, you can temporarily switch to `http://localhost:8088` with `FRONTEND_PORT=8088`

### Gateway Routing

Frontend Nginx routes requests by path:

- `/api/v1/stream/*` -> `llm-stream`
- `/api/v1/project/parse` -> `parser-service`
- `/api/v1/review/*` -> `review-service`
- `/api/v1/blogs/:id/export*` -> `export-service`
- all other `/api/*` -> `core-api`

## Key Workflows

### 1. Source Ingestion

- Git repositories: scan structure, identify key modules, and analyze selected areas
- File upload: supports PDF / DOCX / Markdown / TXT
- ZIP courseware: secure extraction, whitelist filtering, text aggregation, and parse summary
- Long text protection: very large content goes through Map-Reduce chunked analysis

### 2. Scenario and Prompt Control

- Supported scenarios: `ebook interpretation`, `open-book review`, `beginner walkthrough`
- File Analyze additionally locks a `prompt_profile`
- After outline generation, scenario and prompt type become read-only labels in the UI

### 3. Task Center

- Generation tasks: create a generation task first, then subscribe to `/api/v1/tasks/:id/stream`
- Parse tasks: ZIP files and normal files larger than `50MB` default to task-based parsing
- Export tasks: series PDF export defaults to export tasks with controlled download

### 4. Knowledge Review

- Separate entry instead of mixing review into the generator workflow
- Supports random pick and manual note selection
- Supports `light_recall` and `detailed_qa`
- Shows original content preview before the user starts answering
- Returns a structured result such as current goal, what was covered, what was missed, and next-step suggestions

### 5. Output

- Blog series generation
- Draft continuation and polishing
- Markdown / ZIP export
- Async PDF export
- Export to Obsidian Vault

## Quick Start

### 1. Prepare Environment Variables

Recommended prerequisites:

- Docker
- Docker Compose
- DeepSeek API key
- Optional Obsidian Local REST API setup

Copy the environment template:

```bash
cp backend/.env.example backend/.env
```

At minimum, review these variables:

- `DEEPSEEK_API_KEY`
- `JWT_SECRET`
- `OBSIDIAN_REST_API_KEY`
- `OBSIDIAN_VAULT_PATH`

Useful defaults are already provided in `backend/.env.example`, including:

- `POSTGRES_USER`
- `POSTGRES_PASSWORD`
- `POSTGRES_DB`
- `RABBITMQ_URL`
- `RABBITMQ_EXCHANGE`
- `RABBITMQ_GENERATION_QUEUE`
- `RABBITMQ_PARSE_QUEUE`
- `RABBITMQ_EXPORT_QUEUE`

### 2. Start the Full Stack

```bash
docker compose --env-file backend/.env up -d --build
```

To fully restart:

```bash
docker compose --env-file backend/.env down && docker compose --env-file backend/.env up -d --build
```

To scale only the streaming generator service:

```bash
docker compose --env-file backend/.env up -d --build --scale llm-stream=3
```

### 3. Verify Runtime Health

```bash
docker compose --env-file backend/.env ps
curl -I http://localhost
curl --fail http://localhost/api/v1/ping
```

Expected outcome:

- `frontend`, `core-api`, `llm-stream`, `parser-service`, `export-service`, and `review-service` become healthy
- `http://localhost` is reachable
- `/api/v1/ping` returns success

### 4. Resolve Port Conflicts

If host port `:80` is already in use:

```bash
FRONTEND_PORT=8088 docker compose --env-file backend/.env up -d --build frontend
```

Then access the app at `http://localhost:8088`.

## Local Development

### Backend

The local aggregate entrypoint is still kept for local development and integration debugging:

```bash
cd backend
cp .env.example .env
go mod tidy
go run ./cmd/server/main.go
```

Notes:

- The local aggregate server runs at `http://localhost:8080` by default
- `cmd/server` is mainly for local development and integration debugging
- Docker production mode does not use this aggregate entrypoint by default

### Frontend

```bash
cd frontend
npm install
npm run dev
```

The frontend dev server runs at `http://localhost:5173` by default.

### Common Frontend Commands

```bash
cd frontend
npm run dev
npm run build
npm run lint
npm run test
```

## Repository Layout

```text
InkWords/
├── frontend/                    # React frontend and Nginx build source
├── backend/                     # Go backend and service implementations
│   ├── cmd/server/              # Local aggregate debug entrypoint
│   ├── internal/                # Shared domain, infra, and transitional layers
│   ├── services/
│   │   ├── core-api/            # Core API service
│   │   ├── llm-stream/          # Streaming generation service
│   │   ├── parser-service/      # Parsing service
│   │   ├── export-service/      # Export service
│   │   └── review-service/      # Review service
│   ├── db/                      # Database init scripts
│   └── scripts/                 # Utility scripts
├── docs/runbooks/               # Runbooks and troubleshooting docs
├── .trae/documents/             # PRD, architecture, API, database, and project docs
├── docker-compose.yml           # Multi-service orchestration entrypoint
└── README.md                    # Chinese README
```

## Tech Stack

### Frontend

- React 19
- Vite 8
- Tailwind CSS 4
- shadcn/ui
- Zustand
- `@microsoft/fetch-event-source`

### Backend

- Go 1.25
- Gin
- GORM
- PostgreSQL 14
- RabbitMQ
- Redis
- DeepSeek API

### Infrastructure

- Docker
- Docker Compose
- Nginx
- Obsidian Local REST API

## Project Status

The project is already running in a real multi-service shape, while core boundaries are still being tightened:

- Completed: service ownership and deployment shape for `parser-service`, `review-service`, and `export-service`
- Landed: single Nginx gateway entry, task-based generation / parsing / export foundation
- In progress: deeper `core-api / llm-stream` split, ownership cleanup of business writes, and removal of legacy shared boundaries

## Runbooks and Docs

Recommended operational references:

- [Microservices Smoke Check](./docs/runbooks/microservices-smoke-check.md)
- [Core Task Boundary](./docs/runbooks/core-blog-task-boundary.md)
- [Review Database Migration](./docs/runbooks/review-db-migration.md)
- [Service Image Boundaries](./docs/runbooks/service-image-boundaries.md)

Project baseline docs that should stay in sync with code:

- [PRD](./.trae/documents/InkWords_PRD.md)
- [Architecture](./.trae/documents/InkWords_Architecture.md)
- [Database](./.trae/documents/InkWords_Database.md)
- [API](./.trae/documents/InkWords_API.md)
- [Development Plan and Log](./.trae/documents/InkWords_Development_Plan_and_Log.md)
- [Conversation Log](./.trae/documents/InkWords_Conversation_Log.md)

## Development Notes

- If you change business logic, APIs, or table structure, update the related project docs as well
- Docker Compose is the default verification path for the full runtime shape
- The default public entry remains `http://localhost`
- User-facing frontend copy is Chinese by design
- Uploaded and parsed source files follow the "read and dispose" rule and should not be persisted as raw source artifacts

## Note

You can still use InkWords as a "blog generation platform" if that is your main goal. The long-term product direction, however, is a knowledge training loop centered on source ingestion, knowledge review, and optional content output.
