package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMeetsMinRank_NoGateAlwaysPasses(t *testing.T) {
	assert.True(t, meetsMinRank("Pendatang", ""))
}

func TestMeetsMinRank_ExactRankPasses(t *testing.T) {
	assert.True(t, meetsMinRank("Tokoh", "Tokoh"))
}

func TestMeetsMinRank_HigherRankPasses(t *testing.T) {
	assert.True(t, meetsMinRank("Legenda", "Tokoh"))
}

func TestMeetsMinRank_LowerRankFails(t *testing.T) {
	assert.False(t, meetsMinRank("Warga", "Tokoh"))
}

func TestMeetsMinRank_UnknownRankFails(t *testing.T) {
	assert.False(t, meetsMinRank("NotARank", "Tokoh"))
}
