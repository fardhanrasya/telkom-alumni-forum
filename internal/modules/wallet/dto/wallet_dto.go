package dto

import "time"

// WalletResponse carries the raw integer balance — display formatting
// ("1,2rb" vs "1.250 TC") is a FE decision, not BE's.
type WalletResponse struct {
	Balance int `json:"balance"`
}

type TransactionResponse struct {
	ID           uint      `json:"id"`
	Amount       int       `json:"amount"`
	SourceType   string    `json:"source_type"`
	BalanceAfter int       `json:"balance_after"`
	CreatedAt    time.Time `json:"created_at"`
}

type AdminGrantRequest struct {
	Username string `json:"username" binding:"required"`
	Amount   int    `json:"amount" binding:"required,min=1"`
}

type TransactionListResponse struct {
	Data []TransactionResponse `json:"data"`
	Meta struct {
		Limit  int `json:"limit"`
		Offset int `json:"offset"`
	} `json:"meta"`
}
