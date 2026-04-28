---
name: llm-wiki-ingest
description: |
  Ingests new documents, articles, or notes into an Obsidian second brain using the Karpathy LLM Wiki pattern. 
  Use this skill when the user asks to "ingest a document", "add this article to my wiki", "extract entities from this text", or "save this to my second brain".
  It handles extracting entities and concepts, creating structured Markdown cards with YAML frontmatter, and linking them properly into the wiki structure.
---

# LLM Wiki Ingest Workflow

This skill automates the process of ingesting raw materials (web pages, PDFs, documents) into a structured Obsidian Second Brain using the Karpathy LLM Wiki pattern.

## Core Mechanism

When the user provides a source text or document to ingest, you MUST follow these steps:

1. **Analyze the Source**: Read the provided content to understand the domain and key information.
2. **Create Source Card**: Create a metadata card for the raw material in the `wiki/sources/` directory.
3. **Extract Entities & Concepts**: Identify key entities (people, organizations, tools) and concepts (ideas, frameworks, theories) mentioned in the text.
4. **Generate/Update Cards**: For each extracted entity and concept:
   - Create a new card in `wiki/entities/` or `wiki/concepts/` respectively.
   - If a card already exists, update it with the new context.
5. **Sew the Knowledge Network**: Ensure bidirectional links (`[[Page Name]]`) are used extensively between the source card, entities, and concepts.
6. **Update Global Index**: Update the main `wiki/index.md` or domain-specific indexes.
7. **Log the Operation**: Append an entry to `wiki/log.md` detailing the ingestion.

## Detailed Guidelines

For detailed YAML frontmatter templates and architectural requirements of the LLM Wiki pattern, please read [references/schema_and_architecture.md](references/schema_and_architecture.md).

## Step-by-Step Instructions

1. **Read Input**: Ask the user for the raw text or the path to the document to be ingested if not already provided.
2. **Identify Domain**: Determine the core domain (e.g., `#domain/ai`, `#domain/productivity`).
3. **Create Source Card**: Generate the Markdown file for the source.
4. **Extract Knowledge**: Identify 3-5 core entities/concepts. Do not over-extract. Only extract items that have standalone value in a knowledge base.
5. **Create/Update Knowledge Cards**: For each entity/concept, generate its Markdown card following the template. Include a "Context from [[Source Name]]" section where you link back to the source card and explain how it relates.
6. **Update Indexes**: Update `wiki/index.md` and `wiki/hot.md` to reflect the newly ingested knowledge.
7. **Update Log**: Append a timestamped log entry to `wiki/log.md`.
