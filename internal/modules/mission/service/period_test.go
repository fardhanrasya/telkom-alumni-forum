package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPeriodFor_Achievement_IsAlwaysLifetime(t *testing.T) {
	got := PeriodFor("achievement", time.Now())
	assert.Equal(t, AchievementPeriod, got)
}

func TestPeriodFor_Daily_UsesWIBNotUTC(t *testing.T) {
	// 17:30 UTC on Aug 1 is already 00:30 WIB on Aug 2 - the daily period
	// must roll over at WIB midnight, not UTC midnight.
	utc := time.Date(2026, 8, 1, 17, 30, 0, 0, time.UTC)
	assert.Equal(t, "2026-08-02", PeriodFor("daily", utc))
}

func TestPeriodFor_Daily_BeforeWIBMidnightStaysOnPreviousDay(t *testing.T) {
	// 16:59 UTC on Aug 1 is 23:59 WIB on Aug 1 - still the previous day.
	utc := time.Date(2026, 8, 1, 16, 59, 0, 0, time.UTC)
	assert.Equal(t, "2026-08-01", PeriodFor("daily", utc))
}
