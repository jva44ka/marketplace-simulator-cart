package model

type ReservationConfirmationOutboxRecordNew struct {
	Key     string
	Data    []byte
	Headers []byte
}
