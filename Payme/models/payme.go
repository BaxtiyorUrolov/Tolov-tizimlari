package models

import "time"

type Payme struct {
	ID                 string
	UserID             int
	Amount             int
	State              int
	PaymeTransactionID string
	CreateTime         time.Time
	PerformTime        *time.Time
	CancelTime         *time.Time
}
