package service

import (
	"context"

	walletDto "anoa.com/telkomalumiforum/internal/modules/wallet/dto"
	walletRepo "anoa.com/telkomalumiforum/internal/modules/wallet/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CoinGrantRequest is the internal, event-based contract every coin faucet
// (mission claims now; mini-games/admin-grant later) must go through. The
// caller reports what happened; only server-side code decides the amount —
// no untrusted caller ever sets Amount directly on a public endpoint.
type CoinGrantRequest struct {
	UserID      uuid.UUID
	Amount      int
	SourceType  string // "mission_claim" | "admin_grant" | "correction" | ...
	SourceRefID string // idempotency key, unique per (SourceType, SourceRefID)
}

type WalletService interface {
	GetWallet(ctx context.Context, userID uuid.UUID) (*walletDto.WalletResponse, error)
	// Grant credits coin to a user via the CoinGrant contract. Idempotent on
	// (SourceType, SourceRefID) — calling twice with the same key is a no-op.
	Grant(ctx context.Context, req CoinGrantRequest) error
	GetTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) (*walletDto.TransactionListResponse, error)
}

type walletService struct {
	repo walletRepo.WalletRepository
}

func NewWalletService(repo walletRepo.WalletRepository) WalletService {
	return &walletService{repo: repo}
}

func (s *walletService) GetWallet(ctx context.Context, userID uuid.UUID) (*walletDto.WalletResponse, error) {
	wallet, err := s.repo.GetOrCreateWallet(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &walletDto.WalletResponse{Balance: wallet.Balance}, nil
}

func (s *walletService) Grant(ctx context.Context, req CoinGrantRequest) error {
	db := s.repo.DB().WithContext(ctx)
	return db.Transaction(func(tx *gorm.DB) error {
		_, err := s.repo.WithTx(tx).ApplyLedgerEntry(tx, req.UserID, req.Amount, req.SourceType, req.SourceRefID)
		return err
	})
}

func (s *walletService) GetTransactions(ctx context.Context, userID uuid.UUID, limit, offset int) (*walletDto.TransactionListResponse, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	transactions, err := s.repo.GetTransactions(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
	}

	resp := &walletDto.TransactionListResponse{
		Data: make([]walletDto.TransactionResponse, 0, len(transactions)),
	}
	resp.Meta.Limit = limit
	resp.Meta.Offset = offset
	for _, t := range transactions {
		resp.Data = append(resp.Data, walletDto.TransactionResponse{
			ID:           t.ID,
			Amount:       t.Amount,
			SourceType:   t.SourceType,
			BalanceAfter: t.BalanceAfter,
			CreatedAt:    t.CreatedAt,
		})
	}
	return resp, nil
}
