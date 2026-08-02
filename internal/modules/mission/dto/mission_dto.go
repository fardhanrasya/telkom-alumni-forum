package dto

type MissionResponse struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"` // "daily" | "achievement"
	ActionType string `json:"action_type"`
	Target     int    `json:"target"`
	Reward     int    `json:"reward"`
	Progress   int    `json:"progress"`
	Status     string `json:"status"` // "in_progress" | "claimable" | "claimed"
}

type ClaimResponse struct {
	Reward     int `json:"reward"`
	NewBalance int `json:"new_balance"`
}
