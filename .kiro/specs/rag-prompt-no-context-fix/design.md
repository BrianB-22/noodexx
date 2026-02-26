# RAG Prompt No Context Fix - Bugfix Design

## Overview

The bug occurs in the `BuildPrompt()` function in `noodexx/internal/rag/prompt.go`, which unconditionally includes context-related instructions even when no RAG chunks are provided (empty chunks array). This causes the AI to respond with misleading messages about "provided context" when users have explicitly disabled RAG via `cloud_rag_policy: "no_rag"`.

The fix is straightforward: add a conditional check for empty chunks and generate a simple prompt without context instructions when RAG is disabled, while preserving the existing behavior when chunks are present.

## Glossary

- **Bug_Condition (C)**: The condition that triggers the bug - when `BuildPrompt()` is called with an empty chunks array (RAG disabled)
- **Property (P)**: The desired behavior when chunks is empty - generate a simple prompt without any context-related instructions
- **Preservation**: Existing RAG prompt behavior when chunks are present must remain unchanged
- **BuildPrompt()**: The function in `noodexx/internal/rag/prompt.go` that constructs prompts by combining user queries with retrieved RAG context
- **chunks**: Array of `Chunk` structs containing retrieved document segments with Source and Text fields
- **cloud_rag_policy**: Configuration setting that controls RAG behavior ("no_rag" disables RAG search)

## Bug Details

### Fault Condition

The bug manifests when `BuildPrompt()` is called with an empty chunks array (length 0), which occurs when RAG is disabled via `cloud_rag_policy: "no_rag"`. The function unconditionally includes context-related instructions regardless of whether chunks are present, causing the AI to believe it should reference non-existent context.

**Formal Specification:**
```
FUNCTION isBugCondition(input)
  INPUT: input of type BuildPromptInput {query: string, chunks: []Chunk}
  OUTPUT: boolean
  
  RETURN len(input.chunks) == 0
         AND prompt contains "Use the following context"
         AND prompt contains "Context:"
         AND prompt contains "Answer based on the context above:"
END FUNCTION
```

### Examples

- **Example 1**: User asks "What is Go?" with `cloud_rag_policy: "no_rag"`
  - Current: AI responds "I cannot answer based on the provided context"
  - Expected: AI responds naturally "Go is a programming language..."

- **Example 2**: User asks "How do I configure the server?" with RAG disabled
  - Current: Prompt includes "Use the following context" with empty Context section
  - Expected: Prompt is simply "You are a helpful assistant.\n\nUser Question: How do I configure the server?"

- **Example 3**: User asks "Explain authentication" with `cloud_rag_policy: "no_rag"`
  - Current: Prompt includes "Answer based on the context above:" with no context
  - Expected: Prompt has no context-related instructions

- **Edge Case**: User asks question with `cloud_rag_policy: "always"` but no documents in library
  - Current: Includes context instructions with empty chunks (triggers bug)
  - Expected: Should generate simple prompt without context instructions

## Expected Behavior

### Preservation Requirements

**Unchanged Behaviors:**
- When chunks array contains RAG data (len > 0), the prompt must continue to include "Use the following context to answer the user's question"
- When chunks array contains RAG data, the "Context:" section with numbered sources must continue to be included
- When chunks array contains RAG data, the "Answer based on the context above:" instruction must continue to be included
- The formatting of RAG context with source attribution `[N] Source: ...` must remain unchanged

**Scope:**
All inputs where chunks array is NOT empty should be completely unaffected by this fix. This includes:
- RAG-enabled queries with retrieved documents
- Formatting of chunk sources and text
- Order and numbering of context items

## Hypothesized Root Cause

Based on the bug description and code analysis, the root cause is clear:

1. **Missing Conditional Logic**: The `BuildPrompt()` function has no check for empty chunks array
   - The function always writes "Use the following context" regardless of chunks length
   - The function always writes "Context:" section even when empty
   - The function always writes "Answer based on the context above:" even with no context

2. **Unconditional String Building**: All prompt components are written unconditionally
   - No early return or branching based on chunks length
   - The for loop over chunks is silent when empty (writes nothing) but surrounding text remains

3. **Design Assumption**: The function was designed assuming chunks would always be present
   - No consideration for the no_rag policy scenario
   - No handling of the empty chunks case

## Correctness Properties

Property 1: Fault Condition - No Context Instructions When RAG Disabled

_For any_ input where the chunks array is empty (len(chunks) == 0), the fixed BuildPrompt function SHALL generate a simple prompt containing only the system message and user question, without any context-related instructions ("Use the following context", "Context:", "Answer based on the context above:").

**Validates: Requirements 2.1, 2.2, 2.3, 2.4**

Property 2: Preservation - RAG Prompt Format When Enabled

_For any_ input where the chunks array is NOT empty (len(chunks) > 0), the fixed BuildPrompt function SHALL produce exactly the same prompt format as the original function, preserving all context instructions, the Context section with numbered sources, and the "Answer based on the context above:" instruction.

**Validates: Requirements 3.1, 3.2, 3.3, 3.4**

## Fix Implementation

### Changes Required

**File**: `noodexx/internal/rag/prompt.go`

**Function**: `BuildPrompt(query string, chunks []Chunk) string`

**Specific Changes**:
1. **Add Empty Chunks Check**: Add conditional logic at the start of the function
   - Check if `len(chunks) == 0`
   - If true, return a simple prompt without context instructions

2. **Simple Prompt Format**: When chunks is empty, return:
   ```
   You are a helpful assistant.
   
   User Question: [query]
   ```

3. **Preserve Existing Logic**: Keep all existing code for non-empty chunks case
   - No changes to context formatting
   - No changes to source attribution
   - No changes to instruction text

4. **Implementation Pattern**:
   ```go
   func (pb *PromptBuilder) BuildPrompt(query string, chunks []Chunk) string {
       // Handle empty chunks case (RAG disabled)
       if len(chunks) == 0 {
           return fmt.Sprintf("You are a helpful assistant.\n\nUser Question: %s", query)
       }
       
       // Existing logic for non-empty chunks (RAG enabled)
       var sb strings.Builder
       sb.WriteString("You are a helpful assistant. Use the following context to answer the user's question.\n\n")
       // ... rest of existing code unchanged
   }
   ```

## Testing Strategy

### Validation Approach

The testing strategy follows a two-phase approach: first, surface counterexamples that demonstrate the bug on unfixed code, then verify the fix works correctly and preserves existing behavior.

### Exploratory Fault Condition Checking

**Goal**: Surface counterexamples that demonstrate the bug BEFORE implementing the fix. Confirm that empty chunks produce prompts with context instructions.

**Test Plan**: Write tests that call `BuildPrompt()` with empty chunks array and assert that the generated prompt contains context-related instructions. Run these tests on the UNFIXED code to observe failures and confirm the root cause.

**Test Cases**:
1. **Empty Chunks Test**: Call `BuildPrompt("What is Go?", []Chunk{})` on unfixed code
   - Expected failure: Prompt contains "Use the following context"
   - Expected failure: Prompt contains "Context:"
   - Expected failure: Prompt contains "Answer based on the context above:"

2. **No RAG Policy Simulation**: Simulate the no_rag policy scenario with empty chunks
   - Expected failure: Generated prompt misleads AI about context availability

3. **Empty Library Scenario**: Test with valid query but no documents in library
   - Expected failure: Context instructions present with no actual context

**Expected Counterexamples**:
- Prompt includes "Use the following context to answer the user's question" when chunks is empty
- Prompt includes empty "Context:" section
- Prompt includes "Answer based on the context above:" with no context above
- AI would respond with "I cannot answer based on the provided context" when no context was provided

### Fix Checking

**Goal**: Verify that for all inputs where the bug condition holds (empty chunks), the fixed function produces the expected behavior (no context instructions).

**Pseudocode:**
```
FOR ALL input WHERE isBugCondition(input) DO
  result := BuildPrompt_fixed(input.query, input.chunks)
  ASSERT NOT contains(result, "Use the following context")
  ASSERT NOT contains(result, "Context:")
  ASSERT NOT contains(result, "Answer based on the context above:")
  ASSERT contains(result, "You are a helpful assistant")
  ASSERT contains(result, input.query)
END FOR
```

### Preservation Checking

**Goal**: Verify that for all inputs where the bug condition does NOT hold (non-empty chunks), the fixed function produces the same result as the original function.

**Pseudocode:**
```
FOR ALL input WHERE NOT isBugCondition(input) DO
  ASSERT BuildPrompt_original(input.query, input.chunks) = BuildPrompt_fixed(input.query, input.chunks)
END FOR
```

**Testing Approach**: Property-based testing is recommended for preservation checking because:
- It generates many test cases automatically across the input domain (various chunk counts, sources, text lengths)
- It catches edge cases that manual unit tests might miss (special characters, long text, many chunks)
- It provides strong guarantees that behavior is unchanged for all RAG-enabled scenarios

**Test Plan**: Observe behavior on UNFIXED code first with various non-empty chunks arrays, then write property-based tests capturing that exact behavior.

**Test Cases**:
1. **Single Chunk Preservation**: Verify prompt format with 1 chunk remains unchanged
2. **Multiple Chunks Preservation**: Verify prompt format with 2-10 chunks remains unchanged
3. **Source Attribution Preservation**: Verify `[N] Source: ...` formatting continues working
4. **Special Characters Preservation**: Verify chunks with special characters are handled identically

### Unit Tests

- Test `BuildPrompt()` with empty chunks array returns simple prompt without context instructions
- Test `BuildPrompt()` with single chunk includes all context instructions
- Test `BuildPrompt()` with multiple chunks formats them correctly with numbering
- Test edge case: very long query with empty chunks
- Test edge case: empty query string with empty chunks
- Test edge case: query with special characters and empty chunks

### Property-Based Tests

- Generate random queries (various lengths, characters) with empty chunks and verify no context instructions appear
- Generate random chunk arrays (1-100 chunks, various sources and text) and verify prompt format matches original
- Generate random combinations of query and chunks and verify the fix/preservation properties hold
- Test that source attribution numbering is always sequential regardless of chunk count

### Integration Tests

- Test full request flow with `cloud_rag_policy: "no_rag"` and verify AI responds naturally
- Test switching between `no_rag` and `always` policies and verify prompt changes appropriately
- Test with empty document library and `cloud_rag_policy: "always"` (should use simple prompt)
- Test that RAG-enabled queries continue to work with proper context formatting
