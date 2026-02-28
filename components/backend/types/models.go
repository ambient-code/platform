package types

// Model represents an AI model available on the platform (API-facing).
type Model struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Provider  string `json:"provider"`
	Tier      string `json:"tier"`
	IsDefault bool   `json:"isDefault"`
}

// ModelEntry represents a model entry in the manifest file (internal).
// Includes fields not exposed in the API response.
type ModelEntry struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	VertexID  string `json:"vertexId"`
	Provider  string `json:"provider"`
	Tier      string `json:"tier"`
	Available bool   `json:"available"`
}

// ModelManifest represents the top-level model manifest structure.
type ModelManifest struct {
	Version      int          `json:"version"`
	DefaultModel string       `json:"defaultModel"`
	Models       []ModelEntry `json:"models"`
}

// ListModelsResponse is the API response for the models endpoint.
type ListModelsResponse struct {
	Models       []Model `json:"models"`
	DefaultModel string  `json:"defaultModel"`
}
