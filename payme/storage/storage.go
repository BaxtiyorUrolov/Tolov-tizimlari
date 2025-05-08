package storage

import (
	"context"
	"github.com/BaxtiyorUrolov/Tolov-tizimlari/payme/models"
	"github.com/jmoiron/sqlx"
)

// StorageI defines the storage interface for Payme transactions.
type StorageI interface {
	Payme() PaymeI
}

// PaymeI defines the interface for payme-specific storage operations.
type PaymeI interface {
	CheckPayme(ctx context.Context, orderID string, amount int) (bool, error)
	GetPayme(ctx context.Context, orderID string) (*models.Payme, error)
	UpdatePayme(ctx context.Context, payme *models.Payme) error
}

type storage struct {
	db *sqlx.DB
}

// NewStorage creates a new storage instance with the given database.
func NewStorage(db *sqlx.DB) StorageI {
	return &storage{db: db}
}

func (s *storage) Payme() PaymeI {
	return &paymeStorage{db: s.db}
}
