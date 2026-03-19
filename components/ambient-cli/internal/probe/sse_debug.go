package probe

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

var rawSSEClient = &http.Client{
	Timeout:   0,
	Transport: &http.Transport{DisableCompression: true},
}

func stepSendAndStream(ctx context.Context, s *State) error {
	streamCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()

	rawURL := fmt.Sprintf("%s/api/ambient/v1/sessions/%s/ag_ui?after_seq=0",
		strings.TrimRight(s.BaseURL, "/"), s.SessionID)

	s.Log("   GET %s/api/ambient/v1/sessions/%s/ag_ui (raw SSE)…", s.BaseURL, s.SessionID)

	req, err := http.NewRequestWithContext(streamCtx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("build SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("X-Ambient-Project", s.Project)

	s.Log("   dialing…")
	resp, err := rawSSEClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE connect: %w", err)
	}
	defer resp.Body.Close()

	s.Log("   HTTP %d %s", resp.StatusCode, resp.Status)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("SSE bad status: %d", resp.StatusCode)
	}

	msgCh := make(chan map[string]interface{}, 64)
	scanErr := make(chan error, 1)

	go func() {
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 1<<20), 1<<20)
		var dataBuf strings.Builder
		lineCount := 0
		for scanner.Scan() {
			line := scanner.Text()
			lineCount++
			s.Log("   line %d: %q", lineCount, firstN(line, 120))
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
				var outer map[string]interface{}
				if err := json.Unmarshal([]byte(data), &outer); err != nil {
					s.Log("   json err: %v  data=%q", err, firstN(data, 80))
					continue
				}
				select {
				case msgCh <- outer:
				case <-streamCtx.Done():
					scanErr <- streamCtx.Err()
					return
				}
			}
		}
		if err := scanner.Err(); err != nil {
			scanErr <- err
		} else {
			close(scanErr)
		}
	}()

	waitRF := func(label string) error {
		s.Log("   waiting for RUN_FINISHED (%s)…", label)
		for {
			select {
			case <-streamCtx.Done():
				return fmt.Errorf("timeout waiting for RUN_FINISHED (%s): %w", label, streamCtx.Err())
			case err, ok := <-scanErr:
				if ok && err != nil {
					return fmt.Errorf("scan error (%s): %w", label, err)
				}
				return fmt.Errorf("stream closed before RUN_FINISHED (%s)", label)
			case msg := <-msgCh:
				et, _ := msg["event_type"].(string)
				seq, _ := msg["seq"].(float64)
				s.Log("   msg event_type=%q seq=%v", et, seq)
				switch et {
				case "RUN_FINISHED":
					s.Log("   RUN_FINISHED (%s)", label)
					return nil
				case "TEXT_MESSAGE_CONTENT":
					if delta, _ := msg["delta"].(string); delta != "" {
						s.Log("   … %s", firstN(delta, 80))
					}
				}
			}
		}
	}

	if err := waitRF("ignition"); err != nil {
		return err
	}

	helloMsg := "Hello! Please introduce yourself in one sentence."
	s.Log("   POST /sessions/%s/ag_ui  payload=%q", s.SessionID, helloMsg)
	if _, err := s.Client.Sessions().SendAgUI(streamCtx, s.SessionID, helloMsg); err != nil {
		return fmt.Errorf("send message: %w", err)
	}

	return waitRF("response")
}
