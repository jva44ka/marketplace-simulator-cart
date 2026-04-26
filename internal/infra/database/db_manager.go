package database

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jva44ka/marketplace-simulator-cart/internal/infra/database/repository"
	"github.com/jva44ka/marketplace-simulator-cart/internal/service"
)

type DBManagerMetrics interface {
	ReportRequest(method, status string, duration time.Duration)
}

type DBManager struct {
	pool      *pgxpool.Pool
	cartItems *repository.PgxCartItemRepository
	products  *repository.PgxProductRepository
	outbox    *repository.ReservationConfirmationOutboxPgxRepository
}

func NewDBManager(pool *pgxpool.Pool, metrics DBManagerMetrics) *DBManager {
	return &DBManager{
		pool:      pool,
		cartItems: repository.NewPgxCartItemRepository(pool, metrics),
		products:  repository.NewPgxProductRepository(pool, metrics),
		outbox:    repository.NewReservationConfirmationOutboxPgxRepository(pool),
	}
}

func (m *DBManager) CartItemRepo() service.CartItemRepository {
	return m.cartItems
}

func (m *DBManager) ProductRepo() service.ProductRepository {
	return m.products
}

func (m *DBManager) OutboxRepo() service.OutboxRepository {
	return m.outbox
}

func (m *DBManager) InTransaction(ctx context.Context, fn func(pgx.Tx) error) error {
	return pgx.BeginTxFunc(ctx, m.pool, pgx.TxOptions{}, fn)
}

func (m *DBManager) OutboxPgxRepo() *repository.ReservationConfirmationOutboxPgxRepository {
	return m.outbox
}
