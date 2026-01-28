package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SDKConfiguration represents the Claude Agent SDK configuration
type SDKConfiguration struct {
	Model                  string            `json:"model"`
	MaxTokens              int               `json:"maxTokens"`
	Temperature            float64           `json:"temperature"`
	PermissionMode         string            `json:"permissionMode"`
	AllowedTools           []string          `json:"allowedTools"`
	IncludePartialMessages bool              `json:"includePartialMessages"`
	ContinueConversation   bool              `json:"continueConversation"`
	SystemPrompt           string            `json:"systemPrompt"`
	MCPServers             map[string]MCPServerConfig `json:"mcpServers"`
}

// MCPServerConfig represents a single MCP server configuration
type MCPServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
	Enabled bool              `json:"enabled"`
}

// SDKConfigModel is the database model for SDK configuration
type SDKConfigModel struct {
	gorm.Model
	ProjectName     string `gorm:"uniqueIndex:idx_project_user"`
	UserID          string `gorm:"uniqueIndex:idx_project_user"`
	ConfigJSON      string `gorm:"type:text"`
}

// GetSDKConfiguration retrieves the current SDK configuration
func GetSDKConfiguration(c *gin.Context) {
	projectName := c.Param("project")
	userID := c.GetString("user_id") // From auth middleware

	db := c.MustGet("db").(*gorm.DB)

	var configModel SDKConfigModel
	err := db.Where("project_name = ? AND user_id = ?", projectName, userID).
		First(&configModel).Error

	if err == gorm.ErrRecordNotFound {
		// Return default configuration
		c.JSON(http.StatusOK, getDefaultConfiguration())
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration"})
		return
	}

	var config SDKConfiguration
	if err := json.Unmarshal([]byte(configModel.ConfigJSON), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse configuration"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// UpdateSDKConfiguration saves a new SDK configuration
func UpdateSDKConfiguration(c *gin.Context) {
	projectName := c.Param("project")
	userID := c.GetString("user_id")

	var config SDKConfiguration
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid configuration format"})
		return
	}

	// Validate configuration
	if err := validateSDKConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to serialize configuration"})
		return
	}

	db := c.MustGet("db").(*gorm.DB)

	configModel := SDKConfigModel{
		ProjectName: projectName,
		UserID:      userID,
		ConfigJSON:  string(configJSON),
	}

	result := db.Where("project_name = ? AND user_id = ?", projectName, userID).
		Assign(SDKConfigModel{ConfigJSON: string(configJSON)}).
		FirstOrCreate(&configModel)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save configuration"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Configuration saved successfully"})
}

// TestMCPServer tests connectivity to an MCP server
func TestMCPServer(c *gin.Context) {
	serverName := c.Param("server")

	var serverConfig MCPServerConfig
	if err := c.ShouldBindJSON(&serverConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid server configuration"})
		return
	}

	// Test MCP server connectivity
	connected := testMCPServerConnection(serverConfig)

	c.JSON(http.StatusOK, gin.H{
		"server":    serverName,
		"connected": connected,
	})
}

// GetSDKConfigForSession retrieves SDK config for a specific session
// This is called by the runner to get the active configuration
func GetSDKConfigForSession(c *gin.Context) {
	projectName := c.Param("project")
	sessionID := c.Param("session")

	db := c.MustGet("db").(*gorm.DB)

	// Get session to find owner
	var session struct {
		UserID string
	}
	err := db.Table("agentic_sessions").
		Select("user_id").
		Where("project_name = ? AND id = ?", projectName, sessionID).
		First(&session).Error

	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Session not found"})
		return
	}

	// Get user's SDK configuration
	var configModel SDKConfigModel
	err = db.Where("project_name = ? AND user_id = ?", projectName, session.UserID).
		First(&configModel).Error

	if err == gorm.ErrRecordNotFound {
		c.JSON(http.StatusOK, getDefaultConfiguration())
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve configuration"})
		return
	}

	var config SDKConfiguration
	if err := json.Unmarshal([]byte(configModel.ConfigJSON), &config); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to parse configuration"})
		return
	}

	c.JSON(http.StatusOK, config)
}

// Helper functions

func getDefaultConfiguration() SDKConfiguration {
	return SDKConfiguration{
		Model:                  "claude-sonnet-4-5@20250929",
		MaxTokens:              4096,
		Temperature:            1.0,
		PermissionMode:         "acceptEdits",
		AllowedTools:           []string{"Read", "Write", "Bash", "Glob", "Grep", "Edit", "MultiEdit", "WebSearch"},
		IncludePartialMessages: true,
		ContinueConversation:   true,
		SystemPrompt:           "",
		MCPServers:             make(map[string]MCPServerConfig),
	}
}

func validateSDKConfig(config *SDKConfiguration) error {
	// Validate model
	validModels := []string{
		"claude-opus-4-5@20251101",
		"claude-sonnet-4-5@20250929",
		"claude-haiku-4-5@20251001",
	}
	modelValid := false
	for _, m := range validModels {
		if config.Model == m {
			modelValid = true
			break
		}
	}
	if !modelValid {
		return fmt.Errorf("invalid model: %s", config.Model)
	}

	// Validate max tokens
	if config.MaxTokens < 1 || config.MaxTokens > 200000 {
		return fmt.Errorf("maxTokens must be between 1 and 200,000")
	}

	// Validate temperature
	if config.Temperature < 0 || config.Temperature > 1 {
		return fmt.Errorf("temperature must be between 0 and 1")
	}

	// Validate permission mode
	validModes := []string{"acceptEdits", "prompt", "reject"}
	modeValid := false
	for _, m := range validModes {
		if config.PermissionMode == m {
			modeValid = true
			break
		}
	}
	if !modeValid {
		return fmt.Errorf("invalid permissionMode: %s", config.PermissionMode)
	}

	// Validate at least one tool is enabled
	if len(config.AllowedTools) == 0 {
		return fmt.Errorf("at least one tool must be enabled")
	}

	// Validate tools
	validTools := map[string]bool{
		"Read": true, "Write": true, "Bash": true, "Glob": true,
		"Grep": true, "Edit": true, "MultiEdit": true, "WebSearch": true,
		"NotebookEdit": true, "WebFetch": true,
	}
	for _, tool := range config.AllowedTools {
		if !validTools[tool] {
			return fmt.Errorf("invalid tool: %s", tool)
		}
	}

	// Validate MCP servers
	for name, server := range config.MCPServers {
		if strings.TrimSpace(server.Command) == "" {
			return fmt.Errorf("MCP server '%s' has empty command", name)
		}
	}

	return nil
}

func testMCPServerConnection(config MCPServerConfig) bool {
	// TODO: Implement actual MCP server connectivity test
	// For now, just check if command is not empty
	return strings.TrimSpace(config.Command) != ""
}
