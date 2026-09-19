package cronex

import (
	"testing"
	"time"
)

func fixed(t *testing.T, loc *time.Location, y int, mo time.Month, d, h, mi, s int) time.Time {
	t.Helper()
	return time.Date(y, mo, d, h, mi, s, 0, loc)
}

func utcPlus1(t *testing.T) (*time.Location, time.Time) {
	t.Helper()
	loc := time.FixedZone("UTC+1", 3600)
	return loc, fixed(t, loc, 2024, time.January, 1, 0, 0, 0)
}

func TestNextBasics(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)

	tests := []struct {
		expr  string
		after time.Time
		want  time.Time
	}{
		{"0 0 * * *", fixed(t, loc, 2024, time.January, 10, 12, 34, 56), fixed(t, loc, 2024, time.January, 11, 0, 0, 0)},
		{"* * * * *", fixed(t, loc, 2024, time.January, 1, 12, 34, 56), fixed(t, loc, 2024, time.January, 1, 12, 35, 0)},
		{"*/10 * * * *", fixed(t, loc, 2024, time.January, 1, 12, 0, 0), fixed(t, loc, 2024, time.January, 1, 12, 10, 0)},
		{"0 9 * * *", fixed(t, loc, 2024, time.January, 1, 9, 0, 0), fixed(t, loc, 2024, time.January, 2, 9, 0, 0)},
		{"*/15 9-17 * * 1-5", fixed(t, loc, 2024, time.January, 10, 9, 5, 0), fixed(t, loc, 2024, time.January, 10, 9, 15, 0)},
		{"*/15 9-17 * * 1-5", fixed(t, loc, 2024, time.January, 12, 17, 45, 0), fixed(t, loc, 2024, time.January, 15, 9, 0, 0)},
		{"*/15 9-17 * * 1-5", fixed(t, loc, 2024, time.January, 13, 0, 0, 0), fixed(t, loc, 2024, time.January, 15, 9, 0, 0)},
		{"30 6 * * 1-5", fixed(t, loc, 2024, time.January, 12, 23, 59, 59), fixed(t, loc, 2024, time.January, 15, 6, 30, 0)},
	}
	for _, tt := range tests {
		e, err := Parse(tt.expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.expr, err)
		}
		got, err := e.NextErr(tt.after)
		if err != nil {
			t.Fatalf("NextErr(%q, %v): unexpected error: %v", tt.expr, tt.after, err)
		}
		if !got.Equal(tt.want) {
			t.Errorf("Next(%q) after %v = %v, want %v", tt.expr, tt.after.Format("2006-01-02 15:04:05"), got.Format("2006-01-02 15:04:05"), tt.want.Format("2006-01-02 15:04:05"))
		}
	}
}

func TestNextSundayEquivalence(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	after := fixed(t, loc, 2024, time.January, 10, 8, 0, 0) // a Wednesday
	want := fixed(t, loc, 2024, time.January, 14, 9, 0, 0)  // Sunday

	for _, expr := range []string{"0 9 * * 0", "0 9 * * 7", "0 9 * * sun", "0 9 * * SUN"} {
		e, err := Parse(expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", expr, err)
		}
		got, err := e.NextErr(after)
		if err != nil {
			t.Fatalf("NextErr(%q): %v", expr, err)
		}
		if !got.Equal(want) {
			t.Errorf("Next(%q) = %v, want %v", expr, got, want)
		}
	}
}

func TestNextLeapDay(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)

	e, err := Parse("0 0 29 2 *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Leap year immediately ahead.
	got, err := e.NextErr(fixed(t, loc, 2023, time.March, 1, 0, 0, 0))
	if err != nil {
		t.Fatalf("NextErr: %v", err)
	}
	want := fixed(t, loc, 2024, time.February, 29, 0, 0, 0)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}

	// Four years ahead, still inside the five-year window.
	got, err = e.NextErr(fixed(t, loc, 2024, time.February, 29, 12, 0, 0))
	if err != nil {
		t.Fatalf("NextErr: %v", err)
	}
	want = fixed(t, loc, 2028, time.February, 29, 0, 0, 0)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}
}

func TestNextImpossibleReturnsError(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	tests := []string{
		"0 0 30 2 *", // 30 February never exists
		"0 0 31 4 *", // 31 April never exists
	}
	for _, expr := range tests {
		after := fixed(t, loc, 2024, time.January, 1, 0, 0, 0)
		e, err := Parse(expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", expr, err)
		}
		if _, err := e.NextErr(after); err == nil {
			t.Errorf("NextErr(%q): expected an error for an impossible expression", expr)
		}
		if got := e.Next(after); !got.IsZero() {
			t.Errorf("Next(%q): expected zero time on error, got %v", expr, got)
		}
	}
}

func TestNextDomDowOr(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	// 0 0 13 * 5 fires on the 13th AND on Fridays. Monday 1 Jan 2024.
	after := fixed(t, loc, 2024, time.January, 1, 0, 0, 0)
	want := []time.Time{
		fixed(t, loc, 2024, time.January, 5, 0, 0, 0),   // Friday
		fixed(t, loc, 2024, time.January, 12, 0, 0, 0),  // Friday
		fixed(t, loc, 2024, time.January, 13, 0, 0, 0),  // Saturday the 13th
		fixed(t, loc, 2024, time.January, 19, 0, 0, 0),  // Friday
		fixed(t, loc, 2024, time.January, 26, 0, 0, 0),  // Friday
	}
	e, err := Parse("0 0 13 * 5")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := e.NextN(after, len(want))
	if err != nil {
		t.Fatalf("NextN: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("NextN returned %d times, want %d", len(got), len(want))
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Errorf("NextN[%d] = %v, want %v", i, got[i].Format("2006-01-02 15:04:05"), want[i].Format("2006-01-02 15:04:05"))
		}
	}
}

func TestNextDomOnlyMonthLengths(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	// dom=31 only: February and April must be skipped.
	e, err := Parse("0 0 31 * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := e.NextErr(fixed(t, loc, 2024, time.January, 31, 12, 0, 0))
	if err != nil {
		t.Fatalf("NextErr: %v", err)
	}
	want := fixed(t, loc, 2024, time.March, 31, 0, 0, 0)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got.Format("2006-01-02 15:04:05"), want.Format("2006-01-02 15:04:05"))
	}
}

func TestNextAliases(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)

	tests := []struct {
		alias string
		after time.Time
		want  time.Time
	}{
		{"@hourly", fixed(t, loc, 2024, time.January, 1, 12, 34, 0), fixed(t, loc, 2024, time.January, 1, 13, 0, 0)},
		{"@daily", fixed(t, loc, 2024, time.February, 28, 23, 59, 0), fixed(t, loc, 2024, time.February, 29, 0, 0, 0)},
		{"@midnight", fixed(t, loc, 2024, time.February, 28, 23, 59, 0), fixed(t, loc, 2024, time.February, 29, 0, 0, 0)},
		{"@weekly", fixed(t, loc, 2024, time.January, 10, 8, 0, 0), fixed(t, loc, 2024, time.January, 14, 0, 0, 0)}, // next Sunday
		{"@monthly", fixed(t, loc, 2024, time.January, 31, 12, 0, 0), fixed(t, loc, 2024, time.February, 1, 0, 0, 0)},
		{"@yearly", fixed(t, loc, 2024, time.December, 31, 12, 0, 0), fixed(t, loc, 2025, time.January, 1, 0, 0, 0)},
		{"@annually", fixed(t, loc, 2024, time.December, 31, 12, 0, 0), fixed(t, loc, 2025, time.January, 1, 0, 0, 0)},
	}
	for _, tt := range tests {
		e, err := Parse(tt.alias)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.alias, err)
		}
		got, err := e.NextErr(tt.after)
		if err != nil {
			t.Fatalf("NextErr(%q): %v", tt.alias, err)
		}
		if !got.Equal(tt.want) {
			t.Errorf("Next(%q) after %v = %v, want %v", tt.alias, tt.after.Format("2006-01-02 15:04:05"), got.Format("2006-01-02 15:04:05"), tt.want.Format("2006-01-02 15:04:05"))
		}
	}
}

func TestNextYearRollover(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	e, err := Parse("30 23 31 12 *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := e.NextErr(fixed(t, loc, 2024, time.December, 31, 23, 31, 0))
	if err != nil {
		t.Fatalf("NextErr: %v", err)
	}
	want := fixed(t, loc, 2025, time.December, 31, 23, 30, 0)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got.Format("2006-01-02 15:04:05"), want.Format("2006-01-02 15:04:05"))
	}
}

func TestNextKeepsLocation(t *testing.T) {
	loc := time.FixedZone("UTC+1", 3600)
	e, err := Parse("0 0 * * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := e.NextErr(fixed(t, loc, 2024, time.January, 1, 12, 0, 0))
	if err != nil {
		t.Fatalf("NextErr: %v", err)
	}
	if got.Location() != loc {
		t.Errorf("Next result in %v, want %v", got.Location(), loc)
	}
}

func TestNextInUTC(t *testing.T) {
	e, err := Parse("0 0 * * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	got, err := e.NextErr(fixed(t, time.UTC, 2024, time.January, 1, 23, 45, 0))
	if err != nil {
		t.Fatalf("NextErr: %v", err)
	}
	want := fixed(t, time.UTC, 2024, time.January, 2, 0, 0, 0)
	if !got.Equal(want) {
		t.Errorf("Next = %v, want %v", got, want)
	}
}

func TestNextNOrderStrictlyAfter(t *testing.T) {
	_, after := utcPlus1(t)
	e, err := Parse("*/15 * * * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	times, err := e.NextN(after, 5)
	if err != nil {
		t.Fatalf("NextN: %v", err)
	}
	if len(times) != 5 {
		t.Fatalf("NextN returned %d times, want 5", len(times))
	}
	prev := after
	for i, tt := range times {
		if !tt.After(after) {
			t.Errorf("times[%d]=%v is not strictly after %v", i, tt, after)
		}
		if !tt.After(prev) {
			t.Errorf("times not strictly increasing: %v then %v", prev, tt)
		}
		prev = tt
	}
}

func TestNextNRejectsZero(t *testing.T) {
	e, err := Parse("* * * * *")
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	_, after := utcPlus1(t)
	if _, err := e.NextN(after, 0); err == nil {
		t.Error("NextN(after, 0): expected an error")
	}
}