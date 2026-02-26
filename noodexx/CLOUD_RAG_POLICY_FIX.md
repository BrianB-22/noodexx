# Cloud RAG Policy Fix

## Problem

When switching to Cloud AI mode with `cloud_rag_policy: "no_rag"`, the system was still performing RAG searches and the model was responding with "The context provided does not contain information about..." for general knowledge questions.

## Root Cause

The `RAGPolicyEnforcer` was created at startup with a reference to the config object. When the user switched providers via the `/api/privacy-toggle` endpoint:

1. The handler loaded the config from disk
2. Updated the `Privacy.UseLocalAI` field
3. Saved it back to disk
4. Called `providerManager.Reload(cfg)` to update the provider manager

However, the `RAGPolicyEnforcer` still had a reference to the OLD config object in memory, so it continued to use the old `Privacy.UseLocalAI` value.

## Solution

Added a `Reload(cfg interface{})` method to the `RAGPolicyEnforcer` that updates its config reference:

```go
func (e *RAGPolicyEnforcer) Reload(cfg interface{}) {
	if c, ok := cfg.(*config.Config); ok {
		e.config = c
		e.logger.Debug("RAG policy enforcer config reloaded")
	}
}
```

Updated the `handlePrivacyToggle` function to call both:
- `s.providerManager.Reload(cfg)` - updates the provider manager
- `s.ragEnforcer.Reload(cfg)` - updates the RAG policy enforcer

## Files Modified

1. `noodexx/internal/rag/policy_enforcer.go`
   - Added `Reload(cfg interface{})` method

2. `noodexx/internal/api/server.go`
   - Added `Reload(cfg interface{})` to the `RAGEnforcer` interface

3. `noodexx/internal/api/handlers.go`
   - Updated `handlePrivacyToggle` to call `s.ragEnforcer.Reload(cfg)`
   - Fixed duplicate `BuildPrompt` implementation (from previous fix)
   - Added chunk type conversion from `api.Chunk` to `rag.Chunk`

4. `noodexx/adapters.go`
   - Added `Reload(cfg interface{})` method to `apiRAGEnforcerAdapter`

5. `noodexx/web/templates/base.html`
   - Removed alert popup from privacy mode toggle

6. `noodexx/web/templates/chat.html`
   - Removed toast notification from provider switch

## Expected Behavior After Fix

### Local AI Mode (use_local_ai: true)
- RAG is ALWAYS performed
- General knowledge questions: Answered from model's training data (may mention context)
- Document-specific questions: Answered using RAG with local documents
- Status: "🔒 Local AI (ollama) | RAG Enabled (Local)"

### Cloud AI Mode with no_rag (use_local_ai: false, cloud_rag_policy: "no_rag")
- RAG is NEVER performed
- General knowledge questions: Answered from model's training data
- Document-specific questions: Answered from model's training data (no document access)
- Status: "☁️ Cloud AI (OpenAI/Anthropic) | No RAG (Maximum Privacy)"

### Cloud AI Mode with allow_rag (use_local_ai: false, cloud_rag_policy: "allow_rag")
- RAG is performed
- General knowledge questions: Answered from model's training data
- Document-specific questions: Answered using RAG with cloud provider
- Status: "☁️ Cloud AI (OpenAI/Anthropic) | RAG Enabled (Cloud)"

## Testing

To verify the fix:

1. Start the server: `./noodexx`
2. Log in to the web interface
3. Switch to Local AI mode
4. Ask "what is a dog?" - should get a normal answer (may mention context)
5. Switch to Cloud AI mode
6. Ask "what is a dog?" - should get a normal answer WITHOUT mentioning context
7. Check debug.log for RAG policy decisions:
   - Local mode: "RAG enabled: using local AI provider"
   - Cloud mode with no_rag: "RAG disabled: cloud AI with no_rag policy"
8. Verify no popup notifications when switching
9. Verify status indicator updates correctly

## Debug Log Verification

After switching to cloud mode, you should see in debug.log:
```
[timestamp] DEBUG [main] policy_enforcer.go:XX rag.(*RAGPolicyEnforcer).ShouldPerformRAG RAG disabled: cloud AI with no_rag policy
[timestamp] DEBUG [main] adapters.go:XXX main.(*apiLoggerAdapter).Debug skipping RAG search per policy
```

If you still see "RAG enabled: using local AI provider" after switching to cloud mode, the config reload didn't work.
