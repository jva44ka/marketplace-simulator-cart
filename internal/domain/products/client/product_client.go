package client

import (
	"context"
	"fmt"
	"time"

	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pb "github.com/jva44ka/ozon-simulator-go-cart/internal/app/gen/ozon-simulator-go-products/api/v1/proto"
	"google.golang.org/grpc"
)

const (
	AuthHeaderKey string = "x-auth"
)

type ProductClient struct {
	grpcClient pb.ProductsClient
	authToken  string
	timeout    time.Duration
}

func NewProductClient(host string, port string, authToken string, timeout string) (*ProductClient, error) {
	conn, err := grpc.NewClient(host+":"+port, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func (c *ProductClient) GetProductBySku(ctx context.Context, sku uint64) (*model.Product, error) {
	req := &pb.GetProductRequest{
		Sku: sku,
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, AuthHeaderKey, c.authToken)

	resp, err := c.grpcClient.GetProduct(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ProductClient.GetProduct: %w", err)
	}

	product := model.Product{
		Sku:   resp.Sku,
		Price: resp.Price,
		Name:  resp.Name,
		Count: resp.Count,
	}

	return &product, nil
}

func (c *ProductClient) DecreaseProductCount(
	ctx context.Context,
	productCountsBySkus map[uint64]uint32) error {
	req := &pb.DecreaseProductCountRequest{
		Products: make([]*pb.DecreaseProductCountRequest_IncreaseStockBatch, 0),
	}

	for sku, count := range productCountsBySkus {
		req.Products = append(req.Products, &pb.DecreaseProductCountRequest_IncreaseStockBatch{
			Sku:   sku,
			Count: count,
		})
	}

	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	ctx = metadata.AppendToOutgoingContext(ctx, AuthHeaderKey, c.authToken)

	_, err := c.grpcClient.DecreaseProductCount(ctx, req)
	if err != nil {
		return fmt.Errorf("ProductClient.DecreaseProductCount: %w", err)
	}

	return nil
}
