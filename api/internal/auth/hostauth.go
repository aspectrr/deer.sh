package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"github.com/aspectrr/fluid.sh/api/internal/store"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type hostOrgKey struct{}
type hostTokenIDKey struct{}

// OrgIDFromContext returns the org ID attached to a gRPC context by the
// host token auth interceptor.
func OrgIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(hostOrgKey{}).(string)
	return v
}

// TokenIDFromContext returns the host token ID attached to a gRPC context by
// the host token auth interceptor.
func TokenIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(hostTokenIDKey{}).(string)
	return v
}

// HashToken produces a SHA-256 hex digest of a raw bearer token.
func HashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// HostTokenStreamInterceptor returns a gRPC stream server interceptor that
// validates bearer tokens from host daemons. On success it attaches the
// host token's org_id to the stream context.
func HostTokenStreamInterceptor(st store.Store) grpc.StreamServerInterceptor {
	return func(
		srv any,
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		md, ok := metadata.FromIncomingContext(ss.Context())
		if !ok {
			return status.Error(codes.Unauthenticated, "missing metadata")
		}

		vals := md.Get("authorization")
		if len(vals) == 0 {
			return status.Error(codes.Unauthenticated, "missing authorization header")
		}

		raw := vals[0]
		// Strip "Bearer " prefix if present.
		if after, found := strings.CutPrefix(raw, "Bearer "); found {
			raw = after
		}

		hash := HashToken(raw)
		token, err := st.GetHostTokenByHash(ss.Context(), hash)
		if err != nil {
			return status.Error(codes.Unauthenticated, "invalid host token")
		}

		// Attach org_id and token_id to stream context.
		ctx := context.WithValue(ss.Context(), hostOrgKey{}, token.OrgID)
		ctx = context.WithValue(ctx, hostTokenIDKey{}, token.ID)
		wrapped := &wrappedStream{ServerStream: ss, ctx: ctx}
		return handler(srv, wrapped)
	}
}

// WithTokenID returns a context carrying the given token ID.
// Exported for use in tests that bypass the interceptor.
func WithTokenID(ctx context.Context, tokenID string) context.Context {
	return context.WithValue(ctx, hostTokenIDKey{}, tokenID)
}

// WithOrgID returns a context carrying the given org ID.
// Exported for use in tests that bypass the interceptor.
func WithOrgID(ctx context.Context, orgID string) context.Context {
	return context.WithValue(ctx, hostOrgKey{}, orgID)
}

// wrappedStream overrides Context() to return an enriched context.
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}
