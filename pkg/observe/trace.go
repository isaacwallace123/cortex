// Package observe provides distributed tracing primitives for Cortex services.
// Trace IDs are propagated through gRPC metadata under the "x-trace-id" key.
package observe

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const TraceHeader = "x-trace-id"

type traceKey struct{}

// NewID returns a new random trace ID (UUID v4).
func NewID() string {
	return uuid.NewString()
}

// FromContext retrieves the trace ID stored in ctx, or "" if absent.
func FromContext(ctx context.Context) string {
	v, _ := ctx.Value(traceKey{}).(string)
	return v
}

// WithID returns a copy of ctx carrying the given trace ID.
func WithID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, traceKey{}, id)
}

// UnaryServerInterceptor returns a gRPC server interceptor that extracts the
// trace ID from incoming metadata (or generates a new one) and stores it in ctx.
func UnaryServerInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		id := ""
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(TraceHeader); len(vals) > 0 {
				id = vals[0]
			}
		}
		if id == "" {
			id = NewID()
		}
		return handler(WithID(ctx, id), req)
	}
}

// UnaryClientInterceptor returns a gRPC client interceptor that injects the
// trace ID from ctx into outgoing metadata so downstream services receive it.
func UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		if id := FromContext(ctx); id != "" {
			ctx = metadata.AppendToOutgoingContext(ctx, TraceHeader, id)
		}
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
