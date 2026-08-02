package dto

// WalletResponse carries the raw integer balance — display formatting
// ("1,2rb" vs "1.250 TC") is a FE decision, not BE's.
type WalletResponse struct {
	Balance int `json:"balance"`
}
