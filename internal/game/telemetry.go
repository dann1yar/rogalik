package game

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"
)

// Telemetry sends best-effort, fire-and-forget events to the headless
// server (cmd/server). If the server is unreachable the game keeps
// running unaffected — telemetry is purely observational.
type Telemetry struct {
	endpoint string
	client   *http.Client
}

// NewTelemetry builds a client pointed at the given server base URL.
// Pass an empty string to disable telemetry entirely.
func NewTelemetry(endpoint string) *Telemetry {
	return &Telemetry{
		endpoint: endpoint,
		client:   &http.Client{Timeout: 300 * time.Millisecond},
	}
}

type eventPayload struct {
	Type  string `json:"type"`
	Level int    `json:"level"`
}

// Event fires an async, non-blocking POST to the telemetry server.
func (t *Telemetry) Event(kind string, level int) {
	if t == nil || t.endpoint == "" {
		return
	}
	go func() {
		body, _ := json.Marshal(eventPayload{Type: kind, Level: level})
		req, err := http.NewRequest(http.MethodPost, t.endpoint+"/event", bytes.NewReader(body))
		if err != nil {
			return
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := t.client.Do(req)
		if err != nil {
			return
		}
		resp.Body.Close()
	}()
}
