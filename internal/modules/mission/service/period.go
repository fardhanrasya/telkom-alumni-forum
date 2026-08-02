package service

import "time"

// AchievementPeriod is the fixed period key for one-time missions (never resets).
const AchievementPeriod = "lifetime"

var wibZone = time.FixedZone("WIB", 7*3600)

// PeriodFor returns the mission period key for a given kind at time t:
// the WIB calendar date for "daily" missions (so the day rolls over at
// 00:00 WIB, not UTC), or AchievementPeriod for anything else.
func PeriodFor(kind string, t time.Time) string {
	if kind != "daily" {
		return AchievementPeriod
	}
	return t.In(wibZone).Format("2006-01-02")
}
