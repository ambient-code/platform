package sessions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type pushMessageRequest struct {
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
}

type sseWriter struct {
	http.ResponseWriter
	flusher http.Flusher
}

func newSSEWriter(w http.ResponseWriter) *sseWriter {
	sw := &sseWriter{ResponseWriter: w}
	for {
		if f, ok := w.(http.Flusher); ok {
			sw.flusher = f
			break
		}
		type unwrapper interface{ Unwrap() http.ResponseWriter }
		if u, ok := w.(unwrapper); ok {
			w = u.Unwrap()
			continue
		}
		break
	}
	if sw.flusher == nil {
		glog.Warning("newSSEWriter: no http.Flusher found in ResponseWriter chain; SSE pings disabled")
	}
	return sw
}

func (s *sseWriter) Flush() {
	if s.flusher != nil {
		s.flusher.Flush()
	}
}

func (s *sseWriter) Unwrap() http.ResponseWriter {
	return s.ResponseWriter
}

func (h *messageHandler) PushMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if _, err := h.session.Get(ctx, id); err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusBadRequest)
		return
	}

	var req pushMessageRequest
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.EventType == "" {
		req.EventType = "user"
	}
	if req.EventType != "user" {
		http.Error(w, "event_type must be \"user\" on this endpoint", http.StatusBadRequest)
		return
	}

	msg, err := h.msg.Push(ctx, id, req.EventType, req.Payload)
	if err != nil {
		glog.Errorf("PushMessage: session %s: %v", id, err)
		http.Error(w, "failed to push message", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if encErr := json.NewEncoder(w).Encode(msg); encErr != nil {
		glog.Errorf("PushMessage: encode response: %v", encErr)
	}
}

type messageHandler struct {
	session SessionService
	msg     MessageService
}

func NewMessageHandler(session SessionService, msg MessageService) *messageHandler {
	return &messageHandler{session: session, msg: msg}
}

func (h *messageHandler) GetMessages(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Accept") == "text/event-stream" {
		h.streamMessages(w, r, nil)
	} else {
		h.ListMessages(w, r)
	}
}

func (h *messageHandler) ListMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if _, err := h.session.Get(ctx, id); err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	afterSeq := int64(0)
	if v := r.URL.Query().Get("after_seq"); v != "" {
		if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
			afterSeq = parsed
		}
	}

	msgs, err := h.msg.AllBySessionIDAfterSeq(ctx, id, afterSeq)
	if err != nil {
		http.Error(w, "failed to load messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if encErr := json.NewEncoder(w).Encode(msgs); encErr != nil {
		glog.Errorf("ListMessages: encode response: %v", encErr)
	}
}

func (h *messageHandler) StreamTextMessages(w http.ResponseWriter, r *http.Request) {
	h.streamMessages(w, r, func(msg *SessionMessage) bool {
		return strings.HasPrefix(msg.EventType, "TEXT_MESSAGE_")
	})
}

func (h *messageHandler) streamMessages(w http.ResponseWriter, r *http.Request, filter func(*SessionMessage) bool) {
	ctx := r.Context()
	id := mux.Vars(r)["id"]

	if _, err := h.session.Get(ctx, id); err != nil {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	afterSeq := cursorFromRequest(r)

	ch, cancel := h.msg.Subscribe(ctx, id)
	defer cancel()

	existing, err := h.msg.AllBySessionIDAfterSeq(ctx, id, afterSeq)
	if err != nil {
		http.Error(w, "failed to load messages", http.StatusInternalServerError)
		return
	}

	sw := newSSEWriter(w)
	sw.Header().Set("Content-Type", "text/event-stream")
	sw.Header().Set("Cache-Control", "no-cache")
	sw.Header().Set("Connection", "keep-alive")
	sw.Header().Set("X-Accel-Buffering", "no")
	sw.WriteHeader(http.StatusOK)
	sw.Flush()

	writeEvent := func(msg *SessionMessage) bool {
		if filter != nil && !filter(msg) {
			return true
		}
		data, err := json.Marshal(msg)
		if err != nil {
			glog.Errorf("streamMessages: marshal error for session %s seq %d: %v", id, msg.Seq, err)
			return false
		}
		if _, err := fmt.Fprintf(sw, "id: %d\ndata: %s\n\n", msg.Seq, data); err != nil {
			return false
		}
		sw.Flush()
		return true
	}

	var maxReplayed int64
	for i := range existing {
		if !writeEvent(&existing[i]) {
			return
		}
		if existing[i].Seq > maxReplayed {
			maxReplayed = existing[i].Seq
		}
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := fmt.Fprintf(sw, ": ping\n\n"); err != nil {
				return
			}
			sw.Flush()
		case msg, ok := <-ch:
			if !ok {
				return
			}
			if msg.Seq <= maxReplayed {
				continue
			}
			if !writeEvent(msg) {
				return
			}
		}
	}
}

func cursorFromRequest(r *http.Request) int64 {
	if v := r.Header.Get("Last-Event-ID"); v != "" {
		if seq, err := strconv.ParseInt(v, 10, 64); err == nil {
			return seq
		}
	}
	if v := r.URL.Query().Get("after_seq"); v != "" {
		if seq, err := strconv.ParseInt(v, 10, 64); err == nil {
			return seq
		}
	}
	return 0
}
