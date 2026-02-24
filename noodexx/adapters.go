package main

import (
	"context"
	"io"
	"time"

	"noodexx/internal/api"
	"noodexx/internal/ingest"
	"noodexx/internal/llm"
	"noodexx/internal/rag"
	"noodexx/internal/store"
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
