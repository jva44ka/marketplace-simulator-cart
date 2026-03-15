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
	if sku < 1 {
		return errors.New("sku must be greater than zero")
	}

	if userId == uuid.Nil {
		return errors.New("user_id must be not nil")
	}

	if count < 1 {
		return errors.New("count must be greater than zero")
	}

	//Такой продукт уже есть в корзине - прибавляем количество
	// TODO: refactor this
	existingCartItem, err := s.cartRepository.GetCartItem(ctx, userId, sku)
	if err != nil && !errors.Is(err, model.ErrCartItemsNotFound) {
		return fmt.Errorf("cartRepository.GetCartItem: %w", err)
	}
	if existingCartItem != nil {
		resultCount := existingCartItem.Count + count
		err = s.cartRepository.UpdateCartItem(ctx, existingCartItem.Id, model.CartItem{
			Count: resultCount,
		})

		if err != nil {
			return fmt.Errorf("cartRepository.UpdateCartItem: %w", err)
		}

		return nil
	}

	// Продукта нет в корзине - запрашиваем сначала в мастер-системе продуктов, если нет - ошибка
	productInMasterSystem, err := s.productClient.GetProductBySku(ctx, sku)
	if err != nil {
		if errors.Is(err, model.ErrProductNotFound) {
			return fmt.Errorf("productClient.GetProductBySku: %w", err)
		}

		return err
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
	if sku < 1 {
		return errors.New("sku must be greater than zero")
	}

	if userId == uuid.Nil {
		return errors.New("user_id must be not nil")
	}

	err := s.cartRepository.RemoveCartItem(ctx, userId, sku)
	if err != nil {
		return fmt.Errorf("cartRepository.RemoveProduct :%w", err)
	}

	return nil
}

func (s *CartService) RemoveAllProducts(ctx context.Context, userId uuid.UUID) error {
	if userId == uuid.Nil {
		return errors.New("user_id must be not nil")
	}

	err := s.cartRepository.RemoveAllCartItemsByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("cartRepository.RemoveAllCartItemsByUserId :%w", err)
	}

	return nil
}

func (s *CartService) GetItemsByUserId(ctx context.Context, userId uuid.UUID) ([]model.CartItem, error) {
	if userId == uuid.Nil {
		return nil, errors.New("userId must be not Nil")
	}

	cartItems, err := s.cartRepository.GetCartItemsByUserId(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("cartRepository.GetCartItemsByUserId :%w", err)
	}

	return cartItems, nil
}

func (s *CartService) Checkout(ctx context.Context, userId uuid.UUID) error {
	if userId == uuid.Nil {
		return errors.New("user_id must be not nil")
	}

	cartItems, err := s.cartRepository.GetCartItemsByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("cartRepository.GetCartItemsByUserId :%w", err)
	}

	if len(cartItems) == 0 {
		return errors.New("cartItems is empty")
	}

	productCountsBySku := map[uint64]uint32{}
	for _, cartItem := range cartItems {
		productCountsBySku[cartItem.Product.Sku] = cartItem.Count
	}

	//TODO сделать аутбокс для похода в products
	err = s.productClient.DecreaseProductCount(ctx, productCountsBySku)
	if err != nil {
		return fmt.Errorf("productClient.DecreaseProductCount :%w", err)
	}

	err = s.cartRepository.RemoveAllCartItemsByUserId(ctx, userId)
	if err != nil {
		return fmt.Errorf("cartRepository.RemoveAllCartItemsByUserId :%w", err)
	}

	return nil
}
