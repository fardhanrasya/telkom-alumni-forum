package repository

import (
	"context"
	"errors"

	"anoa.com/telkomalumiforum/internal/entity"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ErrInsufficientBalance is returned by ApplyLedgerEntry when a debit would
// push the wallet balance below zero.
var ErrInsufficientBalance = errors.New("insufficient balance")

type WalletRepository interface {
	GetOrCreateWallet(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error)
	// ApplyLedgerEntry credits (delta > 0) or debits (delta < 0) a wallet inside
	// the caller-supplied transaction, guarded by SELECT ... FOR UPDATE and the
	// (source_type, source_ref_id) idempotency key. If a transaction already
	// exists for that key, the existing row is returned as-is (no-op).
	ApplyLedgerEntry(tx *gorm.DB, userID uuid.UUID, delta int, sourceType, sourceRefID string) (*entity.CoinTransaction, error)
	WithTx(tx *gorm.DB) WalletRepository
	DB() *gorm.DB
}

type walletRepository struct {
	db *gorm.DB
}

func NewWalletRepository(db *gorm.DB) WalletRepository {
	return &walletRepository{db: db}
}

func (r *walletRepository) DB() *gorm.DB {
	return r.db
}

func (r *walletRepository) WithTx(tx *gorm.DB) WalletRepository {
	return &walletRepository{db: tx}
}

func (r *walletRepository) GetOrCreateWallet(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error) {
	var wallet entity.Wallet
	err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Where("user_id = ?", userID).
		FirstOrCreate(&wallet, entity.Wallet{UserID: userID, Balance: 0}).Error
	if err != nil {
		return nil, err
	}
	return &wallet, nil
}

func (r *walletRepository) ApplyLedgerEntry(tx *gorm.DB, userID uuid.UUID, delta int, sourceType, sourceRefID string) (*entity.CoinTransaction, error) {
	// A cheap pre-check avoids taking the wallet lock at all for the common
	// case (no prior entry). It is NOT sufficient on its own: without a lock,
	// two concurrent calls for the same idempotency key could both pass it
	// before either commits. The authoritative check is repeated below,
	// after the wallet row lock, which serializes concurrent callers for the
	// same user and guarantees the recheck sees any entry the other call
	// already committed.
	var existing entity.CoinTransaction
	err := tx.Where("source_type = ? AND source_ref_id = ?", sourceType, sourceRefID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	var wallet entity.Wallet
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).First(&wallet).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		wallet = entity.Wallet{UserID: userID, Balance: 0}
		if err := tx.Create(&wallet).Error; err != nil {
			return nil, err
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("user_id = ?", userID).First(&wallet).Error; err != nil {
			return nil, err
		}
	}

	// Authoritative recheck, now that concurrent callers for this user are
	// serialized by the wallet row lock above.
	err = tx.Where("source_type = ? AND source_ref_id = ?", sourceType, sourceRefID).First(&existing).Error
	if err == nil {
		return &existing, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	newBalance := wallet.Balance + delta
	if newBalance < 0 {
		return nil, ErrInsufficientBalance
	}

	if err := tx.Model(&entity.Wallet{}).Where("user_id = ?", userID).
		Update("balance", newBalance).Error; err != nil {
		return nil, err
	}

	entry := &entity.CoinTransaction{
		UserID:       userID,
		Amount:       delta,
		SourceType:   sourceType,
		SourceRefID:  sourceRefID,
		BalanceAfter: newBalance,
	}
	if err := tx.Create(entry).Error; err != nil {
		return nil, err
	}

	return entry, nil
}
