package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jva44ka/ozon-simulator-go-cart/internal/domain/model"
)

type CartRepository interface {
	AddCartItem(_ context.Context, cartItem model.CartItem) (uint64, error)
	UpdateCartItem(_ context.Context, id uint64, cartItem model.CartItem) error
	GetCartItemsByUserId(_ context.Context, userId uuid.UUID) ([]model.CartItem, error)
	GetCartItem(_ context.Context, userId uuid.UUID, sku uint64) (*model.CartItem, error)
	RemoveCartItem(_ context.Context, userId uuid.UUID, sku uint64) error
	RemoveAllCartItemsByUserId(_ context.Context, userId uuid.UUID) error
}

type ProductRepository interface {
	GetProductBySku(ctx context.Context, sku uint64) (model.Product, error)
	AddProduct(ctx context.Context, product model.Product) (*model.Product, error)
}

type ProductClient interface {
	GetProductBySku(ctx context.Context, sku uint64) (*model.Product, error)
	ReserveProduct(ctx context.Context, productCountsBySkus map[uint64]uint32) (map[uint64]int64, error)
	ReleaseReservation(ctx context.Context, reservationIds []int64) error
	ConfirmReservation(ctx context.Context, reservationIds []int64) error
}

type CartService struct {
	cartRepository    CartRepository
	productClient     ProductClient
	productRepository ProductRepository
}

func NewCartService(
	cartRepository CartRepository,
	productClient ProductClient,
	productRepository ProductRepository,
) *CartService {
	return &CartService{
		cartRepository:    cartRepository,
		productClient:     productClient,
		productRepository: productRepository,
	}
}

func (s *CartService) AddProduct(ctx context.Context, userId uuid.UUID, sku uint64, count uint32) error {
	if count < 1 {
		return model.ErrProductsCountMustBeGreaterThanNull
	}

	productInMasterSystem, err := s.productClient.GetProductBySku(ctx, sku)
	if err != nil {
		return fmt.Errorf("productClient.GetProductBySku: %w", err)
	}

	existingCartItem, err := s.cartRepository.GetCartItem(ctx, userId, sku)
	if err != nil && !errors.Is(err, model.ErrCartItemsNotFound) {
		return fmt.Errorf("cartRepository.GetCartItem: %w", err)
	}

	if existingCartItem != nil {
		// Освобождаем старую резервацию, создаём новую на суммарный count
		if existingCartItem.ReservationId != 0 {
			if err = s.productClient.ReleaseReservation(ctx, []int64{existingCartItem.ReservationId}); err != nil {
				return fmt.Errorf("productClient.ReleaseReservation: %w", err)
			}
		}

		newTotal := existingCartItem.Count + count
		reservationIds, err := s.productClient.ReserveProduct(ctx, map[uint64]uint32{sku: newTotal})
		if err != nil {
			return fmt.Errorf("productClient.ReserveProduct: %w", err)
		}

		return s.cartRepository.UpdateCartItem(ctx, existingCartItem.Id, model.CartItem{
			Count:         newTotal,
			ReservationId: reservationIds[sku],
		})
	}

	// Новый элемент корзины: резервируем и добавляем
	reservationIds, err := s.productClient.ReserveProduct(ctx, map[uint64]uint32{sku: count})
	if err != nil {
		return fmt.Errorf("productClient.ReserveProduct: %w", err)
	}

	// Убеждаемся что продукт есть в локальной БД
	_, err = s.productRepository.GetProductBySku(ctx, sku)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			_, err = s.productRepository.AddProduct(ctx, model.Product{
				Sku:   sku,
				Price: productInMasterSystem.Price,
				Name:  productInMasterSystem.Name,
			})
			if err != nil {
				return fmt.Errorf("productRepository.AddProduct: %w", err)
			}
		} else {
			return fmt.Errorf("productRepository.GetProductsBySku: %w", err)
		}
	}

	_, err = s.cartRepository.AddCartItem(ctx, model.CartItem{
		UserId:        userId,
		Count:         count,
		ReservationId: reservationIds[sku],
		Product: model.Product{
			Sku:   sku,
			Price: productInMasterSystem.Price,
			Name:  productInMasterSystem.Name,
		},
	})
	return err
}

func (s *CartService) RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error {
	cartItem, err := s.cartRepository.GetCartItem(ctx, userId, sku)
	if err != nil {
		return fmt.Errorf("cartRepository.GetCartItem: %w", err)
	}

	if cartItem.ReservationId != 0 {
		if err = s.productClient.ReleaseReservation(ctx, []int64{cartItem.ReservationId}); err != nil {
			return fmt.Errorf("productClient.ReleaseReservation: %w", err)
		}
	}

	return s.cartRepository.RemoveCartItem(ctx, userId, sku)
}

func (s *CartService) RemoveAllProducts(ctx context.Context, userId uuid.UUID) error {
	cartItems, err := s.cartRepository.GetCartItemsByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("cartRepository.GetCartItemsByUserId: %w", err)
	}

	if len(cartItems) > 0 {
		ids := make([]int64, 0, len(cartItems))
		for _, item := range cartItems {
			if item.ReservationId != 0 {
				ids = append(ids, item.ReservationId)
			}
		}
		if len(ids) > 0 {
			if err = s.productClient.ReleaseReservation(ctx, ids); err != nil {
				return fmt.Errorf("productClient.ReleaseReservation: %w", err)
			}
		}
	}

	return s.cartRepository.RemoveAllCartItemsByUserId(ctx, userId)
}

func (s *CartService) GetItemsByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, float64, error) {
	cartItems, err := s.cartRepository.GetCartItemsByUserId(ctx, userId)
	if err != nil {
		return nil, 0.0, fmt.Errorf("cartRepository.GetCartItemsByUserId: %w", err)
	}

	totalPrice := 0.0
	for _, cartItem := range cartItems {
		totalPrice += cartItem.Product.Price
	}

	return cartItems, totalPrice, nil
}

func (s *CartService) Checkout(ctx context.Context, userId uuid.UUID) (float64, error) {
	cartItems, err := s.cartRepository.GetCartItemsByUserId(ctx, userId)
	if err != nil {
		return 0.0, fmt.Errorf("cartRepository.GetCartItemsByUserId: %w", err)
	}

	if len(cartItems) == 0 {
		return 0.0, model.ErrCartEmpty
	}

	ids := make([]int64, 0, len(cartItems))
	totalPrice := 0.0
	for _, item := range cartItems {
		if item.ReservationId != 0 {
			ids = append(ids, item.ReservationId)
		}
		totalPrice += item.Product.Price
	}

	if err = s.productClient.ConfirmReservation(ctx, ids); err != nil {
		return 0.0, fmt.Errorf("productClient.ConfirmReservation: %w", err)
	}

	if err = s.cartRepository.RemoveAllCartItemsByUserId(ctx, userId); err != nil {
		return 0.0, fmt.Errorf("cartRepository.RemoveAllCartItemsByUserId: %w", err)
	}

	return totalPrice, nil
}
