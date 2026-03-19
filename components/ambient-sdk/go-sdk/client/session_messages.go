package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

const (
	sseInitialBackoff = 1 * time.Second
	sseMaxBackoff     = 30 * time.Second
	sseScannerBufSize = 1 << 20
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

func (a *SessionAPI) ListMessages(ctx context.Context, sessionID string, afterSeq int64) ([]types.SessionMessage, error) {
	path := fmt.Sprintf("/sessions/%s/messages?after_seq=%d", url.PathEscape(sessionID), afterSeq)
	var result []types.SessionMessage
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (a *SessionAPI) SendAgUI(ctx context.Context, sessionID string, payload string) (*types.SessionMessage, error) {
	push := &types.SessionMessagePush{
		EventType: "user",
		Payload:   payload,
	}
	body, err := json.Marshal(push)
	if err != nil {
		return nil, fmt.Errorf("marshal ag_ui turn: %w", err)
	}
	var result types.SessionMessage
	if err := a.client.do(ctx, http.MethodPost, "/sessions/"+url.PathEscape(sessionID)+"/ag_ui", body, http.StatusCreated, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (a *SessionAPI) StreamAgUI(ctx context.Context, sessionID string, afterSeq int64) (<-chan types.SessionMessage, <-chan error) {
	return a.streamSSE(ctx, sessionID, "ag_ui", afterSeq)
}

// WatchMessages streams session messages from afterSeq onward via SSE.
// Returns a channel of messages, a stop function, and any immediate connection error.
// Call stop() to cancel the stream and release resources.
func (a *SessionAPI) WatchMessages(ctx context.Context, sessionID string, afterSeq int64) (<-chan *types.SessionMessage, func(), error) {
	watchCtx, cancel := context.WithCancel(ctx)
	msgs := make(chan *types.SessionMessage, 64)

	go func() {
		defer close(msgs)

		lastSeq := afterSeq
		backoff := sseInitialBackoff

		for {
			if watchCtx.Err() != nil {
				return
			}

			plain := make(chan types.SessionMessage, 64)
			done := make(chan struct{})
			go func() {
				defer close(done)
				for m := range plain {
					mc := m
					select {
					case msgs <- &mc:
					case <-watchCtx.Done():
						return
					}
				}
			}()

			err := a.consumeSSE(watchCtx, sessionID, "messages", lastSeq, plain, func(seq int64) {
				lastSeq = seq
			})
			close(plain)
			<-done

			if watchCtx.Err() != nil {
				return
			}

			if err != nil {
				a.client.logger.Debug("sse stream error, will reconnect",
					"session_id", sessionID,
					"after_seq", lastSeq,
					"backoff", backoff,
					"err", err,
				)
			}

			select {
			case <-watchCtx.Done():
				return
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > sseMaxBackoff {
				backoff = sseMaxBackoff
			}
		}
	}()

	return msgs, cancel, nil
}

// StreamMessages streams session messages. Deprecated: use WatchMessages instead.
func (a *SessionAPI) StreamMessages(ctx context.Context, sessionID string, afterSeq int64) (<-chan types.SessionMessage, <-chan error) {
	return a.streamSSE(ctx, sessionID, "messages", afterSeq)
}

func (a *SessionAPI) streamSSE(ctx context.Context, sessionID, endpoint string, afterSeq int64) (<-chan types.SessionMessage, <-chan error) {
	msgs := make(chan types.SessionMessage, 64)
	errs := make(chan error, 1)

	go func() {
		defer close(msgs)
		defer close(errs)

		lastSeq := afterSeq
		backoff := sseInitialBackoff

		for {
			if ctx.Err() != nil {
				return
			}

			err := a.consumeSSE(ctx, sessionID, endpoint, lastSeq, msgs, func(seq int64) {
				lastSeq = seq
			})

			if ctx.Err() != nil {
				return
			}

			if err != nil {
				a.client.logger.Debug("sse stream error, will reconnect",
					"endpoint", endpoint,
					"session_id", sessionID,
					"after_seq", lastSeq,
					"backoff", backoff,
					"err", err,
				)
			}

			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}

			backoff *= 2
			if backoff > sseMaxBackoff {
				backoff = sseMaxBackoff
			}
		}
	}()

	return msgs, errs
}

var sseHTTPClient = &http.Client{
	Timeout:   0,
	Transport: &http.Transport{DisableCompression: true},
}

func (a *SessionAPI) consumeSSE(
	ctx context.Context,
	sessionID, endpoint string,
	afterSeq int64,
	msgs chan<- types.SessionMessage,
	onMsg func(seq int64),
) error {
	rawURL := fmt.Sprintf("%s/api/ambient/v1/sessions/%s/%s?after_seq=%d",
		strings.TrimRight(a.client.baseURL, "/"),
		url.PathEscape(sessionID),
		endpoint,
		afterSeq,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	if a.client.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.client.token)
	}
	if a.client.project != "" {
		req.Header.Set("X-Ambient-Project", a.client.project)
	}

	resp, err := sseHTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, sseScannerBufSize), sseScannerBufSize)

	var dataBuf strings.Builder

	for scanner.Scan() {
		if ctx.Err() != nil {
			return nil
		}

		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "data: "):
			if dataBuf.Len() > 0 {
				dataBuf.WriteByte('\n')
			}
			dataBuf.WriteString(strings.TrimPrefix(line, "data: "))

		case line == "":
			if dataBuf.Len() == 0 {
				continue
			}
			data := dataBuf.String()
			dataBuf.Reset()

			var msg types.SessionMessage
			if err := json.Unmarshal([]byte(data), &msg); err != nil {
				continue
			}

			select {
			case msgs <- msg:
				onMsg(msg.Seq)
			case <-ctx.Done():
				return nil
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scanner: %w", err)
	}
	return nil
}
