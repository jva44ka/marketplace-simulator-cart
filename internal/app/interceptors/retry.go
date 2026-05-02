package interceptors

import (
	"context"
	"fmt"
	"math"
	"math/rand/v2"
	"time"

	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewRetryInterceptor(cfg config.RetryConfig) (grpc.UnaryClientInterceptor, error) {
	initialBackoff, err := time.ParseDuration(cfg.InitialBackoff)
	if err != nil {
		return nil, fmt.Errorf("parse retry initial-backoff: %w", err)
	}

	maxBackoff, err := time.ParseDuration(cfg.MaxBackoff)
	if err != nil {
		return nil, fmt.Errorf("parse retry max-backoff: %w", err)
	}

	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		var err error
		for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
			if attempt > 0 {
				backoff := calcBackoff(attempt, initialBackoff, maxBackoff, cfg.Multiplier, cfg.JitterFactor)
				select {
				case <-time.After(backoff):
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			err = invoker(ctx, method, req, reply, cc, opts...)
			if err == nil || !isRetryable(err) {
				return err
			}
		}
		return err
	}, nil
}

func isRetryable(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}
	switch st.Code() {
	case codes.Unavailable, codes.DeadlineExceeded, codes.Internal, codes.Aborted, codes.Unknown:
		return true
	default:
		return false
	}
}

func calcBackoff(attempt int, initial, max time.Duration, multiplier, jitterFactor float64) time.Duration {
	backoff := float64(initial) * math.Pow(multiplier, float64(attempt-1))
	if backoff > float64(max) {
		backoff = float64(max)
	}
	jitter := backoff * jitterFactor * (rand.Float64()*2 - 1)
	return time.Duration(backoff + jitter)
}
