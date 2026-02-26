# Implementation Plan

- [x] 1. Write bug condition exploration test
  - **Property 1: Fault Condition** - Empty Chunks Produce Context Instructions
  - **CRITICAL**: This test MUST FAIL on unfixed code - failure confirms the bug exists
  - **DO NOT attempt to fix the test or the code when it fails**
  - **NOTE**: This test encodes the expected behavior - it will validate the fix when it passes after implementation
  - **GOAL**: Surface counterexamples that demonstrate the bug exists
  - **Scoped PBT Approach**: Scope the property to concrete failing cases - empty chunks array with any query string
  - Test that `BuildPrompt(query, []Chunk{})` produces a prompt WITHOUT context instructions ("Use the following context", "Context:", "Answer based on the context above:")
  - Test with various queries: "What is Go?", "How do I configure the server?", "Explain authentication"
  - Run test on UNFIXED code in `noodexx/internal/rag/prompt.go`
  - **EXPECTED OUTCOME**: Test FAILS (this is correct - it proves the bug exists)
  - Document counterexamples found: prompts that incorrectly include context instructions when chunks is empty
  - Mark task complete when test is written, run, and failure is documented
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 2. Write preservation property tests (BEFORE implementing fix)
  - **Property 2: Preservation** - Non-Empty Chunks Preserve RAG Format
  - **IMPORTANT**: Follow observation-first methodology
  - Observe behavior on UNFIXED code for non-empty chunks arrays
  - Test with single chunk: verify prompt includes "Use the following context", "Context:" section with `[1] Source: ...`, and "Answer based on the context above:"
  - Test with multiple chunks (2-5): verify sequential numbering `[1]`, `[2]`, etc. and proper formatting
  - Test with chunks containing special characters and long text
  - Write property-based tests capturing observed RAG prompt format patterns
  - Property-based testing generates many test cases for stronger guarantees
  - Run tests on UNFIXED code in `noodexx/internal/rag/prompt.go`
  - **EXPECTED OUTCOME**: Tests PASS (this confirms baseline behavior to preserve)
  - Mark task complete when tests are written, run, and passing on unfixed code
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 3. Fix BuildPrompt to handle empty chunks correctly

  - [x] 3.1 Implement the fix in noodexx/internal/rag/prompt.go
    - Add conditional check at start of `BuildPrompt()` function: `if len(chunks) == 0`
    - When chunks is empty, return simple prompt: `"You are a helpful assistant.\n\nUser Question: " + query`
    - Preserve all existing logic for non-empty chunks case (no changes to context formatting, source attribution, or instruction text)
    - _Bug_Condition: isBugCondition(input) where len(input.chunks) == 0_
    - _Expected_Behavior: Prompt contains NO context instructions ("Use the following context", "Context:", "Answer based on the context above:") and only includes system message and user question_
    - _Preservation: When len(chunks) > 0, prompt format must remain identical to original with all context instructions, Context section with numbered sources, and "Answer based on the context above:" instruction_
    - _Requirements: 2.1, 2.2, 2.3, 2.4, 3.1, 3.2, 3.3, 3.4_

  - [x] 3.2 Verify bug condition exploration test now passes
    - **Property 1: Expected Behavior** - Empty Chunks Generate Simple Prompt
    - **IMPORTANT**: Re-run the SAME test from task 1 - do NOT write a new test
    - The test from task 1 encodes the expected behavior
    - When this test passes, it confirms the expected behavior is satisfied
    - Run bug condition exploration test from step 1
    - **EXPECTED OUTCOME**: Test PASSES (confirms bug is fixed)
    - _Requirements: 2.1, 2.2, 2.3, 2.4_

  - [x] 3.3 Verify preservation tests still pass
    - **Property 2: Preservation** - Non-Empty Chunks Preserve RAG Format
    - **IMPORTANT**: Re-run the SAME tests from task 2 - do NOT write new tests
    - Run preservation property tests from step 2
    - **EXPECTED OUTCOME**: Tests PASS (confirms no regressions)
    - Confirm all tests still pass after fix (no regressions in RAG-enabled scenarios)
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 4. Checkpoint - Ensure all tests pass
  - Run all unit tests for `noodexx/internal/rag/prompt.go`
  - Run integration tests with `cloud_rag_policy: "no_rag"` to verify AI responds naturally
  - Run integration tests with `cloud_rag_policy: "always"` to verify RAG still works correctly
  - Ensure all tests pass, ask the user if questions arise
