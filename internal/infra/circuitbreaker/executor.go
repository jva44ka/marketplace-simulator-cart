package circuitbreaker

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/config"
	"github.com/sony/gobreaker/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Executor struct {
	cb *gobreaker.CircuitBreaker[any]
}

func NewExecutor(cfg config.CircuitBreakerConfig, name string) (*Executor, error) {
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
		MaxRequests: cfg.MaxRequests,
		Interval:    interval,
		Timeout:     timeout,
		// Считаем отказом только инфраструктурные ошибки (сервер недоступен,
		// таймаут, внутренняя ошибка). Бизнес-ошибки (NotFound,
		// FailedPrecondition и т.д.) не должны приближать срабатывание CB.
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
				codes.ResourceExhausted: // не имеет смысла ретраить
				return true
			case codes.Aborted: // оптимистичная блокировка — можем ретраить
				return false
			default:
				// Unavailable, DeadlineExceeded, Internal, Unknown и пр. — ретраим.
				return false
			}
		},
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			if counts.Requests < 5 {
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

	return &Executor{cb: gobreaker.NewCircuitBreaker[any](settings)}, nil
}

func (e *Executor) Execute(fn func() (any, error)) (any, error) {
	return e.cb.Execute(fn)
}

// UnaryClientInterceptor возвращает gRPC client interceptor,
// который пропускает каждый исходящий вызов через circuit breaker.
func (e *Executor) UnaryClientInterceptor() grpc.UnaryClientInterceptor {
	return func(
		ctx context.Context,
		method string,
		req, reply any,
		cc *grpc.ClientConn,
		invoker grpc.UnaryInvoker,
		opts ...grpc.CallOption,
	) error {
		_, err := e.cb.Execute(func() (any, error) {
			return nil, invoker(ctx, method, req, reply, cc, opts...)
		})
		return err
	}
}
