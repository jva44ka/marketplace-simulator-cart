package model

import (
	"time"

	"github.com/google/uuid"
)

type ReservationConfirmationOutboxRecord struct {
	RecordId            uuid.UUID
	Key                 string
	Data                []byte
	Headers             []byte
	CreatedAt           time.Time
	RetryCount          int
	IsDeadLetter        bool
	MarkedAsDeadLetterAt *time.Time
	DeadLetterReason    *string
}
