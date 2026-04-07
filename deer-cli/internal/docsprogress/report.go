package docsprogress

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

// RegisterSession registers a session code with the docs-progress API (fire-and-forget).
func RegisterSession(apiURL, sessionCode string) {
	u, err := url.JoinPath(apiURL, "/v1/docs-progress/register")
	if err != nil {
		return
	}
	body, _ := json.Marshal(map[string]string{
		"session_code": sessionCode,
	})
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(u, "application/json", bytes.NewReader(body))
	if err == nil {
		_ = resp.Body.Close()
	}
}

// ReportCompletion sends a fire-and-forget completion event to the docs-progress API.
func ReportCompletion(apiURL, sessionCode string, stepIndex int) {
	u, err := url.JoinPath(apiURL, "/v1/docs-progress/complete")
	if err != nil {
		return
	}
	body, _ := json.Marshal(map[string]any{
		"session_code": sessionCode,
		"step_index":   stepIndex,
	})
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Post(u, "application/json", bytes.NewReader(body))
	if err == nil {
		_ = resp.Body.Close()
	}
}
