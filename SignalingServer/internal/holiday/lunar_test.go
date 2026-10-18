package holiday

import "testing"

// The lunar new year is the one conversion everybody can check by memory, so
// it anchors the library before any policy depends on it. 2023-01-22,
// 2024-02-10, 2025-01-29 and 2026-02-17 are the first days of those lunar
// years.
func TestLunarNewYearKnownDates(t *testing.T) {
	cases := []struct {
		lunarYear int
		want      string
	}{
		{2023, "2023-01-22"},
		{2024, "2024-02-10"},
		{2025, "2025-01-29"},
		{2026, "2026-02-17"},
	}
	for _, tc := range cases {
		got, ok := LunarToSolar(tc.lunarYear, 1, 1)
		if !ok {
			t.Fatalf("lunar %d-01-01: conversion failed", tc.lunarYear)
		}
		if got.Format("2006-01-02") != tc.want {
			t.Fatalf("lunar %d-01-01 = %s, want %s", tc.lunarYear, got.Format("2006-01-02"), tc.want)
		}
	}
}

// 端午 and 中秋 are 5/5 and 8/15 on the lunar calendar. Their 2026 solar dates
// are also in the statutory calendar, so the two independent sources agree.
func TestLunarFestivals2026(t *testing.T) {
	cases := []struct {
		month, day int
		want       string
	}{
		{5, 5, "2026-06-19"},  // 端午节
		{8, 15, "2026-09-25"}, // 中秋节
		{1, 15, "2026-03-03"},
	}
	for _, tc := range cases {
		got, ok := LunarToSolar(2026, tc.month, tc.day)
		if !ok {
			t.Fatalf("lunar 2026-%02d-%02d: conversion failed", tc.month, tc.day)
		}
		if got.Format("2006-01-02") != tc.want {
			t.Fatalf("lunar 2026-%02d-%02d = %s, want %s", tc.month, tc.day, got.Format("2006-01-02"), tc.want)
		}
	}
}

func TestLunarRejectsImpossibleInput(t *testing.T) {
	for _, tc := range []struct{ y, m, d int }{
		{2026, 13, 1}, // no 13th month
		{2026, 0, 1},
		{2026, 1, 0},
		{2026, 1, 31},
		{1800, 1, 1}, // out of the supported range
	} {
		if got, ok := LunarToSolar(tc.y, tc.m, tc.d); ok {
			t.Fatalf("LunarToSolar(%d,%d,%d) = %s, want failure", tc.y, tc.m, tc.d, got.Format("2006-01-02"))
		}
	}
}

// 除夕 is the 30th of the 12th lunar month when that month has 30 days and the
// 29th when it has 29. The clamped conversion keeps a "除夕" entry working in
// both cases instead of silently skipping the short years.
func TestLunarClampedHandlesNewYearsEve(t *testing.T) {
	if _, ok := LunarToSolar(2026, 12, 30); ok {
		t.Fatal("the 12th month of lunar 2026 has 29 days, so 12-30 must not resolve")
	}
	eve, ok := LunarToSolarClamped(2026, 12, 30)
	if !ok {
		t.Fatal("clamped 除夕 must resolve")
	}
	if got := eve.Format("2006-01-02"); got != "2027-02-05" {
		t.Fatalf("除夕 of lunar 2026 = %s, want 2027-02-05 (12-29)", got)
	}
	// When the 30th does exist, it is used as-is.
	if got, ok := LunarToSolarClamped(2026, 12, 29); !ok || got.Format("2006-01-02") != "2027-02-05" {
		t.Fatalf("12-29 should resolve directly, got %v ok=%v", got.Format("2006-01-02"), ok)
	}
}
