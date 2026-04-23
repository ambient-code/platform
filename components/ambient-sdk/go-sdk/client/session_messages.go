package client

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
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

func (a *SessionAPI) ListMessages(ctx context.Context, sessionID string, afterSeq int) ([]types.SessionMessage, error) {
	path := fmt.Sprintf("/sessions/%s/messages?after_seq=%d", url.PathEscape(sessionID), afterSeq)
	var result []types.SessionMessage
	if err := a.client.do(ctx, http.MethodGet, path, nil, http.StatusOK, &result); err != nil {
		return nil, err
	}
	return result, nil
}

// WatchMessages streams session messages from afterSeq onward via SSE.
// Returns a channel of messages, a stop function, and any immediate connection error.
// Call stop() to cancel the stream and release resources.
func (a *SessionAPI) WatchMessages(ctx context.Context, sessionID string, afterSeq int) (<-chan *types.SessionMessage, func(), error) {
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

			err := a.consumeSSE(watchCtx, sessionID, "messages", lastSeq, plain, func(seq int) {
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

func (a *SessionAPI) consumeSSE(
	ctx context.Context,
	sessionID, endpoint string,
	afterSeq int,
	msgs chan<- types.SessionMessage,
	onMsg func(seq int),
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

	resp, err := a.client.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

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
