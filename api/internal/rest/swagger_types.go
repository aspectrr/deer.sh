package rest

// swaggerError is used only for swagger documentation.
// The actual error responses are built by the error package.
type swaggerError struct {
	Error   string `json:"error"`
	Code    int    `json:"code"`
	Details string `json:"details,omitempty"`
}
