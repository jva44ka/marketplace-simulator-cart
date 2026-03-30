package products

import (
	"context"
	"fmt"
	"time"

	pb "github.com/jva44ka/ozon-simulator-go-cart/internal/infra/external_services/products/pb/ozon-simulator-go-products/api/v1/proto"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/model"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	AuthHeaderKey string = "x-auth"
)

type ProductClient struct {
	grpcClient pb.ProductsClient
	authToken  string
	timeout    time.Duration
}

func NewProductClient(host string, port string, authToken string, timeout string, opts ...grpc.DialOption) (*ProductClient, error) {
	dialOpts := append([]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}, opts...)
	conn, err := grpc.NewClient(host+":"+port, dialOpts...)
	if err != nil {
		return nil, err
	}

	parsedTimeout, err := time.ParseDuration(timeout)
	if err != nil {
		return nil, fmt.Errorf("ProductClient.NewProductClient: Error parsing timeout: %w", err)
	}

	client := pb.NewProductsClient(conn)
	return &ProductClient{
		grpcClient: client,
		authToken:  authToken,
		timeout:    parsedTimeout,
	}, nil
}

func (c *ProductClient) GetBySku(ctx context.Context, sku uint64) (*model.Product, error) {
	req := &pb.GetProductRequest{Sku: sku}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, AuthHeaderKey, c.authToken)

	resp, err := c.grpcClient.GetProduct(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			if st.Code() == codes.NotFound {
				return nil, model.ErrProductNotFound
			}
		}
		return nil, fmt.Errorf("ProductClient.GetProduct: %w", err)
	}

	return &model.Product{
		Sku:   resp.Sku,
		Price: resp.Price,
		Name:  resp.Name,
		Count: resp.Count,
	}, nil
}

// ReserveProduct резервирует товары и возвращает map[sku → reservation_id].
func (c *ProductClient) Reserve(
	ctx context.Context,
	productCountsBySkus map[uint64]uint32,
) (map[uint64]int64, error) {
	req := &pb.ReserveProductRequest{
		Products: make([]*pb.ReserveProductRequest_ProductCountBatch, 0, len(productCountsBySkus)),
	}

	for sku, count := range productCountsBySkus {
		req.Products = append(req.Products, &pb.ReserveProductRequest_ProductCountBatch{
			Sku:   sku,
			Count: count,
		})
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, AuthHeaderKey, c.authToken)

	resp, err := c.grpcClient.ReserveProduct(ctx, req)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return nil, model.ErrProductNotFound
			case codes.FailedPrecondition:
				return nil, model.ErrInsufficientStock
			}
		}
		return nil, fmt.Errorf("ProductClient.Reserve: %w", err)
	}

	result := make(map[uint64]int64, len(resp.Results))
	for _, r := range resp.Results {
		result[r.Sku] = r.ReservationId
	}

	return result, nil
}

// ReleaseReservation освобождает резервации по их IDs.
func (c *ProductClient) ReleaseReservation(ctx context.Context, reservationIds []int64) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, AuthHeaderKey, c.authToken)

	_, err := c.grpcClient.ReleaseReservation(ctx, &pb.ReleaseReservationRequest{ReservationIds: reservationIds})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return model.ErrProductNotFound
			case codes.FailedPrecondition:
				return model.ErrInsufficientStock
			}
		}
		return fmt.Errorf("ProductClient.ReleaseReservation: %w", err)
	}

	return nil
}

// ConfirmReservation подтверждает резервации по их IDs.
func (c *ProductClient) ConfirmReservation(ctx context.Context, reservationIds []int64) error {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, AuthHeaderKey, c.authToken)

	_, err := c.grpcClient.ConfirmReservation(ctx, &pb.ConfirmReservationRequest{ReservationIds: reservationIds})
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.NotFound:
				return model.ErrProductNotFound
			case codes.FailedPrecondition:
				return model.ErrInsufficientStock
			}
		}
		return fmt.Errorf("ProductClient.ConfirmReservation: %w", err)
	}

	return nil
}
