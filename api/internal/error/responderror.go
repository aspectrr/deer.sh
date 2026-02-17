package error

import (
	"net/http"

	serverJSON "github.com/aspectrr/fluid.sh/api/internal/json"
)

type ErrorResponse struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}

func RespondError(w http.ResponseWriter, status int, err error) {
	_ = serverJSON.RespondJSON(w, status, ErrorResponse{
		Error: err.Error(),
		Code:  status,
	})
}
