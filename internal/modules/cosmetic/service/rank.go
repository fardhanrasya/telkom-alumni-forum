package service

// rankOrder mirrors internal/modules/leaderboard/service/rank_helper.go's
// permanent rank ladder, lowest first. Kept as a local literal (rather than
// importing leaderboard internals) since it's a stable, publicly documented
// UI concept, not private leaderboard state.
var rankOrder = []string{"Pendatang", "Warga", "Aktivis", "Tokoh", "Sepuh", "Legenda"}

func rankIndex(rank string) int {
	for i, r := range rankOrder {
		if r == rank {
			return i
		}
	}
	return -1
}

// meetsMinRank reports whether currentRank is at or above minRank. An empty
// minRank means the cosmetic has no rank gate.
func meetsMinRank(currentRank, minRank string) bool {
	if minRank == "" {
		return true
	}
	currentIdx := rankIndex(currentRank)
	minIdx := rankIndex(minRank)
	if currentIdx == -1 || minIdx == -1 {
		return false
	}
	return currentIdx >= minIdx
}
