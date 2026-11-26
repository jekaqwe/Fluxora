package main

import (
	"testing"
	"time"
)

func TestCalculateActivityStreakCountsDistinctDays(t *testing.T) {
	today := time.Date(2026, 11, 24, 15, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	dayBefore := today.AddDate(0, 0, -2)
	tests := []struct {
		name  string
		dates []time.Time
		want  int
	}{
		{"multiple operations on same day", []time.Time{today, today.Add(time.Hour), today}, 1},
		{"consecutive days", []time.Time{today, yesterday, dayBefore}, 3},
		{"streak can continue from yesterday", []time.Time{yesterday, dayBefore}, 2},
		{"gap breaks streak", []time.Time{today, dayBefore}, 1},
		{"stale activity", []time.Time{dayBefore}, 0},
		{"no activity", nil, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := calculateActivityStreak(tt.dates, today); got != tt.want {
				t.Fatalf("calculateActivityStreak()=%d, want %d", got, tt.want)
			}
		})
	}
}
