// Package types defines common type definitions for AgenticSession, ProjectSettings, and RFE workflows.
package types

// Common types used across the application

type GitRepository struct {
	URL      string       `json:"url"`
	Branch   *string      `json:"branch,omitempty"`
	Provider ProviderType `json:"provider,omitempty"` // Optional: auto-detected if not specified
}

type UserContext struct {
	UserID      string   `json:"userId" binding:"required"`
	DisplayName string   `json:"displayName" binding:"required"`
	Groups      []string `json:"groups" binding:"required"`
}

type BotAccountRef struct {
	Name string `json:"name" binding:"required"`
}

type ResourceOverrides struct {
	CPU           string `json:"cpu,omitempty"`
	Memory        string `json:"memory,omitempty"`
	StorageClass  string `json:"storageClass,omitempty"`
	PriorityClass string `json:"priorityClass,omitempty"`
}

type LLMSettings struct {
	Model       string  `json:"model"`
	Temperature float64 `json:"temperature"`
	MaxTokens   int     `json:"maxTokens"`
}

type GitConfig struct {
	Repositories []GitRepository `json:"repositories,omitempty"`
}

type Paths struct {
	Workspace string `json:"workspace,omitempty"`
	Messages  string `json:"messages,omitempty"`
	Inbox     string `json:"inbox,omitempty"`
}

// Common repository browsing types (used by both GitHub and GitLab)

// Branch represents a Git branch (common format for UI)
type Branch struct {
	Name      string     `json:"name"`
	Protected bool       `json:"protected"`
	Default   bool       `json:"default,omitempty"`
	Commit    CommitInfo `json:"commit,omitempty"`
}

// CommitInfo represents basic commit information
type CommitInfo struct {
	SHA       string `json:"sha"`
	Message   string `json:"message,omitempty"`
	Author    string `json:"author,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// TreeEntry represents a file or directory in a repository
type TreeEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
	Type string `json:"type"` // "blob" (file) or "tree" (directory)
	Mode string `json:"mode,omitempty"`
	SHA  string `json:"sha,omitempty"`
	Size int    `json:"size,omitempty"`
}

// FileContent represents file contents from a repository
type FileContent struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Content  string `json:"content"`
	Encoding string `json:"encoding"` // "base64" or "utf-8"
	Size     int    `json:"size"`
	SHA      string `json:"sha,omitempty"`
}

// BoolPtr returns a pointer to the given bool value.
func BoolPtr(b bool) *bool {
	return &b
}

// StringPtr returns a pointer to the given string value.
func StringPtr(s string) *string {
	return &s
}

// IntPtr returns a pointer to the given int value.
func IntPtr(i int) *int {
	return &i
}

// PaginationParams represents common pagination request parameters
type PaginationParams struct {
	Limit    int    `form:"limit"`    // Number of items per page (default: 20, max: 100)
	Offset   int    `form:"offset"`   // Offset for offset-based pagination
	Continue string `form:"continue"` // Continuation token for k8s-style pagination
	Search   string `form:"search"`   // Search/filter term
}

// PaginatedResponse is a generic paginated response structure
type PaginatedResponse struct {
	Items      interface{} `json:"items"`
	TotalCount int         `json:"totalCount"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	HasMore    bool        `json:"hasMore"`
	Continue   string      `json:"continue,omitempty"`   // For k8s-style pagination
	NextOffset *int        `json:"nextOffset,omitempty"` // For offset-based pagination
}

// DefaultPaginationLimit is the default number of items per page
const DefaultPaginationLimit = 20

// MaxPaginationLimit is the maximum allowed items per page
const MaxPaginationLimit = 100

// DefaultBranch is the default Git branch name used when no branch is specified
const DefaultBranch = "main"

// NormalizePaginationParams ensures pagination params are within valid bounds
func NormalizePaginationParams(params *PaginationParams) {
	if params.Limit <= 0 {
		params.Limit = DefaultPaginationLimit
	}
	if params.Limit > MaxPaginationLimit {
		params.Limit = MaxPaginationLimit
	}
	if params.Offset < 0 {
		params.Offset = 0
	}
}
