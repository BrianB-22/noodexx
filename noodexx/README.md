# Noodexx

Noodexx is a privacy-first, local-first AI knowledge assistant that enables users to ingest, index, search, and interact with their documents through a modern conversational interface.

It combines Retrieval-Augmented Generation (RAG), modular AI provider support, automated ingestion, and an extensible plugin system to deliver a secure and customizable knowledge workspace.

---

## Overview

Noodexx allows users to:

- Index local documents
- Perform semantic search across their knowledge base
- Ask natural language questions grounded in their own data
- Run AI models locally or via cloud providers
- Extend functionality through plugins (“skills”)
- Operate entirely in privacy mode if required

By default, the system runs locally and keeps all indexed data under user control.

---

## Core Features

### Intelligent Document Indexing

Noodexx supports ingestion of:

- `.txt`
- `.md`
- `.pdf`
- `.html`
- Entire local folders (via file watching)

Documents are:

- Split into overlapping chunks
- Embedded into vector representations
- Stored in a local SQLite database
- Made searchable using cosine similarity

Optional auto-summarization provides high-level document insights.

---

### Retrieval-Augmented Conversational AI (RAG)

Users can:

- Ask questions in natural language
- Retrieve relevant document segments automatically
- Generate responses grounded in indexed content
- Receive streaming responses in real time

This ensures that answers are based on the user's own knowledge base rather than generic model output.

---

### Multi-Provider AI Support

Noodexx supports multiple LLM providers through a unified interface:

- **Ollama (local models)**
- **OpenAI**
- **Anthropic**

Providers are configurable through the application settings.

When **Privacy Mode** is enabled, only local models (Ollama) are allowed.

---

### Privacy Mode

Privacy Mode enforces strict local-only operation:

- Cloud providers disabled
- URL ingestion blocked
- Network-dependent skills disabled
- Localhost validation enforced
- No outbound API calls

Additional safeguards include:

- PII detection (SSN, credit cards, API keys, emails, phone numbers, private keys)
- File size limits
- Allowed/blocked extension filtering
- System directory protection
- Structured audit logging

---

### Extensible Skill System

Noodexx includes a plugin architecture that allows users to extend functionality.

Skills:

- Are defined via `skill.json`
- Run as controlled subprocesses
- Use structured JSON input/output
- Support manual, keyword, timer, or event triggers
- Enforce execution timeouts
- Respect privacy mode constraints

This enables custom automation and domain-specific integrations without modifying core application code.

---

### Automated Folder Monitoring

Noodexx can monitor configured directories and:

- Automatically ingest new files
- Re-index modified files
- Remove deleted files from the database
- Apply safety checks before processing

This supports continuous synchronization of knowledge sources.

---

### Modern Web Interface

The application provides a responsive browser-based UI with:

- Dashboard overview and system metrics
- Chat interface with session history
- Document library with tagging and filtering
- Drag-and-drop ingestion
- Settings management (provider, privacy, guardrails)
- Real-time notifications via WebSockets
- Partial page updates via HTMX

The server runs locally by default (127.0.0.1:8080).

---

## Architecture

Noodexx follows a modular package structure:

- `store` – SQLite database abstraction
- `llm` – LLM provider abstraction layer
- `rag` – Chunking, vector search, prompt building
- `ingest` – Document parsing and guardrails
- `api` – HTTP server and WebSocket hub
- `skills` – Plugin loader and executor
- `watcher` – Filesystem monitoring
- `config` – Configuration management
- `logging` – Structured logging

The backend is written in Go and uses only pure Go dependencies (no CGo).

---

## Technical Characteristics

- Backend: Go 1.21+
- Database: SQLite (pure Go driver)
- Frontend: HTMX + vanilla JavaScript
- WebSockets: Real-time notifications
- Default bind address: `127.0.0.1:8080`
- Binary size: ~15–20MB
- Memory baseline: ~50–100MB

---

## Intended Use Cases

Noodexx is designed for:

- Developers managing technical documentation
- Analysts working with private research data
- Knowledge workers requiring local AI augmentation
- Security-conscious individuals
- Organizations requiring data sovereignty

---

## Security Considerations

Recommended practices:

- Keep default localhost binding unless remote access is required
- Enable Privacy Mode when handling sensitive data
- Review third-party skills before enabling
- Use secure API keys for cloud providers
- Periodically review audit logs

---

## Summary

Noodexx provides a secure, extensible, and privacy-controlled AI knowledge assistant that operates locally while supporting optional cloud model integration.

It combines semantic search, conversational AI, automated ingestion, plugin extensibility, and real-time UI responsiveness into a cohesive, production-grade knowledge platform.
