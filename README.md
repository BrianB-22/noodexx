# Noodexx

Noodexx is a privacy-first, local AI knowledge assistant that enables users to ingest, index, search, and interact with their documents through a modern conversational interface.

It combines retrieval-augmented generation (RAG), modular AI provider support, and an extensible plugin system to deliver a secure and customizable knowledge workspace.

---

## Overview

Noodexx provides:

- Conversational access to personal knowledge
- Local document indexing with semantic search
- Configurable AI model support (local or cloud)
- Strict privacy controls
- Extensible automation via plugins (“skills”)
- Real-time, responsive web interface

The system runs locally by default, ensuring sensitive data remains under user control.

---

## Core Features

### 1. Intelligent Document Indexing

Supports ingestion of:

- `.txt`, `.md`
- `.pdf`
- `.html`
- Watched local folders (automatic ingestion)

Documents are:

- Chunked into overlapping segments
- Embedded into vector representations
- Stored in a local SQLite database
- Searchable via cosine similarity

Optional auto-summarization generates document overviews.

---

### 2. Retrieval-Augmented Conversational AI

Noodexx allows users to:

- Ask natural language questions
- Retrieve semantically relevant document segments
- Generate grounded responses using indexed content
- Stream answers in real time

All answers are contextualized against your data.

---

### 3. Multi-Provider AI Support

Supported providers:

- **Ollama (local)**
- **OpenAI**
- **Anthropic**

Privacy mode restricts the system to local-only providers.

---

### 4. Privacy-First Operation

Privacy mode enforces:

- Local-only model usage
- No outbound network requests
- URL ingestion disabled
- Network-dependent skills disabled
- Strict configuration validation

Security guardrails include:

- PII detection (SSN, credit card, API keys, private keys, email, phone)
- File size limits
- Extension whitelisting
- System directory protection
- Structured audit logging

---

### 5. Skill Plugin System

Noodexx supports user-defined plugins:

- Defined via `skill.json`
- Executed as controlled subprocesses
- JSON input/output contract
- Timeout enforcement
- Manual, keyword, timer, or event triggers
- Privacy mode compliance

Skills allow automation and integration without modifying core code.

---

### 6. Folder Watching

Optional folder monitoring:

- Detects new or modified files
- Automatically ingests allowed files
- Removes deleted files from index
- Applies guardrails before processing

---

### 7. Modern Web Interface

Includes:

- Dashboard with system metrics
- Chat interface with session history
- Document library with tagging/filtering
- Drag-and-drop file ingestion
- Settings management (provider, privacy, guardrails)
- Real-time notifications (WebSockets)
- HTMX-based partial updates

Runs locally at:

