package auth

import (
	"context"
	"fmt"
	"testing"

	"github.com/aspectrr/fluid.sh/api/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestHashToken_Deterministic(t *testing.T) {
	h1 := HashToken("my-secret-token")
	h2 := HashToken("my-secret-token")
	if h1 != h2 {
		t.Fatalf("HashToken not deterministic: %q != %q", h1, h2)
	}
}

func TestHashToken_Length(t *testing.T) {
	h := HashToken("anything")
	if len(h) != 64 {
		t.Fatalf("HashToken length = %d, want 64", len(h))
	}
}

func TestOrgIDFromContext_Empty(t *testing.T) {
	ctx := context.Background()
	if got := OrgIDFromContext(ctx); got != "" {
		t.Fatalf("OrgIDFromContext on fresh context = %q, want empty", got)
	}
}

func TestOrgIDFromContext_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), hostOrgKey{}, "org-123")
	if got := OrgIDFromContext(ctx); got != "org-123" {
		t.Fatalf("OrgIDFromContext = %q, want %q", got, "org-123")
	}
}

// --- mock gRPC server stream ---

type mockServerStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (m *mockServerStream) Context() context.Context {
	return m.ctx
}

// --- HostTokenStreamInterceptor tests ---

func TestHostTokenStreamInterceptor_MissingMetadata(t *testing.T) {
	st := &mockStore{}
	interceptor := HostTokenStreamInterceptor(st)

	// Context with no metadata at all.
	ss := &mockServerStream{ctx: context.Background()}

	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(_ any, _ grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestHostTokenStreamInterceptor_MissingAuthorization(t *testing.T) {
	st := &mockStore{}
	interceptor := HostTokenStreamInterceptor(st)

	// Metadata present but no "authorization" key.
	md := metadata.New(map[string]string{"other": "value"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &mockServerStream{ctx: ctx}

	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(_ any, _ grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestHostTokenStreamInterceptor_InvalidToken(t *testing.T) {
	st := &mockStore{
		getHostTokenByHashFn: func(_ context.Context, _ string) (*store.HostToken, error) {
			return nil, fmt.Errorf("not found")
		},
	}
	interceptor := HostTokenStreamInterceptor(st)

	md := metadata.New(map[string]string{"authorization": "Bearer bad-token"})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &mockServerStream{ctx: ctx}

	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(_ any, _ grpc.ServerStream) error {
		t.Fatal("handler should not be called")
		return nil
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if s, ok := status.FromError(err); !ok || s.Code() != codes.Unauthenticated {
		t.Fatalf("expected Unauthenticated, got %v", err)
	}
}

func TestHostTokenStreamInterceptor_ValidToken(t *testing.T) {
	rawToken := "valid-host-token"
	expectedOrgID := "org-456"

	st := &mockStore{
		getHostTokenByHashFn: func(_ context.Context, hash string) (*store.HostToken, error) {
			want := HashToken(rawToken)
			if hash != want {
				return nil, fmt.Errorf("unexpected hash")
			}
			return &store.HostToken{
				ID:    "tok-1",
				OrgID: expectedOrgID,
				Name:  "test-host",
			}, nil
		},
	}
	interceptor := HostTokenStreamInterceptor(st)

	md := metadata.New(map[string]string{"authorization": "Bearer " + rawToken})
	ctx := metadata.NewIncomingContext(context.Background(), md)
	ss := &mockServerStream{ctx: ctx}

	var handlerCalled bool
	err := interceptor(nil, ss, &grpc.StreamServerInfo{}, func(_ any, stream grpc.ServerStream) error {
		handlerCalled = true
		orgID := OrgIDFromContext(stream.Context())
		if orgID != expectedOrgID {
			t.Fatalf("OrgIDFromContext = %q, want %q", orgID, expectedOrgID)
		}
		return nil
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("handler was not called")
	}
}
