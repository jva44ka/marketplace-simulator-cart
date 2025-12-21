package model

import "github.com/google/uuid"

type CartItem struct {
	Id      uint64
	Product Product
	UserId  uuid.UUID
	Count   uint32
}
