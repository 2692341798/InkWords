# LLM Wiki Pattern: Schema and Architecture

This document describes the architectural requirements and YAML frontmatter schema for the Karpathy LLM Wiki pattern. 

## Directory Structure

The Obsidian Second Brain should adhere to the following directory layout:
- `.raw/` - The untouched original input files provided by the user (NEVER modify these).
- `wiki/` - The root folder for generated LLM Wiki markdown notes.
  - `wiki/index.md` - The global entry point and master index.
  - `wiki/hot.md` - Recent additions and current context (Hot cache).
  - `wiki/log.md` - Operation log of ingestions and updates.
  - `wiki/sources/` - Cards describing the raw materials ingested.
  - `wiki/entities/` - Cards for specific people, organizations, tools, projects, etc.
  - `wiki/concepts/` - Cards for abstract ideas, theories, patterns, frameworks, etc.
  - `wiki/domains/` - High-level domain index files (e.g., `wiki/domains/ai/_index.md`).

## YAML Frontmatter Schema

Every generated markdown file MUST contain the following YAML frontmatter exactly. 

```yaml
---
type: <source|entity|concept|domain|meta>
title: "The clear title of the page"
created: YYYY-MM-DD
updated: YYYY-MM-DD
tags:
  - "#domain/domain_name"
status: <seed|developing|mature|evergreen>
related:
  - "[[Related Page]]"
---
```

### Frontmatter Fields

- **type**: The category of the card. Must be one of `source`, `entity`, `concept`, `domain`, or `meta`.
- **title**: The human-readable title.
- **created**: Date of card creation (YYYY-MM-DD).
- **updated**: Date of the last modification (YYYY-MM-DD).
- **tags**: Domain tags, e.g., `#domain/ai`, `#domain/productivity`. Can include specific sub-tags.
- **status**: The maturity of the card's knowledge:
  - `seed`: Just created, basic context extracted.
  - `developing`: Contains details from multiple sources.
  - `mature`: Well-fleshed out and highly linked.
  - `evergreen`: Core, highly refined knowledge.
- **related**: A list of Obsidian wikilinks to conceptually related pages.

## Content Guidelines

- **Bidirectional Linking**: Make sure to link to other pages in the body text using `[[Page Name]]`.
- **Source Tracing**: Every entity and concept card MUST explain where the information came from, referencing the source card (`[[Source Name]]`).
- **Immutability of `.raw/`**: You must NEVER modify the user's raw files. You only create and update cards within the `wiki/` directory.
