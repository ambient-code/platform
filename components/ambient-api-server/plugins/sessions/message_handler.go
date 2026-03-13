package sessions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
)

type messageHandler struct {
	session SessionService
	msg     MessageService
}

func NewMessageHandler(session SessionService, msg MessageService) *messageHandler {
	return &messageHandler{session: session, msg: msg}
}

func (h *messageHandler) StreamMessages(w http.ResponseWriter, r *http.Request) {
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

	ch, cancel := h.msg.Subscribe(ctx, id)
	defer cancel()

	existing, err := h.msg.AllBySessionIDAfterSeq(ctx, id, afterSeq)
	if err != nil {
		http.Error(w, "failed to load messages", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	flusher, canFlush := w.(http.Flusher)

	writeEvent := func(msg *SessionMessage) bool {
		data, err := json.Marshal(msg)
		if err != nil {
			glog.Errorf("StreamMessages: marshal error for session %s seq %d: %v", id, msg.Seq, err)
			return false
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return false
		}
		if canFlush {
			flusher.Flush()
		}
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

	for {
		select {
		case <-ctx.Done():
			return
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
