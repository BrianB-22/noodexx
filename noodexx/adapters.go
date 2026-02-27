package main

import (
	"context"
	"fmt"
	"io"
	"time"

	"noodexx/internal/api"
	"noodexx/internal/auth"
	"noodexx/internal/config"
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
		LoadForUser(ctx context.Context, userID int64) ([]*skillsSkill, error)
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

func (sla *skillsLoaderAdapter) LoadForUser(ctx context.Context, userID int64) ([]*api.Skill, error) {
	skills, err := sla.loader.LoadForUser(ctx, userID)
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

// skillsStoreAdapter adapts store.Store to skills.Store interface
type skillsStoreAdapter struct {
	store *store.Store
}

func (ssa *skillsStoreAdapter) GetUserSkills(ctx context.Context, userID int64) ([]skills.SkillMetadata, error) {
	storeSkills, err := ssa.store.GetUserSkills(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Convert store.Skill to skills.SkillMetadata
	skillsMetadata := make([]skills.SkillMetadata, len(storeSkills))
	for i, s := range storeSkills {
		skillsMetadata[i] = skills.SkillMetadata{
			ID:        s.ID,
			UserID:    s.UserID,
			Name:      s.Name,
			Path:      s.Path,
			Enabled:   s.Enabled,
			CreatedAt: s.CreatedAt,
		}
	}
	return skillsMetadata, nil
}

// watcherStoreAdapter adapts store.Store to watcher.Store interface
type watcherStoreAdapter struct {
	store *store.Store
}

func (wsa *watcherStoreAdapter) AddWatchedFolder(ctx context.Context, userID int64, path string) error {
	return wsa.store.AddWatchedFolder(ctx, userID, path)
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
			UserID:   swf.UserID,
			Path:     swf.Path,
			Active:   swf.Active,
			LastScan: swf.LastScan,
		}
	}
	return watcherFolders, nil
}

func (wsa *watcherStoreAdapter) DeleteSource(ctx context.Context, source string) error {
	// Use local-default user (ID=1) for backward compatibility
	return wsa.store.DeleteChunksBySource(ctx, 1, source)
}

// apiStoreAdapter adapts store.Store to api.Store interface
type apiStoreAdapter struct {
	store *store.Store
}

func (asa *apiStoreAdapter) SaveChunk(ctx context.Context, source, text string, embedding []float32, tags []string, summary string) error {
	// Use local-default user (ID=1) for backward compatibility
	return asa.store.SaveChunk(ctx, 1, source, text, embedding, tags, summary)
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

func (asa *apiStoreAdapter) SearchByUser(ctx context.Context, userID int64, queryVec []float32, topK int) ([]api.Chunk, error) {
	storeChunks, err := asa.store.SearchByUser(ctx, userID, queryVec, topK)
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

func (asa *apiStoreAdapter) LibraryByUser(ctx context.Context, userID int64) ([]api.LibraryEntry, error) {
	storeLibrary, err := asa.store.LibraryByUser(ctx, userID)
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
	// Use local-default user (ID=1) for backward compatibility
	return asa.store.DeleteChunksBySource(ctx, 1, source)
}

func (asa *apiStoreAdapter) SaveMessage(ctx context.Context, sessionID, role, content string) error {
	return asa.store.SaveMessage(ctx, sessionID, role, content)
}

func (asa *apiStoreAdapter) SaveChatMessage(ctx context.Context, userID int64, sessionID, role, content, providerMode string) error {
	return asa.store.SaveChatMessage(ctx, userID, sessionID, role, content, providerMode)
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
			ID:           sm.ID,
			SessionID:    sm.SessionID,
			Role:         sm.Role,
			Content:      sm.Content,
			ProviderMode: sm.ProviderMode,
			CreatedAt:    sm.CreatedAt,
		}
	}
	return apiMessages, nil
}

func (asa *apiStoreAdapter) GetSessionMessages(ctx context.Context, userID int64, sessionID string) ([]api.ChatMessage, error) {
	storeMessages, err := asa.store.GetSessionMessages(ctx, userID, sessionID)
	if err != nil {
		return nil, err
	}

	// Convert store.ChatMessage to api.ChatMessage
	apiMessages := make([]api.ChatMessage, len(storeMessages))
	for i, sm := range storeMessages {
		apiMessages[i] = api.ChatMessage{
			ID:           sm.ID,
			SessionID:    sm.SessionID,
			Role:         sm.Role,
			Content:      sm.Content,
			ProviderMode: sm.ProviderMode,
			CreatedAt:    sm.CreatedAt,
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

func (asa *apiStoreAdapter) GetUserSessions(ctx context.Context, userID int64) ([]api.Session, error) {
	storeSessions, err := asa.store.GetUserSessions(ctx, userID)
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

func (asa *apiStoreAdapter) GetSessionOwner(ctx context.Context, sessionID string) (int64, error) {
	return asa.store.GetSessionOwner(ctx, sessionID)
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

// User management methods
func (asa *apiStoreAdapter) GetUserByUsername(ctx context.Context, username string) (*api.User, error) {
	user, err := asa.store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}
	return &api.User{
		ID:                 user.ID,
		Username:           user.Username,
		PasswordHash:       user.PasswordHash,
		Email:              email,
		IsAdmin:            user.IsAdmin,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt,
		LastLogin:          user.LastLogin,
		DarkMode:           user.DarkMode,
	}, nil
}

func (asa *apiStoreAdapter) GetUserByID(ctx context.Context, userID int64) (*api.User, error) {
	user, err := asa.store.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}
	return &api.User{
		ID:                 user.ID,
		Username:           user.Username,
		PasswordHash:       user.PasswordHash,
		Email:              email,
		IsAdmin:            user.IsAdmin,
		MustChangePassword: user.MustChangePassword,
		CreatedAt:          user.CreatedAt,
		LastLogin:          user.LastLogin,
		DarkMode:           user.DarkMode,
	}, nil
}

func (asa *apiStoreAdapter) CreateUser(ctx context.Context, username, password, email string, isAdmin, mustChangePassword bool) (int64, error) {
	return asa.store.CreateUser(ctx, username, password, email, isAdmin, mustChangePassword)
}

func (asa *apiStoreAdapter) UpdatePassword(ctx context.Context, userID int64, newPassword string) error {
	return asa.store.UpdatePassword(ctx, userID, newPassword)
}

func (asa *apiStoreAdapter) UpdateUserDarkMode(ctx context.Context, userID int64, darkMode bool) error {
	return asa.store.UpdateUserDarkMode(ctx, userID, darkMode)
}

func (asa *apiStoreAdapter) ListUsers(ctx context.Context) ([]api.User, error) {
	storeUsers, err := asa.store.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	apiUsers := make([]api.User, len(storeUsers))
	for i, su := range storeUsers {
		email := ""
		if su.Email.Valid {
			email = su.Email.String
		}
		apiUsers[i] = api.User{
			ID:                 su.ID,
			Username:           su.Username,
			PasswordHash:       su.PasswordHash,
			Email:              email,
			IsAdmin:            su.IsAdmin,
			MustChangePassword: su.MustChangePassword,
			CreatedAt:          su.CreatedAt,
			LastLogin:          su.LastLogin,
			DarkMode:           su.DarkMode,
		}
	}
	return apiUsers, nil
}

func (asa *apiStoreAdapter) DeleteUser(ctx context.Context, userID int64) error {
	return asa.store.DeleteUser(ctx, userID)
}

// Skills management methods
func (asa *apiStoreAdapter) GetUserSkills(ctx context.Context, userID int64) ([]api.Skill, error) {
	storeSkills, err := asa.store.GetUserSkills(ctx, userID)
	if err != nil {
		return nil, err
	}

	apiSkills := make([]api.Skill, len(storeSkills))
	for i, ss := range storeSkills {
		apiSkills[i] = api.Skill{
			Name: ss.Name,
			Path: ss.Path,
		}
	}
	return apiSkills, nil
}

// Watched folders management methods
func (asa *apiStoreAdapter) GetWatchedFoldersByUser(ctx context.Context, userID int64) ([]api.WatchedFolder, error) {
	storeWatchedFolders, err := asa.store.GetWatchedFoldersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	apiWatchedFolders := make([]api.WatchedFolder, len(storeWatchedFolders))
	for i, swf := range storeWatchedFolders {
		apiWatchedFolders[i] = api.WatchedFolder{
			ID:     swf.ID,
			Path:   swf.Path,
			UserID: userID, // Use the userID parameter since it's not in store.WatchedFolder
		}
	}
	return apiWatchedFolders, nil
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

func (asla *apiSkillsLoaderAdapter) LoadForUser(ctx context.Context, userID int64) ([]*api.Skill, error) {
	skillsList, err := asla.loader.LoadForUser(ctx, userID)
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
			UserID:      s.UserID,
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

// apiAuthProviderAdapter adapts auth.Provider to api.AuthProvider interface
type apiAuthProviderAdapter struct {
	provider interface {
		Login(ctx context.Context, username, password string) (string, error)
		Logout(ctx context.Context, token string) error
		ValidateToken(ctx context.Context, token string) (int64, error)
		RefreshToken(ctx context.Context, token string) (string, error)
	}
}

func (aapa *apiAuthProviderAdapter) Login(ctx context.Context, username, password string) (string, error) {
	return aapa.provider.Login(ctx, username, password)
}

func (aapa *apiAuthProviderAdapter) Logout(ctx context.Context, token string) error {
	return aapa.provider.Logout(ctx, token)
}

func (aapa *apiAuthProviderAdapter) ValidateToken(ctx context.Context, token string) (int64, error) {
	return aapa.provider.ValidateToken(ctx, token)
}

func (aapa *apiAuthProviderAdapter) RefreshToken(ctx context.Context, token string) (string, error) {
	return aapa.provider.RefreshToken(ctx, token)
}

// authStoreAdapter adapts store.Store to auth.Store interface
type authStoreAdapter struct {
	store *store.Store
}

func (asa *authStoreAdapter) GetUserByUsername(ctx context.Context, username string) (*auth.User, error) {
	user, err := asa.store.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}
	return &auth.User{
		ID:                 user.ID,
		Username:           user.Username,
		PasswordHash:       user.PasswordHash,
		Email:              email,
		IsAdmin:            user.IsAdmin,
		MustChangePassword: user.MustChangePassword,
	}, nil
}

func (asa *authStoreAdapter) UpdateLastLogin(ctx context.Context, userID int64) error {
	return asa.store.UpdateLastLogin(ctx, userID)
}

func (asa *authStoreAdapter) CreateSessionToken(ctx context.Context, token string, userID int64, expiresAt interface{}) error {
	// Convert interface{} to time.Time
	var expiresAtTime time.Time
	switch v := expiresAt.(type) {
	case time.Time:
		expiresAtTime = v
	default:
		return fmt.Errorf("invalid expiresAt type: %T", expiresAt)
	}
	return asa.store.CreateSessionToken(ctx, token, userID, expiresAtTime)
}

func (asa *authStoreAdapter) GetSessionToken(ctx context.Context, token string) (*auth.SessionToken, error) {
	sessionToken, err := asa.store.GetSessionToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if sessionToken == nil {
		return nil, nil
	}
	return &auth.SessionToken{
		Token:     sessionToken.Token,
		UserID:    sessionToken.UserID,
		ExpiresAt: sessionToken.ExpiresAt,
	}, nil
}

func (asa *authStoreAdapter) DeleteSessionToken(ctx context.Context, token string) error {
	return asa.store.DeleteSessionToken(ctx, token)
}

func (asa *authStoreAdapter) IsAccountLocked(ctx context.Context, username string) (bool, interface{}) {
	locked, until := asa.store.IsAccountLocked(ctx, username)
	return locked, until
}

func (asa *authStoreAdapter) RecordFailedLogin(ctx context.Context, username string) error {
	return asa.store.RecordFailedLogin(ctx, username)
}

func (asa *authStoreAdapter) ClearFailedLogins(ctx context.Context, username string) error {
	return asa.store.ClearFailedLogins(ctx, username)
}

// apiProviderManagerAdapter adapts provider.DualProviderManager to api.ProviderManager interface
type apiProviderManagerAdapter struct {
	manager interface {
		GetActiveProvider() (llm.Provider, error)
		GetLocalProvider() llm.Provider
		GetCloudProvider() llm.Provider
		IsLocalMode() bool
		GetProviderName() string
		Reload(cfg *config.Config) error
	}
}

func (apma *apiProviderManagerAdapter) GetActiveProvider() (api.LLMProvider, error) {
	provider, err := apma.manager.GetActiveProvider()
	if err != nil {
		return nil, err
	}
	// Wrap the llm.Provider in an apiProviderAdapter
	return &apiProviderAdapter{provider: provider}, nil
}

func (apma *apiProviderManagerAdapter) GetLocalProvider() api.LLMProvider {
	provider := apma.manager.GetLocalProvider()
	if provider == nil {
		return nil
	}
	return &apiProviderAdapter{provider: provider}
}

func (apma *apiProviderManagerAdapter) GetCloudProvider() api.LLMProvider {
	provider := apma.manager.GetCloudProvider()
	if provider == nil {
		return nil
	}
	return &apiProviderAdapter{provider: provider}
}

func (apma *apiProviderManagerAdapter) IsLocalMode() bool {
	return apma.manager.IsLocalMode()
}

func (apma *apiProviderManagerAdapter) GetProviderName() string {
	return apma.manager.GetProviderName()
}

func (apma *apiProviderManagerAdapter) Reload(cfg interface{}) error {
	// Convert interface{} to *config.Config
	configCfg, ok := cfg.(*config.Config)
	if !ok {
		return fmt.Errorf("invalid config type: expected *config.Config, got %T", cfg)
	}
	return apma.manager.Reload(configCfg)
}

// apiRAGEnforcerAdapter adapts rag.RAGPolicyEnforcer to api.RAGEnforcer interface
type apiRAGEnforcerAdapter struct {
	enforcer interface {
		ShouldPerformRAG() bool
		GetRAGStatus() string
		Reload(cfg interface{})
	}
}

func (area *apiRAGEnforcerAdapter) ShouldPerformRAG() bool {
	return area.enforcer.ShouldPerformRAG()
}

func (area *apiRAGEnforcerAdapter) GetRAGStatus() string {
	return area.enforcer.GetRAGStatus()
}

func (area *apiRAGEnforcerAdapter) Reload(cfg interface{}) {
	area.enforcer.Reload(cfg)
}
