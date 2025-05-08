package storage

import (
	"context"
	"github.com/BaxtiyorUrolov/Tolov-tizimlari/payme/models"
	"github.com/jmoiron/sqlx"
)

type paymeStorage struct {
	db *sqlx.DB
}

// CheckPayme checks if a transaction with the given order ID and amount exists.
func (s *paymeStorage) CheckPayme(ctx context.Context, orderID string, amount int) (bool, error) {
	var exists bool
	query := `SELECT EXISTS (SELECT 1 FROM payme WHERE id = $1 AND amount = $2 AND state = 1)`
	err := s.db.GetContext(ctx, &exists, query, orderID, amount)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// GetPayme retrieves a payme transaction by its order ID.
func (s *paymeStorage) GetPayme(ctx context.Context, orderID string) (*models.Payme, error) {
	var payme models.Payme
	query := `SELECT * FROM payme WHERE id = $1`
	err := s.db.GetContext(ctx, &payme, query, orderID)
	if err != nil {
		return nil, err
	}
	return &payme, nil
}

// UpdatePayme updates the state and timestamps of a payme transaction.
func (s *paymeStorage) UpdatePayme(ctx context.Context, payme *models.Payme) error {
	query := `
		UPDATE payme
		SET state = :state,
			perform_time = :perform_time,
			cancel_time = :cancel_time
		WHERE id = :id`
	_, err := s.db.NamedExecContext(ctx, query, payme)
	return err
}
