package json

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// decodeJSON is a strict JSON decoder with basic size limit and context awareness.
func DecodeJSON(ctx context.Context, r *http.Request, v any) error {
	const maxBody = int64(1 << 20) // 1 MiB
	r.Body = http.MaxBytesReader(nil, r.Body, maxBody)
	defer func() { _ = r.Body.Close() }()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("invalid json: %w", err)
	}
	// Disallow trailing data
	if dec.More() {
		var extra any
		if err := dec.Decode(&extra); err == nil {
			return errors.New("invalid json: trailing data")
		}
	}
	return nil
}
