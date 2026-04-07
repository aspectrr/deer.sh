package agent

import (
	"context"
	"testing"
)

func TestTokenCreds_GetRequestMetadata(t *testing.T) {
	creds := tokenCreds{token: "my-secret-token", insecure: false}

	md, err := creds.GetRequestMetadata(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := "Bearer my-secret-token"
	if got := md["authorization"]; got != want {
		t.Errorf("authorization = %q, want %q", got, want)
	}
}

func TestTokenCreds_RequireTransportSecurity(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
		want     bool
	}{
		{"secure requires transport security", false, true},
		{"insecure does not require transport security", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := tokenCreds{token: "tok", insecure: tt.insecure}
			if got := creds.RequireTransportSecurity(); got != tt.want {
				t.Errorf("RequireTransportSecurity() = %v, want %v", got, tt.want)
			}
		})
	}
}
