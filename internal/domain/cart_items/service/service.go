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
	DecreaseProductCount(ctx context.Context, productCountsBySkus map[uint64]uint32) error
}

type CartService struct {
	cartRepository    CartRepository
	productClient     ProductClient
	productRepository ProductRepository
}

func NewCartService(cartRepository CartRepository, productService ProductClient, productRepository ProductRepository) *CartService {
	return &CartService{cartRepository: cartRepository, productClient: productService, productRepository: productRepository}
}

func (s *CartService) AddProduct(ctx context.Context, userId uuid.UUID, sku uint64, count uint32) error {
	if count < 1 {
		return model.ErrProductsCountMustBeGreaterThanNull
	}

	// Всегда запрашиваем актуальный остаток из мастер-системы
	productInMasterSystem, err := s.productClient.GetProductBySku(ctx, sku)
	if err != nil {
		return fmt.Errorf("productClient.GetProductBySku: %w", err)
	}

	existingCartItem, err := s.cartRepository.GetCartItem(ctx, userId, sku)
	if err != nil && !errors.Is(err, model.ErrCartItemsNotFound) {
		return fmt.Errorf("cartRepository.GetCartItem: %w", err)
	}

	alreadyInCart := uint32(0)
	if existingCartItem != nil {
		alreadyInCart = existingCartItem.Count
	}

	if alreadyInCart+count > productInMasterSystem.Count {
		return model.ErrInsufficientStock
	}

	// Товар уже есть в корзине — прибавляем количество
	if existingCartItem != nil {
		err = s.cartRepository.UpdateCartItem(ctx, existingCartItem.Id, model.CartItem{
			Count: alreadyInCart + count,
		})
		if err != nil {
			return fmt.Errorf("cartRepository.UpdateCartItem: %w", err)
		}
		return nil
	}

	// Теперь смотрим у себя в базе есть ли этот продукт, если нет - добавляем
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

	cartItem := model.CartItem{
		UserId: userId,
		Count:  count,
		Product: model.Product{
			Sku:   sku,
			Price: productInMasterSystem.Price,
			Name:  productInMasterSystem.Name,
		},
	}

	_, err = s.cartRepository.AddCartItem(ctx, cartItem)
	if err != nil {
		return fmt.Errorf("cartRepository.AddCartItem :%w", err)
	}

	return nil
}

func (s *CartService) RemoveProduct(ctx context.Context, userId uuid.UUID, sku uint64) error {
	err := s.cartRepository.RemoveCartItem(ctx, userId, sku)
	if err != nil {
		return fmt.Errorf("cartRepository.RemoveProduct :%w", err)
	}

	return nil
}

func (s *CartService) RemoveAllProducts(ctx context.Context, userId uuid.UUID) error {
	err := s.cartRepository.RemoveAllCartItemsByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("cartRepository.RemoveAllCartItemsByUserId :%w", err)
	}

	return nil
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

	productCountsBySku := map[uint64]uint32{}
	totalPrice := 0.0
	for _, cartItem := range cartItems {
		productCountsBySku[cartItem.Product.Sku] = cartItem.Count
		totalPrice += cartItem.Product.Price
	}

	//TODO сделать аутбокс для похода в products
	err = s.productClient.DecreaseProductCount(ctx, productCountsBySku)
	if err != nil {
		return 0.0, fmt.Errorf("productClient.DecreaseProductCount :%w", err)
	}

	err = s.cartRepository.RemoveAllCartItemsByUserId(ctx, userId)
	if err != nil {
		return 0.0, fmt.Errorf("cartRepository.RemoveAllCartItemsByUserId :%w", err)
	}

	return totalPrice, nil
}
