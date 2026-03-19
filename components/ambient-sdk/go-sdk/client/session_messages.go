package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func (a *SessionAPI) PushMessage(ctx context.Context, sessionID, payload string) (*types.SessionMessage, error) {
	push := struct {
		EventType string `json:"event_type"`
		Payload   string `json:"payload"`
	}{EventType: "user", Payload: payload}
	body, err := json.Marshal(push)
	if err != nil {
		return nil, fmt.Errorf("marshal message: %w", err)
	}
	var result types.SessionMessage
	if err := a.client.do(ctx, http.MethodPost, "/sessions/"+url.PathEscape(sessionID)+"/messages", body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *SessionAPI) ListMessages(ctx context.Context, sessionID string, afterSeq int) ([]types.SessionMessage, error) {
	path := fmt.Sprintf("/sessions/%s/messages?after_seq=%d", url.PathEscape(sessionID), afterSeq)
	var result []types.SessionMessage
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

