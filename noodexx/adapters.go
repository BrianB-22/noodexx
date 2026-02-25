package main

import (
	"context"
	"io"
	"time"

	"noodexx/internal/api"
	"noodexx/internal/ingest"
	"noodexx/internal/llm"
	"noodexx/internal/logging"
	"noodexx/internal/rag"
	"noodexx/internal/skills"
	"noodexx/internal/store"
	"noodexx/internal/watcher"
)

// storeAdapter adapts store.Store to rag.Store interface
type storeAdapter struct {
	store *store.Store
}

func (sa *storeAdapter) Search(ctx context.Context, queryVec []float32, topK int) ([]rag.Chunk, error) {
	storeChunks, err := sa.store.Search(ctx, queryVec, topK)
	if err != nil {
		return nil, err
	}

	// Convert store.Chunk to rag.Chunk
	ragChunks := make([]rag.Chunk, len(storeChunks))
	for i, sc := range storeChunks {
		ragChunks[i] = rag.Chunk{
			Source: sc.Source,
			Text:   sc.Text,
			Score:  0, // Score calculated by store
		}
	}
	return ragChunks, nil
}

// providerAdapter adapts llm.Provider to ingest.LLMProvider interface
type providerAdapter struct {
	provider llm.Provider
}

func (pa *providerAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return pa.provider.Embed(ctx, text)
}

func (pa *providerAdapter) Stream(ctx context.Context, messages []ingest.Message, w io.Writer) (string, error) {
	// Convert ingest.Message to llm.Message
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return pa.provider.Stream(ctx, llmMessages, w)
}

// skillsLoaderAdapter adapts skills.Loader to api.SkillsLoader interface
type skillsLoaderAdapter struct {
	loader interface {
		LoadAll() ([]*skillsSkill, error)
	}
}

type skillsSkill struct {
	Name        string
	Version     string
	Description string
	Executable  string
	Triggers    []skillsTrigger
	Timeout     time.Duration
	RequiresNet bool
	Path        string
}

type skillsTrigger struct {
	Type       string
	Parameters map[string]interface{}
}

func (sla *skillsLoaderAdapter) LoadAll() ([]*api.Skill, error) {
	skills, err := sla.loader.LoadAll()
	if err != nil {
		return nil, err
	}

	// Convert skills.Skill to api.Skill
	apiSkills := make([]*api.Skill, len(skills))
	for i, s := range skills {
		triggers := make([]api.SkillTrigger, len(s.Triggers))
		for j, t := range s.Triggers {
			triggers[j] = api.SkillTrigger{
				Type:       t.Type,
				Parameters: t.Parameters,
			}
		}

		apiSkills[i] = &api.Skill{
			Name:        s.Name,
			Version:     s.Version,
			Description: s.Description,
			Executable:  s.Executable,
			Triggers:    triggers,
			Timeout:     s.Timeout,
			RequiresNet: s.RequiresNet,
			Path:        s.Path,
		}
	}
	return apiSkills, nil
}

// skillsExecutorAdapter adapts skills.Executor to api.SkillsExecutor interface
type skillsExecutorAdapter struct {
	executor interface {
		Execute(ctx context.Context, skill *skillsSkill, input skillsInput) (*skillsOutput, error)
	}
}

type skillsInput struct {
	Query    string
	Context  map[string]interface{}
	Settings map[string]interface{}
}

type skillsOutput struct {
	Result   string
	Error    string
	Metadata map[string]interface{}
}

func (sea *skillsExecutorAdapter) Execute(ctx context.Context, skill *api.Skill, input api.SkillInput) (*api.SkillOutput, error) {
	// Convert api.Skill to skills.Skill
	triggers := make([]skillsTrigger, len(skill.Triggers))
	for i, t := range skill.Triggers {
		triggers[i] = skillsTrigger{
			Type:       t.Type,
			Parameters: t.Parameters,
		}
	}

	skillsSkill := &skillsSkill{
		Name:        skill.Name,
		Version:     skill.Version,
		Description: skill.Description,
		Executable:  skill.Executable,
		Triggers:    triggers,
		Timeout:     skill.Timeout,
		RequiresNet: skill.RequiresNet,
		Path:        skill.Path,
	}

	// Convert api.SkillInput to skills.Input
	skillsInput := skillsInput{
		Query:    input.Query,
		Context:  input.Context,
		Settings: input.Settings,
	}

	// Execute
	output, err := sea.executor.Execute(ctx, skillsSkill, skillsInput)
	if err != nil {
		return nil, err
	}

	// Convert skills.Output to api.SkillOutput
	return &api.SkillOutput{
		Result:   output.Result,
		Error:    output.Error,
		Metadata: output.Metadata,
	}, nil
}

// watcherStoreAdapter adapts store.Store to watcher.Store interface
type watcherStoreAdapter struct {
	store *store.Store
}

func (wsa *watcherStoreAdapter) AddWatchedFolder(ctx context.Context, path string) error {
	return wsa.store.AddWatchedFolder(ctx, path)
}

func (wsa *watcherStoreAdapter) GetWatchedFolders(ctx context.Context) ([]watcher.WatchedFolder, error) {
	storeWatchedFolders, err := wsa.store.GetWatchedFolders(ctx)
	if err != nil {
		return nil, err
	}

	// Convert store.WatchedFolder to watcher.WatchedFolder
	watcherFolders := make([]watcher.WatchedFolder, len(storeWatchedFolders))
	for i, swf := range storeWatchedFolders {
		watcherFolders[i] = watcher.WatchedFolder{
			ID:       swf.ID,
			Path:     swf.Path,
			Active:   swf.Active,
			LastScan: swf.LastScan,
		}
	}
	return watcherFolders, nil
}

func (wsa *watcherStoreAdapter) DeleteSource(ctx context.Context, source string) error {
	return wsa.store.DeleteSource(ctx, source)
}

// apiStoreAdapter adapts store.Store to api.Store interface
type apiStoreAdapter struct {
	store *store.Store
}

func (asa *apiStoreAdapter) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	return asa.store.SaveChunk(ctx, source, text, embedding, tags, summary)
}

func (asa *apiStoreAdapter) Search(ctx context.Context, queryVec []float32, topK int) ([]api.Chunk, error) {
	storeChunks, err := asa.store.Search(ctx, queryVec, topK)
	if err != nil {
		return nil, err
	}

	// Convert store.Chunk to api.Chunk
	apiChunks := make([]api.Chunk, len(storeChunks))
	for i, sc := range storeChunks {
		apiChunks[i] = api.Chunk{
			Source: sc.Source,
			Text:   sc.Text,
			Score:  0, // Score calculated by store
		}
	}
	return apiChunks, nil
}

func (asa *apiStoreAdapter) Library(ctx context.Context) ([]api.LibraryEntry, error) {
	storeLibrary, err := asa.store.Library(ctx)
	if err != nil {
		return nil, err
	}

	// Convert store.LibraryEntry to api.LibraryEntry
	apiLibrary := make([]api.LibraryEntry, len(storeLibrary))
	for i, sle := range storeLibrary {
		apiLibrary[i] = api.LibraryEntry{
			Source:     sle.Source,
			ChunkCount: sle.ChunkCount,
			Summary:    sle.Summary,
			Tags:       sle.Tags,
			CreatedAt:  sle.CreatedAt,
		}
	}
	return apiLibrary, nil
}

func (asa *apiStoreAdapter) DeleteSource(ctx context.Context, source string) error {
	return asa.store.DeleteSource(ctx, source)
}

func (asa *apiStoreAdapter) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	return asa.store.SaveMessage(ctx, sessionID, role, content)
}

func (asa *apiStoreAdapter) GetSessionHistory(ctx context.Context, sessionID string) ([]api.ChatMessage, error) {
	storeMessages, err := asa.store.GetSessionHistory(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Convert store.ChatMessage to api.ChatMessage
	apiMessages := make([]api.ChatMessage, len(storeMessages))
	for i, sm := range storeMessages {
		apiMessages[i] = api.ChatMessage{
			ID:        sm.ID,
			SessionID: sm.SessionID,
			Role:      sm.Role,
			Content:   sm.Content,
			CreatedAt: sm.CreatedAt,
		}
	}
	return apiMessages, nil
}

func (asa *apiStoreAdapter) ListSessions(ctx context.Context) ([]api.Session, error) {
	storeSessions, err := asa.store.ListSessions(ctx)
	if err != nil {
		return nil, err
	}

	// Convert store.Session to api.Session
	apiSessions := make([]api.Session, len(storeSessions))
	for i, ss := range storeSessions {
		apiSessions[i] = api.Session{
			ID:            ss.ID,
			LastMessageAt: ss.LastMessageAt,
			MessageCount:  ss.MessageCount,
		}
	}
	return apiSessions, nil
}

func (asa *apiStoreAdapter) AddAuditEntry(ctx context.Context, opType, details, userCtx string) error {
	return asa.store.AddAuditEntry(ctx, opType, details, userCtx)
}

func (asa *apiStoreAdapter) GetAuditLog(ctx context.Context, opType string, from, to time.Time) ([]api.AuditEntry, error) {
	storeAudit, err := asa.store.GetAuditLog(ctx, opType, from, to)
	if err != nil {
		return nil, err
	}

	// Convert store.AuditEntry to api.AuditEntry
	apiAudit := make([]api.AuditEntry, len(storeAudit))
	for i, sa := range storeAudit {
		apiAudit[i] = api.AuditEntry{
			ID:            sa.ID,
			Timestamp:     sa.Timestamp,
			OperationType: sa.OperationType,
			Details:       sa.Details,
			UserContext:   sa.UserContext,
		}
	}
	return apiAudit, nil
}

// apiProviderAdapter adapts llm.Provider to api.LLMProvider interface
type apiProviderAdapter struct {
	provider llm.Provider
}

func (apa *apiProviderAdapter) Embed(ctx context.Context, text string) ([]float32, error) {
	return apa.provider.Embed(ctx, text)
}

func (apa *apiProviderAdapter) Stream(ctx context.Context, messages []api.Message, w io.Writer) (string, error) {
	// Convert api.Message to llm.Message
	llmMessages := make([]llm.Message, len(messages))
	for i, msg := range messages {
		llmMessages[i] = llm.Message{
			Role:    msg.Role,
			Content: msg.Content,
		}
	}
	return apa.provider.Stream(ctx, llmMessages, w)
}

func (apa *apiProviderAdapter) Name() string {
	return apa.provider.Name()
}

func (apa *apiProviderAdapter) IsLocal() bool {
	return apa.provider.IsLocal()
}

// apiSearcherAdapter adapts rag.Searcher to api.Searcher interface
type apiSearcherAdapter struct {
	searcher *rag.Searcher
}

func (asa *apiSearcherAdapter) Search(ctx context.Context, queryVec []float32, topK int) ([]api.Chunk, error) {
	ragChunks, err := asa.searcher.Search(ctx, queryVec, topK)
	if err != nil {
		return nil, err
	}

	// Convert rag.Chunk to api.Chunk
	apiChunks := make([]api.Chunk, len(ragChunks))
	for i, rc := range ragChunks {
		apiChunks[i] = api.Chunk{
			Source: rc.Source,
			Text:   rc.Text,
			Score:  rc.Score,
		}
	}
	return apiChunks, nil
}

// apiSkillsLoaderAdapter adapts skills.Loader to api.SkillsLoader interface
type apiSkillsLoaderAdapter struct {
	loader *skills.Loader
}

func (asla *apiSkillsLoaderAdapter) LoadAll() ([]*api.Skill, error) {
	skillsList, err := asla.loader.LoadAll()
	if err != nil {
		return nil, err
	}

	// Convert skills.Skill to api.Skill
	apiSkills := make([]*api.Skill, len(skillsList))
	for i, s := range skillsList {
		triggers := make([]api.SkillTrigger, len(s.Triggers))
		for j, t := range s.Triggers {
			triggers[j] = api.SkillTrigger{
				Type:       t.Type,
				Parameters: t.Parameters,
			}
		}

		apiSkills[i] = &api.Skill{
			Name:        s.Name,
			Version:     s.Version,
			Description: s.Description,
			Executable:  s.Executable,
			Triggers:    triggers,
			Timeout:     s.Timeout,
			RequiresNet: s.RequiresNet,
			Path:        s.Path,
		}
	}
	return apiSkills, nil
}

// apiSkillsExecutorAdapter adapts skills.Executor to api.SkillsExecutor interface
type apiSkillsExecutorAdapter struct {
	executor *skills.Executor
}

func (asea *apiSkillsExecutorAdapter) Execute(ctx context.Context, skill *api.Skill, input api.SkillInput) (*api.SkillOutput, error) {
	// Convert api.Skill to skills.Skill
	triggers := make([]skills.Trigger, len(skill.Triggers))
	for i, t := range skill.Triggers {
		triggers[i] = skills.Trigger{
			Type:       t.Type,
			Parameters: t.Parameters,
		}
	}

	skillsSkill := &skills.Skill{
		Name:        skill.Name,
		Version:     skill.Version,
		Description: skill.Description,
		Executable:  skill.Executable,
		Triggers:    triggers,
		Timeout:     skill.Timeout,
		RequiresNet: skill.RequiresNet,
		Path:        skill.Path,
	}

	// Convert api.SkillInput to skills.Input
	skillsInput := skills.Input{
		Query:    input.Query,
		Context:  input.Context,
		Settings: input.Settings,
	}

	// Execute
	output, err := asea.executor.Execute(ctx, skillsSkill, skillsInput)
	if err != nil {
		return nil, err
	}

	// Convert skills.Output to api.SkillOutput
	return &api.SkillOutput{
		Result:   output.Result,
		Error:    output.Error,
		Metadata: output.Metadata,
	}, nil
}

// apiLoggerAdapter adapts logging.Logger to api.Logger interface
type apiLoggerAdapter struct {
	logger *logging.Logger
}

func (ala *apiLoggerAdapter) Debug(format string, args ...interface{}) {
	ala.logger.Debug(format, args...)
}

func (ala *apiLoggerAdapter) Info(format string, args ...interface{}) {
	ala.logger.Info(format, args...)
}

func (ala *apiLoggerAdapter) Warn(format string, args ...interface{}) {
	ala.logger.Warn(format, args...)
}

func (ala *apiLoggerAdapter) Error(format string, args ...interface{}) {
	ala.logger.Error(format, args...)
}

func (ala *apiLoggerAdapter) WithContext(key string, value interface{}) api.Logger {
	return &apiLoggerAdapter{logger: ala.logger.WithContext(key, value)}
}

func (ala *apiLoggerAdapter) WithFields(fields map[string]interface{}) api.Logger {
	return &apiLoggerAdapter{logger: ala.logger.WithFields(fields)}
}
