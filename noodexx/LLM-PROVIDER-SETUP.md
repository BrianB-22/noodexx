# LLM Provider Setup Guide

This guide explains how to configure Noodexx's dual-provider AI system, which allows you to run both local AI (Ollama) and cloud AI (OpenAI/Anthropic) simultaneously and switch between them instantly.

## Overview

Noodexx supports a dual-provider architecture where you can:
- Configure both a local AI provider and a cloud AI provider at the same time
- Switch between them instantly using the privacy toggle in the chat UI
- Control whether your documents are sent to cloud providers (RAG policy)

## Configuration Structure

Your `config.json` file now supports separate sections for local and cloud providers:

```json
{
  "local_provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text:latest",
    "ollama_chat_model": "llama3.2"
  },
  "cloud_provider": {
    "type": "openai",
    "openai_key": "sk-your-key-here",
    "openai_embed_model": "text-embedding-3-small",
    "openai_chat_model": "gpt-4"
  },
  "privacy": {
    "use_local_ai": true,
    "cloud_rag_policy": "no_rag"
  }
}
```

## Configuration Options

### Local Provider (Ollama)

The local provider must be Ollama running on your machine:

```json
"local_provider": {
  "type": "ollama",
  "ollama_endpoint": "http://localhost:11434",
  "ollama_embed_model": "nomic-embed-text:latest",
  "ollama_chat_model": "llama3.2"
}
```

**Required fields:**
- `type`: Must be `"ollama"`
- `ollama_endpoint`: Must be a localhost URL (e.g., `http://localhost:11434` or `http://127.0.0.1:11434`)
- `ollama_embed_model`: The embedding model name (e.g., `nomic-embed-text:latest`)
- `ollama_chat_model`: The chat model name (e.g., `llama3.2`, `mistral`, etc.)

**To leave unconfigured:** Set `type` to empty string `""` or omit the section entirely.

### Cloud Provider (OpenAI or Anthropic)

You can configure either OpenAI or Anthropic as your cloud provider.

#### OpenAI Configuration

```json
"cloud_provider": {
  "type": "openai",
  "openai_key": "sk-proj-...",
  "openai_embed_model": "text-embedding-3-small",
  "openai_chat_model": "gpt-4"
}
```

**Required fields:**
- `type`: Must be `"openai"`
- `openai_key`: Your OpenAI API key (starts with `sk-`)
- `openai_embed_model`: Embedding model (e.g., `text-embedding-3-small`, `text-embedding-3-large`)
- `openai_chat_model`: Chat model (e.g., `gpt-4`, `gpt-4-turbo`, `gpt-3.5-turbo`)

#### Anthropic Configuration

```json
"cloud_provider": {
  "type": "anthropic",
  "anthropic_key": "sk-ant-...",
  "anthropic_chat_model": "claude-3-sonnet-20240229"
}
```

**Required fields:**
- `type`: Must be `"anthropic"`
- `anthropic_key`: Your Anthropic API key (starts with `sk-ant-`)
- `anthropic_chat_model`: Chat model (e.g., `claude-3-sonnet-20240229`, `claude-3-opus-20240229`)

**Note:** Anthropic doesn't require a separate embedding model configuration.

**To leave unconfigured:** Set `type` to empty string `""` or omit the section entirely.

### Privacy Settings

The privacy section controls which provider is active and how RAG (document context) is handled:

```json
"privacy": {
  "use_local_ai": true,
  "cloud_rag_policy": "no_rag"
}
```

**Fields:**
- `use_local_ai`: 
  - `true` = Use local AI provider (Ollama)
  - `false` = Use cloud AI provider (OpenAI/Anthropic)
  - This can be toggled in the UI without editing the config file

- `cloud_rag_policy`: Controls whether document context is sent to cloud providers
  - `"no_rag"` = Don't send your documents to cloud AI (more private, less context)
  - `"allow_rag"` = Send document context to cloud AI (less private, more accurate answers)
  - This setting only affects cloud providers; local AI always uses RAG

## Common Setup Scenarios

### Scenario 1: Local AI Only (Maximum Privacy)

Best for: Sensitive data, offline work, complete privacy

```json
{
  "local_provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text:latest",
    "ollama_chat_model": "llama3.2"
  },
  "cloud_provider": {
    "type": ""
  },
  "privacy": {
    "use_local_ai": true,
    "cloud_rag_policy": "no_rag"
  }
}
```

### Scenario 2: Cloud AI Only

Best for: Maximum AI capability, no local setup required

```json
{
  "local_provider": {
    "type": ""
  },
  "cloud_provider": {
    "type": "openai",
    "openai_key": "sk-...",
    "openai_embed_model": "text-embedding-3-small",
    "openai_chat_model": "gpt-4"
  },
  "privacy": {
    "use_local_ai": false,
    "cloud_rag_policy": "allow_rag"
  }
}
```

### Scenario 3: Both Providers (Recommended)

Best for: Flexibility to switch based on query sensitivity

```json
{
  "local_provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    "ollama_embed_model": "nomic-embed-text:latest",
    "ollama_chat_model": "llama3.2"
  },
  "cloud_provider": {
    "type": "openai",
    "openai_key": "sk-...",
    "openai_embed_model": "text-embedding-3-small",
    "openai_chat_model": "gpt-4"
  },
  "privacy": {
    "use_local_ai": true,
    "cloud_rag_policy": "no_rag"
  }
}
```

With both configured, you can:
- Use local AI for sensitive queries about your documents
- Switch to cloud AI for general questions or when you need more powerful models
- Toggle between them instantly using the privacy switch in the chat UI

## Using the Privacy Toggle

Once you have providers configured, you can switch between them in the chat interface:

1. Look for the privacy toggle in the chat sidebar (above the message input)
2. Click "🔒 Local AI" to use your local Ollama instance
3. Click "☁️ Cloud AI" to use your cloud provider (OpenAI/Anthropic)
4. The active provider is shown at the top of the chat interface

The toggle state is saved automatically, so your choice persists across sessions.

## RAG (Document Context) Behavior

**RAG** (Retrieval Augmented Generation) means searching your local documents and sending relevant snippets to the AI as context.

### With Local AI
- RAG is **always enabled**
- Your documents never leave your machine
- The `cloud_rag_policy` setting has no effect

### With Cloud AI
- RAG behavior depends on the `cloud_rag_policy` setting:
  - `"no_rag"`: Documents are NOT sent to cloud (more private, AI has no context)
  - `"allow_rag"`: Documents ARE sent to cloud (less private, AI has full context)

The current RAG status is displayed in the chat interface next to the provider name.

## Validation Rules

Noodexx validates your configuration on startup and when saving settings:

### Local Provider Rules
- Type must be `"ollama"` (or empty to disable)
- Endpoint must use `localhost` or `127.0.0.1`
- Both embed and chat models must be specified

### Cloud Provider Rules
- Type must be `"openai"` or `"anthropic"` (or empty to disable)
- API key is required for the selected provider
- Required models must be specified

### Privacy Rules
- `cloud_rag_policy` must be either `"no_rag"` or `"allow_rag"`

If validation fails, you'll see specific error messages indicating which fields need correction.

## Backward Compatibility

If you have an existing `config.json` with the old single-provider format:

```json
{
  "provider": {
    "type": "ollama",
    "ollama_endpoint": "http://localhost:11434",
    ...
  }
}
```

Noodexx will automatically migrate it on first load:
- If the old provider was Ollama, it becomes `local_provider`
- If the old provider was OpenAI/Anthropic, it becomes `cloud_provider`
- The old `provider` section is kept for compatibility but the new sections take precedence

## Troubleshooting

### "Provider not configured" error
- Check that the provider you're trying to use has `type` set correctly
- Verify all required fields are filled in for that provider type
- Make sure API keys are valid (for cloud providers)

### "Unable to connect to provider" error
- For Ollama: Ensure Ollama is running (`ollama serve`)
- For Ollama: Check the endpoint URL is correct
- For cloud providers: Verify your API key is valid and has credits

### RAG not working as expected
- With local AI: RAG is always on, check that documents are ingested
- With cloud AI: Check the `cloud_rag_policy` setting matches your expectation
- Look at the RAG status indicator in the chat interface

### Settings not persisting
- Ensure the config file is writable
- Check file permissions on `config.json`
- Look for error messages in the application logs

## Security Best Practices

1. **API Keys**: Never commit `config.json` with real API keys to version control
2. **Local First**: Use local AI for sensitive documents and queries
3. **RAG Policy**: Set `cloud_rag_policy` to `"no_rag"` if your documents contain sensitive information
4. **Endpoint Security**: Only use localhost endpoints for local providers
5. **File Permissions**: Restrict read access to `config.json` since it contains API keys

## Getting Help

If you encounter issues:
1. Check the application logs for detailed error messages
2. Verify your configuration against the examples in this guide
3. Test each provider independently before using both
4. Use the settings page in the UI to validate your configuration
