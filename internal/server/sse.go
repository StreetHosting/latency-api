package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func writeSSE(w http.ResponseWriter, flusher http.Flusher, eventType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	if eventType == "" {
		eventType = "message"
	}
	if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", eventType, data); err != nil {
		return err
	}
	flusher.Flush()
	return nil
}

func wantsMTRStream(r *http.Request) bool {
	if r.URL.Query().Get("stream") == "1" {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}
