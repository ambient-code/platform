package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

// StreamEvents opens a live SSE stream from the runner pod via the api-server proxy.
// The caller is responsible for closing the returned io.ReadCloser.
// Returns an error immediately if the session has no runner scheduled (404) or the runner
// is unreachable (502). The stream ends when the runner emits RUN_FINISHED/RUN_ERROR or
// the client closes the connection via ctx cancellation.
func (a *SessionAPI) StreamEvents(ctx context.Context, sessionID string) (io.ReadCloser, error) {
	rawURL := a.client.baseURL + "/api/ambient/v1/sessions/" + url.PathEscape(sessionID) + "/events"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Authorization", "Bearer "+a.client.token)
	req.Header.Set("X-Ambient-Project", a.client.project)

	resp, err := a.client.streamingClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("connect to event stream: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}
	return resp.Body, nil
}
