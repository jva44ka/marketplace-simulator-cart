package interceptors

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
)

func NewTimerInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		defer func(now time.Time) {
			slog.InfoContext(ctx, "grpc outgoing request", "method", method, "duration", time.Since(now))
		}(time.Now())
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}