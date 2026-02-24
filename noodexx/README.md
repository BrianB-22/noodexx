![Noodexx Logo](web/static/logo-large.png)

# Noodexx

**A privacy-first, local-first AI knowledge assistant with RAG, modular architecture, and extensible plugin system.**

Noodexx enables you to ingest, index, search, and interact with your documents through a modern conversational interface. Built with Go, it combines Retrieval-Augmented Generation (RAG), multi-provider LLM support, automated folder watching, and a powerful skill system—all while keeping your data under your control.

---

## Quick Start

### Prerequisites

- Go 1.21 or later
- Ollama (for local models) or API keys for OpenAI/Anthropic

### Installation

```bash
# Clone the repository
git clone https://github.com/yourusername/noodexx.git
cd noodexx

# Build the application
go build -o noodexx

# Run Noodexx
./noodexx
```

The server will start on `http://127.0.0.1:8080` by default.

### First Steps

1. Open your browser to `http://127.0.0.1:8080`
2. Navigate to **Settings** to configure your LLM provider
3. Go to **Library** to ingest your first document (drag & drop supported)
4. Start chatting in the **Chat** interface

---

## Phase 2 Features Overview

### Modern Web Interface

- **Dashboard**: System overview with metrics, activity feed, and quick actions
- **Chat Interface**: Conversational AI with persistent session history and markdown rendering
- **Library**: Visual card grid with drag-and-drop upload, tagging, and filtering
- **Settings**: Configure providers, privacy mode, guardrails, and skills
- **Real-time Updates**: WebSocket notifications for background operations
- **Command Palette**: Keyboard-driven navigation (⌘K / Ctrl+K)

### Modular Architecture

Phase 2 refactors the monolithic codebase into focused packages:

- `internal/store` - SQLite database abstraction
- `internal/llm` - Multi-provider LLM interface (Ollama, OpenAI, Anthropic)
- `internal/rag` - Chunking, vector search, and prompt building
- `internal/ingest` - Document parsing with PII detection and guardrails
- `internal/api` - HTTP handlers and WebSocket hub
- `internal/skills` - Plugin system for extensibility
- `internal/watcher` - Automated folder monitoring
- `internal/config` - Configuration management
- `internal/logging` - Structured logging system

### Privacy Mode

Enable **Privacy Mode** to ensure all operations stay local:

- Restricts LLM provider to Ollama only
- Blocks URL ingestion (no external network requests)
- Disables network-dependent skills
- Validates localhost-only endpoints
- Displays privacy indicator in UI

### Extensible Skill System

Create custom skills (scripts, binaries, programs) to extend Noodexx:

- Define skills with `skill.json` metadata
- Support for manual, timer, keyword, and event triggers
- JSON-based stdin/stdout communication
- Configurable timeouts and settings
- Privacy mode enforcement
- Example skills included (weather, summarize-url, daily-digest)

### Automated Folder Watching

Monitor directories for automatic ingestion:

- Auto-ingest new files
- Re-index modified files
- Remove deleted files from database
- Configurable file type filters and size limits
- Concurrent processing with rate limiting

### Enhanced Security

- **PII Detection**: Warns before ingesting sensitive data (SSNs, credit cards, API keys, private keys)
- **Ingestion Guardrails**: File size limits, extension allowlists, sensitive filename detection
- **Audit Logging**: Complete record of all operations (ingestions, queries, deletions, config changes)
- **Localhost Binding**: Defaults to 127.0.0.1 (not exposed to network)
- **API Key Encryption**: Secure storage of cloud provider credentials

---

## Configuration Guide

Noodexx uses a `config.json` file for configuration. On first run, a default configuration is created automatically.

### Configuration File Location

The `config.json` file is located in the application directory (same directory as the `noodexx` binary).

### Default Configuration

```json
{
  "provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text",
    "ollama_chat_model": "llama3.2",
    "openai_key": "",
    "openai_embed_model": "text-embedding-3-small",
    "openai_chat_model": "gpt-4",
    "anthropic_key": "",
    "anthropic_embed_model": "",
    "anthropic_chat_model": "claude-3-sonnet-20240229"
  },
  "privacy": {
    "enabled": true
  },
  "folders": [],
  "logging": {
    "level": "info",
    "file": "",
    "max_size_mb": 100,
    "max_backups": 3
  },
  "guardrails": {
    "max_file_size_mb": 10,
    "allowed_extensions": [".txt", ".md", ".pdf", ".html"],
    "max_concurrent": 3,
    "pii_detection": "normal",
    "auto_summarize": true
  },
  "server": {
    "port": 8080,
    "bind_address": "127.0.0.1"
  }
}
```

### Configuration Examples

#### Local-Only Setup (Privacy Mode)

```json
{
  "provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text",
    "ollama_chat_model": "llama3.2"
  },
  "privacy": {
    "enabled": true
  },
  "folders": [
    "/Users/yourname/Documents/notes",
    "/Users/yourname/Documents/research"
  ],
  "server": {
    "port": 8080,
    "bind_address": "127.0.0.1"
  }
}
```

#### OpenAI Cloud Setup

```json
{
  "provider": {
    "type": "openai",
    "openai_key": "sk-your-api-key-here",
    "openai_embed_model": "text-embedding-3-small",
    "openai_chat_model": "gpt-4"
  },
  "privacy": {
    "enabled": false
  },
  "guardrails": {
    "max_file_size_mb": 20,
    "pii_detection": "strict",
    "auto_summarize": true
  }
}
```

#### Anthropic Cloud Setup

```json
{
  "provider": {
    "type": "anthropic",
    "anthropic_key": "sk-ant-your-api-key-here",
    "anthropic_chat_model": "claude-3-opus-20240229"
  },
  "privacy": {
    "enabled": false
  }
}
```

#### Development Setup with Debug Logging

```json
{
  "provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text",
    "ollama_chat_model": "llama3.2"
  },
  "logging": {
    "level": "debug",
    "file": "noodexx.log",
    "max_size_mb": 50,
    "max_backups": 5
  },
  "guardrails": {
    "pii_detection": "off",
    "auto_summarize": false
  }
}
```

### Environment Variable Overrides

All configuration values can be overridden with environment variables:

```bash
# Provider configuration
export NOODEXX_PROVIDER=openai
export NOODEXX_OPENAI_KEY=sk-your-key
export NOODEXX_OPENAI_CHAT_MODEL=gpt-4

# Privacy mode
export NOODEXX_PRIVACY_MODE=true

# Server configuration
export NOODEXX_SERVER_PORT=3000
export NOODEXX_SERVER_BIND_ADDRESS=0.0.0.0

# Logging
export NOODEXX_LOG_LEVEL=debug
export NOODEXX_LOG_FILE=/var/log/noodexx.log

# Run Noodexx
./noodexx
```

---

## Privacy Mode Explained

Privacy Mode is a system-wide setting that enforces strict local-only operation. When enabled:

### What Privacy Mode Does

1. **Restricts LLM Provider**: Only Ollama (local models) can be used
2. **Blocks External Network**: URL ingestion is disabled
3. **Validates Endpoints**: Ensures Ollama endpoint is localhost/127.0.0.1
4. **Disables Network Skills**: Skills requiring network access won't load
5. **UI Indicators**: Shows "🔒 Privacy Mode Active" badge in settings and dashboard

### When to Use Privacy Mode

- Working with sensitive or confidential documents
- Compliance requirements (HIPAA, GDPR, etc.)
- Air-gapped or offline environments
- Personal preference for data sovereignty
- Testing without cloud API costs

### Enabling Privacy Mode

**Via UI:**
1. Navigate to **Settings**
2. Toggle **Privacy Mode** switch at the top
3. If using a cloud provider, you'll be prompted to switch to Ollama

**Via Configuration:**
```json
{
  "privacy": {
    "enabled": true
  },
  "provider": {
    "type": "ollama"
  }
}
```

**Via Environment Variable:**
```bash
export NOODEXX_PRIVACY_MODE=true
./noodexx
```

### Privacy Mode Guarantees

When Privacy Mode is enabled, Noodexx guarantees:

- ✅ All embeddings generated locally via Ollama
- ✅ All chat completions processed locally
- ✅ No data sent to external APIs
- ✅ No URL fetching or external HTTP requests
- ✅ All data stored in local SQLite database
- ✅ Skills with `requires_network: true` are blocked

---

## Skill Development Guide

Skills are custom executables that extend Noodexx functionality. A skill can be a shell script, Python script, compiled binary, or any executable program.

### Skill Structure

```
skills/
└── my-skill/
    ├── skill.json       # Metadata and configuration
    └── run.sh           # Executable (or run.py, run, etc.)
```

### Creating a Skill

#### 1. Create Skill Directory

```bash
mkdir -p skills/my-skill
cd skills/my-skill
```

#### 2. Create skill.json

```json
{
  "name": "my-skill",
  "version": "1.0.0",
  "description": "A custom skill that does something useful",
  "executable": "run.sh",
  "triggers": [
    {
      "type": "manual",
      "parameters": {}
    }
  ],
  "settings_schema": {
    "type": "object",
    "properties": {
      "api_key": {
        "type": "string",
        "description": "API key for external service"
      },
      "timeout": {
        "type": "integer",
        "description": "Timeout in seconds",
        "default": 30
      }
    }
  },
  "timeout": 30,
  "requires_network": false
}
```

#### 3. Create Executable

**Shell Script Example (run.sh):**

```bash
#!/bin/bash

# Read JSON input from stdin
INPUT=$(cat)

# Parse input (using jq for JSON parsing)
QUERY=$(echo "$INPUT" | jq -r '.query')

# Do something useful
RESULT="Processed: $QUERY"

# Return JSON output
cat <<EOF
{
  "result": "$RESULT",
  "error": "",
  "metadata": {
    "processed_at": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  }
}
EOF
```

**Python Script Example (run.py):**

```python
#!/usr/bin/env python3
import json
import sys
from datetime import datetime

# Read input from stdin
input_data = json.load(sys.stdin)

query = input_data.get('query', '')
context = input_data.get('context', {})
settings = input_data.get('settings', {})

# Do something useful
result = f"Processed: {query}"

# Return output
output = {
    "result": result,
    "error": "",
    "metadata": {
        "processed_at": datetime.utcnow().isoformat() + "Z"
    }
}

json.dump(output, sys.stdout)
```

#### 4. Make Executable

```bash
chmod +x run.sh  # or run.py
```

### Skill Input Format

Skills receive JSON input on stdin:

```json
{
  "query": "user query or trigger data",
  "context": {
    "session_id": "abc123",
    "user": "username"
  },
  "settings": {
    "api_key": "configured-value",
    "timeout": 30
  }
}
```

### Skill Output Format

Skills must return JSON on stdout:

```json
{
  "result": "The skill's output text",
  "error": "",
  "metadata": {
    "custom_field": "value"
  }
}
```

If an error occurs:

```json
{
  "result": "",
  "error": "Error message describing what went wrong",
  "metadata": {}
}
```

### Trigger Types

#### Manual Trigger

Skill appears in command palette and can be invoked by name:

```json
{
  "triggers": [
    {
      "type": "manual",
      "parameters": {}
    }
  ]
}
```

#### Keyword Trigger

Skill executes when chat message contains specified keywords:

```json
{
  "triggers": [
    {
      "type": "keyword",
      "parameters": {
        "keywords": ["weather", "forecast"]
      }
    }
  ]
}
```

#### Timer Trigger

Skill executes on a schedule (cron-like):

```json
{
  "triggers": [
    {
      "type": "timer",
      "parameters": {
        "schedule": "0 9 * * *"
      }
    }
  ]
}
```

#### Event Trigger

Skill executes when system events occur:

```json
{
  "triggers": [
    {
      "type": "event",
      "parameters": {
        "events": ["ingest_complete", "alert_created"]
      }
    }
  ]
}
```

### Environment Variables

Skills receive these environment variables:

- `NOODEXX_PRIVACY_MODE` - "true" if privacy mode is enabled
- `NOODEXX_SKILL_NAME` - The skill's name
- `NOODEXX_SKILL_VERSION` - The skill's version
- Custom settings from `skill.json` as `NOODEXX_SETTING_<KEY>`

### Example Skills

Noodexx includes example skills in `skills/examples/`:

- **weather** - Fetches weather forecast from wttr.in
- **summarize-url** - Fetches and summarizes a URL
- **daily-digest** - Generates a daily summary (timer-based)

Study these examples to learn skill development patterns.

### Best Practices

1. **Validate Input**: Always check that required fields are present
2. **Handle Errors**: Return descriptive error messages in the `error` field
3. **Respect Timeouts**: Complete execution within the configured timeout
4. **Privacy Mode**: Check `NOODEXX_PRIVACY_MODE` and avoid network calls when enabled
5. **Logging**: Log to stderr (stdout is reserved for JSON output)
6. **Testing**: Test skills independently before integrating with Noodexx

---

## API Documentation

Noodexx provides HTTP endpoints for all operations. The API uses JSON for data exchange and supports both full page renders and HTMX partial updates.

### Base URL

```
http://127.0.0.1:8080
```

### Page Endpoints

#### GET /

**Dashboard page** - System overview with metrics and activity feed

**Response:** HTML page

---

#### GET /chat

**Chat interface** - Conversational AI with session history

**Response:** HTML page

---

#### GET /library

**Library page** - Document management with card grid view

**Response:** HTML page

---

#### GET /settings

**Settings page** - Configure providers, privacy, and guardrails

**Response:** HTML page

---

### API Endpoints

#### POST /api/ask

**Send a chat message and receive streaming response**

**Request Body:**
```json
{
  "query": "What is the capital of France?",
  "session_id": "abc123"
}
```

**Response:** Server-Sent Events (SSE) stream with markdown-rendered HTML chunks

---

#### POST /api/ingest/text

**Ingest plain text**

**Request Body:**
```json
{
  "source": "my-note.txt",
  "text": "This is the content to ingest",
  "tags": ["notes", "personal"]
}
```

**Response:**
```json
{
  "success": true,
  "message": "Ingested 3 chunks from my-note.txt"
}
```

---

#### POST /api/ingest/url

**Ingest content from a URL** (disabled in privacy mode)

**Request Body:**
```json
{
  "url": "https://example.com/article",
  "tags": ["articles", "research"]
}
```

**Response:**
```json
{
  "success": true,
  "message": "Ingested 5 chunks from https://example.com/article"
}
```

---

#### POST /api/ingest/file

**Upload and ingest a file**

**Request:** multipart/form-data
- `file` - The file to upload
- `tags` - Comma-separated tags (optional)

**Response:**
```json
{
  "success": true,
  "message": "Ingested 8 chunks from document.pdf"
}
```

---

#### GET /api/sessions

**List all chat sessions**

**Response:**
```json
{
  "sessions": [
    {
      "id": "abc123",
      "last_message_at": "2024-01-15T10:30:00Z",
      "message_count": 12
    }
  ]
}
```

---

#### GET /api/session/{session_id}

**Get message history for a session**

**Response:**
```json
{
  "messages": [
    {
      "id": 1,
      "session_id": "abc123",
      "role": "user",
      "content": "What is RAG?",
      "created_at": "2024-01-15T10:30:00Z"
    },
    {
      "id": 2,
      "session_id": "abc123",
      "role": "assistant",
      "content": "RAG stands for Retrieval-Augmented Generation...",
      "created_at": "2024-01-15T10:30:05Z"
    }
  ]
}
```

---

#### DELETE /api/delete

**Delete a document source**

**Request Body:**
```json
{
  "source": "my-note.txt"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Deleted my-note.txt"
}
```

---

#### GET /api/config

**Get current configuration**

**Response:**
```json
{
  "provider": "ollama",
  "privacy_mode": true,
  "model": "llama3.2"
}
```

---

#### POST /api/config

**Update configuration**

**Request Body:**
```json
{
  "provider": {
    "type": "openai",
    "openai_key": "sk-...",
    "openai_chat_model": "gpt-4"
  },
  "privacy": {
    "enabled": false
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Configuration updated"
}
```

---

#### POST /api/test-connection

**Test LLM provider connection**

**Request Body:**
```json
{
  "provider": {
    "type": "openai",
    "openai_key": "sk-...",
    "openai_embed_model": "text-embedding-3-small"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Connection successful"
}
```

---

#### GET /api/activity

**Get recent activity feed**

**Response:**
```json
{
  "activities": [
    {
      "id": 1,
      "timestamp": "2024-01-15T10:30:00Z",
      "operation_type": "ingest",
      "details": "Ingested document.pdf",
      "user_context": ""
    }
  ]
}
```

---

#### GET /api/skills

**List all loaded skills**

**Response:**
```json
{
  "skills": [
    {
      "name": "weather",
      "version": "1.0.0",
      "description": "Fetch weather forecast",
      "triggers": [{"type": "manual"}],
      "requires_network": true
    }
  ]
}
```

---

#### POST /api/skills/run

**Execute a skill**

**Request Body:**
```json
{
  "skill_name": "weather",
  "input": {
    "query": "Seattle",
    "context": {},
    "settings": {}
  }
}
```

**Response:**
```json
{
  "result": "Weather in Seattle: Partly cloudy, 15°C",
  "error": "",
  "metadata": {}
}
```

---

### WebSocket Endpoint

#### WS /ws

**Real-time notifications**

Connect to receive real-time updates:

```javascript
const ws = new WebSocket('ws://127.0.0.1:8080/ws');

ws.onmessage = (event) => {
  const data = JSON.parse(event.data);
  console.log('Update:', data);
};
```

**Message Format:**
```json
{
  "type": "ingest_complete",
  "data": {
    "source": "document.pdf",
    "chunks": 8
  }
}
```

---

## Troubleshooting

### Common Issues

#### Ollama Connection Failed

**Symptom:** Error message "failed to connect to Ollama"

**Solutions:**
1. Ensure Ollama is running: `ollama serve`
2. Check Ollama endpoint in config.json: `http://localhost:11434`
3. Verify Ollama is accessible: `curl http://localhost:11434/api/tags`
4. Check firewall settings

---

#### Models Not Found

**Symptom:** Error message "model not found"

**Solutions:**
1. Pull the required models:
   ```bash
   ollama pull nomic-embed-text
   ollama pull llama3.2
   ```
2. Verify models are installed: `ollama list`
3. Update model names in config.json to match installed models

---

#### Privacy Mode Blocks Cloud Providers

**Symptom:** Cannot select OpenAI/Anthropic in settings

**Solutions:**
1. Disable privacy mode in settings UI
2. Or update config.json:
   ```json
   {
     "privacy": {
       "enabled": false
     }
   }
   ```
3. Restart Noodexx

---

#### File Upload Fails

**Symptom:** "File size exceeds limit" or "Extension not allowed"

**Solutions:**
1. Check file size (default limit: 10MB)
2. Verify file extension is allowed (.txt, .md, .pdf, .html)
3. Adjust guardrails in config.json:
   ```json
   {
     "guardrails": {
       "max_file_size_mb": 20,
       "allowed_extensions": [".txt", ".md", ".pdf", ".html", ".docx"]
     }
   }
   ```

---

#### PII Detection Blocks Ingestion

**Symptom:** "PII detected - ingestion blocked"

**Solutions:**
1. Review the file for sensitive data (SSNs, credit cards, API keys)
2. Remove sensitive data before ingesting
3. Or adjust PII detection level:
   ```json
   {
     "guardrails": {
       "pii_detection": "off"
     }
   }
   ```
   **Warning:** Only disable PII detection if you're certain the data is safe

---

#### Port Already in Use

**Symptom:** "bind: address already in use"

**Solutions:**
1. Change the port in config.json:
   ```json
   {
     "server": {
       "port": 3000
     }
   }
   ```
2. Or use environment variable:
   ```bash
   export NOODEXX_SERVER_PORT=3000
   ./noodexx
   ```
3. Or stop the process using port 8080:
   ```bash
   lsof -ti:8080 | xargs kill
   ```

---

#### Templates Not Found

**Symptom:** "failed to load templates"

**Solutions:**
1. Ensure you're running Noodexx from the correct directory
2. Verify `web/templates/` directory exists
3. Check file permissions on template files

---

#### Database Locked

**Symptom:** "database is locked"

**Solutions:**
1. Ensure only one Noodexx instance is running
2. Check for stale lock files
3. Restart Noodexx
4. If persistent, backup and recreate database:
   ```bash
   cp noodexx.db noodexx.db.backup
   rm noodexx.db
   ./noodexx  # Will create new database
   ```

---

#### Skill Execution Timeout

**Symptom:** "skill execution timed out"

**Solutions:**
1. Increase timeout in skill.json:
   ```json
   {
     "timeout": 60
   }
   ```
2. Optimize skill code for faster execution
3. Check if skill is waiting for external resources

---

#### WebSocket Connection Failed

**Symptom:** Real-time updates not working

**Solutions:**
1. Check browser console for WebSocket errors
2. Verify server is running and accessible
3. Check if proxy/firewall blocks WebSocket connections
4. Try accessing directly without reverse proxy

---

### Debug Mode

Enable debug logging for detailed troubleshooting:

**Via config.json:**
```json
{
  "logging": {
    "level": "debug",
    "file": "noodexx.log"
  }
}
```

**Via environment variable:**
```bash
export NOODEXX_LOG_LEVEL=debug
./noodexx
```

Debug logs include:
- All HTTP requests and responses
- Database queries
- LLM API calls
- Skill executions
- File system events

---

### Getting Help

If you encounter issues not covered here:

1. Check the logs (especially with debug mode enabled)
2. Review the audit log in settings for operation history
3. Search existing GitHub issues
4. Open a new issue with:
   - Noodexx version
   - Operating system
   - Configuration (redact API keys)
   - Error messages
   - Steps to reproduce

---

## Architecture Overview

### System Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                         Browser                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐   │
│  │Dashboard │  │   Chat   │  │ Library  │  │ Settings │   │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘   │
│       │             │              │              │          │
│       └─────────────┴──────────────┴──────────────┘          │
│                     │                                         │
│              HTMX + WebSocket                                │
└─────────────────────┼─────────────────────────────────────────┘
                      │
┌─────────────────────┼─────────────────────────────────────────┐
│                     ▼                                         │
│              HTTP Server (main.go)                           │
│                     │                                         │
│       ┌─────────────┴─────────────┐                         │
│       │                           │                         │
│       ▼                           ▼                         │
│  ┌─────────┐              ┌──────────────┐                 │
│  │   API   │              │  WebSocket   │                 │
│  │ Package │              │     Hub      │                 │
│  └────┬────┘              └──────────────┘                 │
│       │                                                     │
│       ├──────┬──────┬──────┬──────┬──────┐              │
│       ▼      ▼      ▼      ▼      ▼      ▼              │
│   ┌──────┬──────┬──────┬──────┬──────┬──────┐          │
│   │Store │ LLM  │ RAG  │Ingest│Skills│Config│          │
│   └──┬───┴──────┴──────┴──────┴──────┴──────┘          │
│      │                                                   │
│      ▼                                                   │
│  ┌────────┐              ┌──────────────┐              │
│  │ SQLite │              │ Folder Watch │              │
│  │   DB   │              │   (fsnotify) │              │
│  └────────┘              └──────────────┘              │
│                                                         │
│                     ┌──────────────┐                   │
│                     │   Skills/    │                   │
│                     │  (external   │                   │
│                     │  processes)  │                   │
│                     └──────────────┘                   │
└─────────────────────────────────────────────────────────┘
```

### Package Responsibilities

#### internal/store
- SQLite database operations
- Schema migrations
- CRUD operations for chunks, messages, sessions, audit logs
- Vector storage and retrieval

#### internal/llm
- LLM provider abstraction
- Ollama, OpenAI, Anthropic implementations
- Embedding generation
- Streaming chat completions
- Privacy mode enforcement

#### internal/rag
- Text chunking with overlap
- Vector similarity search (cosine similarity)
- Prompt construction with retrieved context

#### internal/ingest
- Document parsing (text, PDF, HTML)
- PII detection
- Guardrails enforcement
- Auto-summarization

#### internal/api
- HTTP request handlers
- WebSocket hub for real-time updates
- Template rendering
- Session management

#### internal/skills
- Skill discovery and loading
- Subprocess execution
- Trigger handling (manual, timer, keyword, event)
- JSON communication protocol

#### internal/watcher
- Filesystem monitoring with fsnotify
- Automatic ingestion on file changes
- Event queue management

#### internal/config
- Configuration loading and validation
- Environment variable overrides
- Default configuration generation

#### internal/logging
- Structured logging with levels
- Component-based logging
- Optional file output with rotation

---

## Technology Stack

### Backend
- **Language**: Go 1.21+
- **Database**: SQLite (modernc.org/sqlite - pure Go, no CGo)
- **HTTP Server**: Go standard library net/http
- **WebSocket**: gorilla/websocket
- **Markdown**: goldmark
- **File Watching**: fsnotify
- **PDF Parsing**: go-shiori/go-readability

### Frontend
- **HTMX**: Partial page updates without full reloads
- **Vanilla JavaScript**: WebSocket handling, command palette, interactions
- **CSS**: Custom responsive design with transitions
- **Markdown Rendering**: Server-side with goldmark

### LLM Providers
- **Ollama**: Local models (llama3.2, nomic-embed-text, etc.)
- **OpenAI**: GPT-4, GPT-3.5-turbo, text-embedding-3-small/large
- **Anthropic**: Claude 3 (Opus, Sonnet, Haiku)

---

## Performance Characteristics

- **Binary Size**: ~15-20MB (single executable)
- **Memory Baseline**: ~50-100MB (idle)
- **Memory Under Load**: ~200-500MB (depends on model and concurrent operations)
- **Database Size**: Varies by content (embeddings are ~1KB per chunk)
- **Startup Time**: <1 second
- **Embedding Speed**: Depends on provider (Ollama: ~100ms, OpenAI: ~200ms)
- **Chat Response**: Streaming starts in <500ms

---

## Security Considerations

### Default Security Posture

- Binds to `127.0.0.1` (localhost only)
- Privacy mode enabled by default
- PII detection enabled
- File size limits enforced
- Extension allowlists
- Audit logging enabled

### Recommended Practices

1. **Keep localhost binding** unless remote access is required
2. **Enable privacy mode** for sensitive data
3. **Review skills** before enabling (especially network-dependent ones)
4. **Use strong API keys** for cloud providers
5. **Regularly review audit logs** for unexpected activity
6. **Keep Noodexx updated** for security patches
7. **Backup database** regularly
8. **Limit watched folders** to trusted directories

### Threat Model

Noodexx is designed for:
- ✅ Single-user local deployment
- ✅ Trusted local network deployment
- ❌ Public internet exposure (not recommended)
- ❌ Multi-tenant environments (not supported)

---

## License

[Your License Here]

---

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Submit a pull request

---

## Acknowledgments

Built with:
- [Ollama](https://ollama.ai/) - Local LLM runtime
- [HTMX](https://htmx.org/) - Modern web interactions
- [goldmark](https://github.com/yuin/goldmark) - Markdown rendering
- [fsnotify](https://github.com/fsnotify/fsnotify) - File system notifications
- [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket support

---

**Noodexx** - Your privacy-first AI knowledge assistant
