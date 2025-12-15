package types

import (
	"fmt"
	"strings"
)

// AgenticSession represents the structure of our custom resource
type AgenticSession struct {
	APIVersion string                 `json:"apiVersion"`
	Kind       string                 `json:"kind"`
	Metadata   map[string]interface{} `json:"metadata"`
	Spec       AgenticSessionSpec     `json:"spec"`
	Status     *AgenticSessionStatus  `json:"status,omitempty"`
}

type AgenticSessionSpec struct {
	InitialPrompt        string             `json:"initialPrompt,omitempty"`
	Interactive          bool               `json:"interactive,omitempty"`
	DisplayName          string             `json:"displayName"`
	LLMSettings          LLMSettings        `json:"llmSettings"`
	Timeout              int                `json:"timeout"`
	UserContext          *UserContext       `json:"userContext,omitempty"`
	BotAccount           *BotAccountRef     `json:"botAccount,omitempty"`
	ResourceOverrides    *ResourceOverrides `json:"resourceOverrides,omitempty"`
	EnvironmentVariables map[string]string  `json:"environmentVariables,omitempty"`
	Project              string             `json:"project,omitempty"`
	// Multi-repo support
	Repos []SimpleRepo `json:"repos,omitempty"`
	// Active workflow for dynamic workflow switching
	ActiveWorkflow *WorkflowSelection `json:"activeWorkflow,omitempty"`
}

// SimpleRepo represents a repository configuration with support for both
// legacy (url/branch) and new (input/output/autoPush) formats
type SimpleRepo struct {
	// New structure (preferred)
	Input    *RepoLocation `json:"input,omitempty"`
	Output   *RepoLocation `json:"output,omitempty"`
	AutoPush *bool         `json:"autoPush,omitempty"`

	// Legacy structure (deprecated, for backwards compatibility)
	URL    string  `json:"url,omitempty"`
	Branch *string `json:"branch,omitempty"`
}

// RepoLocation represents a git repository location (input source or output target)
type RepoLocation struct {
	URL    string  `json:"url"`
	Branch *string `json:"branch,omitempty"`
}

type AgenticSessionStatus struct {
	ObservedGeneration int64               `json:"observedGeneration,omitempty"`
	Phase              string              `json:"phase,omitempty"`
	StartTime          *string             `json:"startTime,omitempty"`
	CompletionTime     *string             `json:"completionTime,omitempty"`
	ReconciledRepos    []ReconciledRepo    `json:"reconciledRepos,omitempty"`
	ReconciledWorkflow *ReconciledWorkflow `json:"reconciledWorkflow,omitempty"`
	SDKSessionID       string              `json:"sdkSessionId,omitempty"`
	SDKRestartCount    int                 `json:"sdkRestartCount,omitempty"`
	Conditions         []Condition         `json:"conditions,omitempty"`
}

type CreateAgenticSessionRequest struct {
	InitialPrompt   string       `json:"initialPrompt,omitempty"`
	DisplayName     string       `json:"displayName,omitempty"`
	LLMSettings     *LLMSettings `json:"llmSettings,omitempty"`
	Timeout         *int         `json:"timeout,omitempty"`
	Interactive     *bool        `json:"interactive,omitempty"`
	ParentSessionID string       `json:"parent_session_id,omitempty"`
	// Multi-repo support
	Repos                []SimpleRepo      `json:"repos,omitempty"`
	AutoPushOnComplete   *bool             `json:"autoPushOnComplete,omitempty"`
	UserContext          *UserContext      `json:"userContext,omitempty"`
	EnvironmentVariables map[string]string `json:"environmentVariables,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
}

type CloneSessionRequest struct {
	TargetProject  string `json:"targetProject" binding:"required"`
	NewSessionName string `json:"newSessionName" binding:"required"`
}

type UpdateAgenticSessionRequest struct {
	InitialPrompt *string      `json:"initialPrompt,omitempty"`
	DisplayName   *string      `json:"displayName,omitempty"`
	Timeout       *int         `json:"timeout,omitempty"`
	LLMSettings   *LLMSettings `json:"llmSettings,omitempty"`
}

type CloneAgenticSessionRequest struct {
	TargetProject     string `json:"targetProject,omitempty"`
	TargetSessionName string `json:"targetSessionName,omitempty"`
	DisplayName       string `json:"displayName,omitempty"`
	InitialPrompt     string `json:"initialPrompt,omitempty"`
}

// WorkflowSelection represents a workflow to load into the session
type WorkflowSelection struct {
	GitURL string `json:"gitUrl" binding:"required"`
	Branch string `json:"branch,omitempty"`
	Path   string `json:"path,omitempty"`
}

// ReconciledRepo captures reconciliation state for a repository
type ReconciledRepo struct {
	URL      string  `json:"url"`
	Branch   string  `json:"branch"`
	Name     string  `json:"name,omitempty"`
	Status   string  `json:"status,omitempty"`
	ClonedAt *string `json:"clonedAt,omitempty"`
}

// ReconciledWorkflow captures reconciliation state for the active workflow
type ReconciledWorkflow struct {
	GitURL    string  `json:"gitUrl"`
	Branch    string  `json:"branch"`
	Path      string  `json:"path,omitempty"`
	Status    string  `json:"status,omitempty"`
	AppliedAt *string `json:"appliedAt,omitempty"`
}

// Condition mirrors metav1.Condition for API transport
type Condition struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	Reason             string `json:"reason,omitempty"`
	Message            string `json:"message,omitempty"`
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
}

// NormalizeRepo converts a legacy repo format to the new input/output structure.
// If the repo already uses the new format, it returns the repo as-is.
// Legacy: {url: "...", branch: "..."} -> New: {input: {url: "...", branch: "..."}, autoPush: sessionDefaultAutoPush}
// The autoPush field is set to the session's default value (sessionDefaultAutoPush parameter).
// Returns an error if the repo has an empty URL.
func (r *SimpleRepo) NormalizeRepo(sessionDefaultAutoPush bool) (SimpleRepo, error) {
	// If already using new format, validate and return as-is
	if r.Input != nil {
		if strings.TrimSpace(r.Input.URL) == "" {
			return SimpleRepo{}, fmt.Errorf("cannot normalize repo with empty input.url")
		}

		// Validate that output differs from input (if output is specified)
		if r.Output != nil {
			inputURL := strings.TrimSpace(r.Input.URL)
			outputURL := strings.TrimSpace(r.Output.URL)
			inputBranch := ""
			outputBranch := ""
			if r.Input.Branch != nil {
				inputBranch = strings.TrimSpace(*r.Input.Branch)
			}
			if r.Output.Branch != nil {
				outputBranch = strings.TrimSpace(*r.Output.Branch)
			}

			// Output must differ from input in either URL or branch
			if inputURL == outputURL && inputBranch == outputBranch {
				return SimpleRepo{}, fmt.Errorf("output repository must differ from input (different URL or branch required)")
			}
		}

		return *r, nil
	}

	// Validate legacy format before normalizing
	if strings.TrimSpace(r.URL) == "" {
		return SimpleRepo{}, fmt.Errorf("cannot normalize repo with empty url")
	}

	// Convert legacy format to new format
	normalized := SimpleRepo{
		Input: &RepoLocation{
			URL:    r.URL,
			Branch: r.Branch,
		},
		AutoPush: BoolPtr(sessionDefaultAutoPush),
	}

	return normalized, nil
}

// ToMapForCR converts SimpleRepo to a map suitable for CustomResource spec.repos[]
func (r *SimpleRepo) ToMapForCR() map[string]interface{} {
	m := make(map[string]interface{})

	// Use new format if Input is defined
	if r.Input != nil {
		inputMap := map[string]interface{}{
			"url": r.Input.URL,
		}
		if r.Input.Branch != nil {
			inputMap["branch"] = *r.Input.Branch
		}
		m["input"] = inputMap

		// Add output if defined
		if r.Output != nil {
			outputMap := map[string]interface{}{
				"url": r.Output.URL,
			}
			if r.Output.Branch != nil {
				outputMap["branch"] = *r.Output.Branch
			}
			m["output"] = outputMap
		}

		// Add autoPush flag
		if r.AutoPush != nil {
			m["autoPush"] = *r.AutoPush
		}
	} else {
		// Legacy format - preserve for backward compatibility with un-normalized repos
		// This path should only be reached for repos that haven't been normalized yet
		// (e.g., when reading existing CRs created before the new format was introduced)
		m["url"] = r.URL
		if r.Branch != nil {
			m["branch"] = *r.Branch
		}
	}

	return m
}
