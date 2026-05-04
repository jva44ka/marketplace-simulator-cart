package circuitbreaker

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
	"github.com/sony/gobreaker/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Executor struct {
	cb   atomic.Pointer[gobreaker.CircuitBreaker[any]]
	name string
}

func NewExecutor(cfg config.CircuitBreakerConfig, name string) (*Executor, error) {
	cb, err := buildCircuitBreaker(cfg, name)
	if err != nil {
		return nil, err
	}

	e := &Executor{name: name}
	e.cb.Store(cb)
	return e, nil
}

// Update пересоздаёт CircuitBreaker с новыми настройками атомарно.
// Вызывается при изменении конфига в etcd.
func (e *Executor) Update(cfg config.CircuitBreakerConfig) {
	cb, err := buildCircuitBreaker(cfg, e.name)
	if err != nil {
		slog.Error("circuitbreaker: failed to apply new config", "name", e.name, "err", err)
		return
	}
	e.cb.Store(cb)
	slog.Info("circuitbreaker: config updated", "name", e.name)
}

func (e *Executor) Execute(fn func() (any, error)) (any, error) {
	return e.cb.Load().Execute(fn)
}

func (e *Executor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		_, err := e.cb.Load().Execute(func() (any, error) {
			return nil, invoker(ctx, method, req, reply, cc, opts...)
		})
		return err
	}
}

func buildCircuitBreaker(cfg config.CircuitBreakerConfig, name string) (*gobreaker.CircuitBreaker[any], error) {
	interval, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		return nil, fmt.Errorf("parse circuit-breaker interval: %w", err)
	}

	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("parse circuit-breaker timeout: %w", err)
	}

	settings := gobreaker.Settings{
		Name:        name,
		MaxRequests: cfg.HalfOpenRequests,
		Interval:    interval,
		Timeout:     timeout,
		IsSuccessful: func(err error) bool {
			if err == nil {
				return true
			}
			st, ok := status.FromError(err)
			if !ok {
				// Не gRPC-ошибка — считаем отказом.
				return false
			}
			switch st.Code() {
			case codes.NotFound,
				codes.FailedPrecondition,
				codes.AlreadyExists,
				codes.InvalidArgument,
				codes.PermissionDenied,
				codes.Unauthenticated,
				codes.Aborted:
				return true
			case codes.Unavailable,
				codes.DeadlineExceeded,
				codes.ResourceExhausted,
				codes.Internal:
				return false
			default:
				return false
			}
		},
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < cfg.MinRequestsToTrip {
				return false
			}
			return float64(counts.TotalFailures)/float64(counts.Requests) >= cfg.Threshold
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			slog.Warn("circuit breaker state changed",
				"name", name,
				"from", from.String(),
				"to", to.String(),
			)
		},
	}

	return gobreaker.NewCircuitBreaker[any](settings), nil
}
