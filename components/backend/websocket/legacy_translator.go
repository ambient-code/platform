package websocket

import (
	"ambient-code-backend/types"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"os"
	"time"
)

// MigrateLegacySessionToAGUI converts old message format to AG-UI events
// Creates a MESSAGES_SNAPSHOT from legacy messages and persists it
func MigrateLegacySessionToAGUI(sessionID string) error {
	// Check if session has legacy messages
	legacyPath := StateBaseDir + "/sessions/" + sessionID + "/messages.json"
	data, err := os.ReadFile(legacyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No legacy file, nothing to migrate
			return nil
		}
		return err
	}

	log.Printf("LegacyMigration: Found legacy messages.json for %s, converting to AG-UI", sessionID)

	var legacyData struct {
		Messages  []map[string]interface{} `json:"messages"`
		SessionID string                   `json:"sessionId"`
	}

	if err := json.Unmarshal(data, &legacyData); err != nil {
		log.Printf("LegacyMigration: Failed to parse legacy messages: %v", err)
		return err
	}

	// Convert to AG-UI Message format
	messages := make([]types.Message, 0)
	
	for _, legacyMsg := range legacyData.Messages {
		msgType, _ := legacyMsg["type"].(string)
		payload, _ := legacyMsg["payload"].(map[string]interface{})
		
		switch msgType {
		case "user_message":
			content, _ := payload["content"].(string)
			messages = append(messages, types.Message{
				ID:      generateEventID(),
				Role:    types.RoleUser,
				Content: content,
			})

		case "agent.message":
			// Check if it's a text message
			if content, ok := payload["content"].(map[string]interface{}); ok {
				textType, _ := content["type"].(string)
				if textType == "text_block" {
					text, _ := content["text"].(string)
					messages = append(messages, types.Message{
						ID:      generateEventID(),
						Role:    types.RoleAssistant,
						Content: text,
					})
				}
			}
			// Tool calls will be reconstructed from tool_result pairs
			
		// system.message, agent.running, agent.waiting are not chat messages, skip
		}
	}

	if len(messages) == 0 {
		log.Printf("LegacyMigration: No chat messages found in legacy file")
		return nil
	}

	log.Printf("LegacyMigration: Converted %d legacy messages to AG-UI format", len(messages))

	// Create MESSAGES_SNAPSHOT event and persist it
	snapshot := map[string]interface{}{
		"type":      types.EventTypeMessagesSnapshot,
		"threadId":  sessionID,
		"runId":     "legacy-migration",
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"messages":  messages,
	}

	// Persist to agui-events.jsonl
	persistAGUIEventMap(sessionID, "legacy-migration", snapshot)

	log.Printf("LegacyMigration: Persisted MESSAGES_SNAPSHOT with %d messages", len(messages))

	// Rename legacy file to indicate it's been migrated
	migratedPath := legacyPath + ".migrated"
	if err := os.Rename(legacyPath, migratedPath); err != nil {
		log.Printf("LegacyMigration: Warning - failed to rename legacy file: %v", err)
	}

	return nil
}

// generateEventID creates a random ID for events
func generateEventID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

