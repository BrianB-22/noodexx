# Requirements Document

## Introduction

This feature enhances Noodexx's AI provider configuration by enabling dual provider setup (local and cloud) with granular privacy controls. Users can seamlessly switch between local Ollama and cloud AI providers via a privacy toggle, while administrators control whether document context (RAG) is shared with cloud providers through a configurable policy setting.

## Glossary

- **Noodexx**: The AI-powered knowledge base application with RAG capabilities
- **Privacy_Toggle**: The UI control in the sidebar that switches between local and cloud AI modes
- **Local_AI_Provider**: The Ollama provider running on the user's local machine
- **Cloud_AI_Provider**: Remote AI providers (OpenAI or Anthropic) accessed via API
- **RAG**: Retrieval Augmented Generation - the process of searching local documents and sending relevant snippets as context to the LLM
- **RAG_Policy**: Administrator-configured setting that controls whether RAG is performed when using cloud AI
- **Settings_Page**: The application configuration interface where providers are configured
- **Config_Store**: The persistent storage mechanism (config.json) for application settings
- **Admin_User**: A user with administrative privileges who can configure system-wide policies
- **Query_Context**: The document snippets retrieved via RAG that are sent to the LLM along with the user's query

## Requirements

### Requirement 1: Dual Provider Configuration

**User Story:** As an administrator, I want to configure both a local AI provider and a cloud AI provider simultaneously, so that users can switch between them without reconfiguring settings.

#### Acceptance Criteria

1. THE Settings_Page SHALL display a separate configuration section for Local_AI_Provider
2. THE Settings_Page SHALL display a separate configuration section for Cloud_AI_Provider
3. WHEN an administrator saves Local_AI_Provider configuration, THE Config_Store SHALL persist the local provider settings independently
4. WHEN an administrator saves Cloud_AI_Provider configuration, THE Config_Store SHALL persist the cloud provider settings independently
5. THE Noodexx SHALL allow both provider configurations to exist simultaneously in Config_Store

### Requirement 2: Privacy Toggle Provider Switching

**User Story:** As a user, I want to switch between local and cloud AI providers with a single toggle, so that I can quickly control my data privacy without navigating to settings.

#### Acceptance Criteria

1. WHEN Privacy_Toggle is set to "Local AI" mode, THE Noodexx SHALL route all LLM requests to Local_AI_Provider
2. WHEN Privacy_Toggle is set to "Cloud AI" mode, THE Noodexx SHALL route all LLM requests to Cloud_AI_Provider
3. WHEN a user changes Privacy_Toggle state, THE Noodexx SHALL switch providers within 1 second without requiring application restart
4. THE Config_Store SHALL persist the current Privacy_Toggle state
5. WHEN Noodexx starts, THE Noodexx SHALL restore the Privacy_Toggle to its last saved state

### Requirement 3: Cloud AI RAG Policy Configuration

**User Story:** As an administrator, I want to control whether document context is sent to cloud AI providers, so that I can enforce organizational privacy policies.

#### Acceptance Criteria

1. THE Settings_Page SHALL display a "Cloud AI RAG Policy" configuration option accessible only to Admin_User
2. THE Settings_Page SHALL offer exactly two policy options: "No RAG with Cloud AI" and "Allow RAG with Cloud AI"
3. WHEN Admin_User selects "No RAG with Cloud AI", THE Config_Store SHALL persist this policy setting
4. WHEN Admin_User selects "Allow RAG with Cloud AI", THE Config_Store SHALL persist this policy setting
5. THE Settings_Page SHALL display explanatory text describing the privacy implications of each policy option

### Requirement 4: RAG Behavior with Local AI

**User Story:** As a user, I want RAG to always work with local AI, so that I can access my full knowledge base without privacy concerns.

#### Acceptance Criteria

1. WHILE Privacy_Toggle is set to "Local AI" mode, THE Noodexx SHALL perform RAG search for every user query
2. WHILE Privacy_Toggle is set to "Local AI" mode, THE Noodexx SHALL send Query_Context to Local_AI_Provider
3. WHILE Privacy_Toggle is set to "Local AI" mode, THE RAG_Policy SHALL have no effect on RAG behavior

### Requirement 5: RAG Behavior with Cloud AI - No RAG Policy

**User Story:** As a user, I want to use cloud AI without sharing my documents, so that I can benefit from powerful cloud models while maintaining maximum privacy.

#### Acceptance Criteria

1. WHILE Privacy_Toggle is set to "Cloud AI" mode AND RAG_Policy is "No RAG with Cloud AI", THE Noodexx SHALL NOT perform RAG search
2. WHILE Privacy_Toggle is set to "Cloud AI" mode AND RAG_Policy is "No RAG with Cloud AI", THE Noodexx SHALL send only the user query to Cloud_AI_Provider without Query_Context
3. WHILE Privacy_Toggle is set to "Cloud AI" mode AND RAG_Policy is "No RAG with Cloud AI", THE Noodexx SHALL display a visual indicator that RAG is disabled

### Requirement 6: RAG Behavior with Cloud AI - Allow RAG Policy

**User Story:** As a user, I want to use cloud AI with my full knowledge base, so that I can get the most accurate and contextual responses.

#### Acceptance Criteria

1. WHILE Privacy_Toggle is set to "Cloud AI" mode AND RAG_Policy is "Allow RAG with Cloud AI", THE Noodexx SHALL perform RAG search for every user query
2. WHILE Privacy_Toggle is set to "Cloud AI" mode AND RAG_Policy is "Allow RAG with Cloud AI", THE Noodexx SHALL send Query_Context to Cloud_AI_Provider
3. WHILE Privacy_Toggle is set to "Cloud AI" mode AND RAG_Policy is "Allow RAG with Cloud AI", THE Noodexx SHALL display a visual indicator that RAG is enabled and document snippets are being shared

### Requirement 7: Active Provider Indication

**User Story:** As a user, I want to see which AI provider is currently active, so that I know where my queries are being sent.

#### Acceptance Criteria

1. WHILE Privacy_Toggle is set to "Local AI" mode, THE Noodexx SHALL display "Local AI" or the Local_AI_Provider name in the user interface
2. WHILE Privacy_Toggle is set to "Cloud AI" mode, THE Noodexx SHALL display "Cloud AI" or the Cloud_AI_Provider name in the user interface
3. THE Noodexx SHALL update the active provider indication within 500 milliseconds of Privacy_Toggle state change

### Requirement 8: Configuration Validation

**User Story:** As a user, I want to be notified if a provider is not properly configured, so that I understand why my queries are failing.

#### Acceptance Criteria

1. WHEN Privacy_Toggle is set to "Local AI" mode AND Local_AI_Provider is not configured, THE Noodexx SHALL display an error message directing the user to Settings_Page
2. WHEN Privacy_Toggle is set to "Cloud AI" mode AND Cloud_AI_Provider is not configured, THE Noodexx SHALL display an error message directing the user to Settings_Page
3. IF a user attempts to send a query with an unconfigured provider, THEN THE Noodexx SHALL prevent the query and display a configuration error message

### Requirement 9: Settings Persistence and Recovery

**User Story:** As a user, I want my provider configurations and privacy settings to persist across application restarts, so that I don't need to reconfigure them each time.

#### Acceptance Criteria

1. WHEN Noodexx shuts down, THE Config_Store SHALL retain all Local_AI_Provider configuration values
2. WHEN Noodexx shuts down, THE Config_Store SHALL retain all Cloud_AI_Provider configuration values
3. WHEN Noodexx shuts down, THE Config_Store SHALL retain the RAG_Policy setting
4. WHEN Noodexx shuts down, THE Config_Store SHALL retain the Privacy_Toggle state
5. WHEN Noodexx starts, THE Noodexx SHALL load all persisted settings from Config_Store within 2 seconds

### Requirement 10: Settings Page User Experience

**User Story:** As an administrator, I want clear and organized provider configuration sections, so that I can easily understand and manage the dual provider setup.

#### Acceptance Criteria

1. THE Settings_Page SHALL visually separate Local_AI_Provider configuration from Cloud_AI_Provider configuration
2. THE Settings_Page SHALL display the RAG_Policy setting in a prominent location with clear privacy implications
3. THE Settings_Page SHALL display help text explaining the difference between local and cloud provider modes
4. WHEN Admin_User saves settings, THE Settings_Page SHALL display a confirmation message within 1 second
5. IF settings validation fails, THEN THE Settings_Page SHALL display specific error messages indicating which fields are invalid
