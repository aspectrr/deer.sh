package error

import (
	"log/slog"
	"net/http"

	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

// RespondError logs the actual error and returns a generic message to the client.
func RespondError(w http.ResponseWriter, status int, err error) {
	if err != nil {
		slog.Warn("api error", "status", status, "error", err.Error())
	}
	msg := http.StatusText(status)
	if msg == "" {
		msg = "unexpected error"
	}
	_ = serverJSON.RespondJSON(w, status, ErrorResponse{
		Error: msg,
		Code:  status,
	})
}

// RespondErrorMsg logs the internal error and returns a specific user-facing message.
func RespondErrorMsg(w http.ResponseWriter, status int, userMsg string, internalErr error) {
	if internalErr != nil {
		slog.Warn("api error", "status", status, "error", internalErr.Error(), "user_msg", userMsg)
	}
	_ = serverJSON.RespondJSON(w, status, ErrorResponse{
		Error: userMsg,
		Code:  status,
	})
}
