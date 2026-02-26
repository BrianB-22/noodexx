# Bugfix Requirements Document

## Introduction

When using cloud AI with `cloud_rag_policy` set to `"no_rag"`, the AI responses incorrectly mention "provided context" even though no RAG search is performed and no documents are retrieved. This occurs because the `BuildPrompt()` function in `noodexx/internal/rag/prompt.go` generates context-related instructions even when the chunks array is empty, misleading the AI into believing it should reference context that doesn't exist.

This bug confuses users who have explicitly disabled RAG, as the AI responds with messages like "I cannot answer based on the provided context" when no context was actually provided.

## Bug Analysis

### Current Behavior (Defect)

1.1 WHEN chunks is empty (RAG disabled) THEN the system includes "Use the following context to answer the user's question" in the prompt

1.2 WHEN chunks is empty (RAG disabled) THEN the system includes an empty "Context:" section in the prompt

1.3 WHEN chunks is empty (RAG disabled) THEN the system includes "Answer based on the context above:" instruction in the prompt

1.4 WHEN chunks is empty (RAG disabled) THEN the AI responds with misleading messages like "I cannot answer based on the provided context"

### Expected Behavior (Correct)

2.1 WHEN chunks is empty (RAG disabled) THEN the system SHALL NOT include any context-related instructions in the prompt

2.2 WHEN chunks is empty (RAG disabled) THEN the system SHALL NOT include a "Context:" section in the prompt

2.3 WHEN chunks is empty (RAG disabled) THEN the system SHALL NOT include "Answer based on the context above:" instruction in the prompt

2.4 WHEN chunks is empty (RAG disabled) THEN the AI SHALL respond naturally without any reference to "provided context"

### Unchanged Behavior (Regression Prevention)

3.1 WHEN chunks contains RAG data (RAG enabled) THEN the system SHALL CONTINUE TO include "Use the following context to answer the user's question" in the prompt

3.2 WHEN chunks contains RAG data (RAG enabled) THEN the system SHALL CONTINUE TO include the "Context:" section with retrieved documents in the prompt

3.3 WHEN chunks contains RAG data (RAG enabled) THEN the system SHALL CONTINUE TO include "Answer based on the context above:" instruction in the prompt

3.4 WHEN chunks contains RAG data (RAG enabled) THEN the AI SHALL CONTINUE TO answer based on the provided context
