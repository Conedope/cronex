package cronex

import (
	"reflect"
	"strings"
	"testing"
)

func fullRange(min, max int) []int {
	vals := make([]int, 0, max-min+1)
	for v := min; v <= max; v++ {
		vals = append(vals, v)
	}
	return vals
}

var (
	everyMinute = fullRange(0, 59)
	everyHour   = fullRange(0, 23)
	everyDom    = fullRange(1, 31)
	everyMonth  = fullRange(1, 12)
	everyDow    = fullRange(0, 7)
)

func TestParseFields(t *testing.T) {
	tests := []struct {
		expr   string
		minute []int
		hour   []int
		dom    []int
		month  []int
		dow    []int
	}{
		{"* * * * *", everyMinute, everyHour, everyDom, everyMonth, everyDow},
		{"*/15 * * * *", []int{0, 15, 30, 45}, everyHour, everyDom, everyMonth, everyDow},
		{"1,5,9 * * * *", []int{1, 5, 9}, everyHour, everyDom, everyMonth, everyDow},
		{"3-7 * * * *", []int{3, 4, 5, 6, 7}, everyHour, everyDom, everyMonth, everyDow},
		{"2-20/4 * * * *", []int{2, 6, 10, 14, 18}, everyHour, everyDom, everyMonth, everyDow},
		{"0 9 * * *", []int{0}, []int{9}, everyDom, everyMonth, everyDow},
		{"0 9 * * 1-5", []int{0}, []int{9}, everyDom, everyMonth, []int{1, 2, 3, 4, 5}},
		{"0 0 1 jan *", []int{0}, []int{0}, []int{1}, []int{1}, everyDow},
		{"0 0 * jan-mar *", []int{0}, []int{0}, everyDom, []int{1, 2, 3}, everyDow},
		{"0 0 * * mon-fri", []int{0}, []int{0}, everyDom, everyMonth, []int{1, 2, 3, 4, 5}},
		{"0 0 * * SUN", []int{0}, []int{0}, everyDom, everyMonth, []int{7}},
		{"0 0 * * 0", []int{0}, []int{0}, everyDom, everyMonth, []int{0}},
		{"0 0 * * 7", []int{0}, []int{0}, everyDom, everyMonth, []int{7}},
		{"0 0 * * sat-sun", []int{0}, []int{0}, everyDom, everyMonth, []int{6, 7}},
		{"15 14,18 * * *", []int{15}, []int{14, 18}, everyDom, everyMonth, everyDow},
		{"5 */2 * * *", []int{5}, []int{0, 2, 4, 6, 8, 10, 12, 14, 16, 18, 20, 22}, everyDom, everyMonth, everyDow},
		{"24 * * * *", []int{24}, everyHour, everyDom, everyMonth, everyDow},
		{"0 0 1,15 6,12 *", []int{0}, []int{0}, []int{1, 15}, []int{6, 12}, everyDow},
		{"0,0 * * * *", []int{0}, everyHour, everyDom, everyMonth, everyDow},
		{"5 * * * *", []int{5}, everyHour, everyDom, everyMonth, everyDow},
	}
	for _, tt := range tests {
		e, err := Parse(tt.expr)
		if err != nil {
			t.Fatalf("Parse(%q): unexpected error: %v", tt.expr, err)
		}
		if !reflect.DeepEqual(e.Minute.Values, tt.minute) {
			t.Errorf("Parse(%q) minute = %v, want %v", tt.expr, e.Minute.Values, tt.minute)
		}
		if !reflect.DeepEqual(e.Hour.Values, tt.hour) {
			t.Errorf("Parse(%q) hour = %v, want %v", tt.expr, e.Hour.Values, tt.hour)
		}
		if !reflect.DeepEqual(e.Dom.Values, tt.dom) {
			t.Errorf("Parse(%q) dom = %v, want %v", tt.expr, e.Dom.Values, tt.dom)
		}
		if !reflect.DeepEqual(e.Month.Values, tt.month) {
			t.Errorf("Parse(%q) month = %v, want %v", tt.expr, e.Month.Values, tt.month)
		}
		if !reflect.DeepEqual(e.Dow.Values, tt.dow) {
			t.Errorf("Parse(%q) dow = %v, want %v", tt.expr, e.Dow.Values, tt.dow)
		}
	}
}

func TestParseAliases(t *testing.T) {
	tests := []struct {
		alias, standard string
	}{
		{"@yearly", "0 0 1 1 *"},
		{"@annually", "0 0 1 1 *"},
		{"@monthly", "0 0 1 * *"},
		{"@weekly", "0 0 * * 0"},
		{"@daily", "0 0 * * *"},
		{"@midnight", "0 0 * * *"},
		{"@hourly", "0 * * * *"},
		{"@HOURLY", "0 * * * *"},
	}
	for _, tt := range tests {
		got, err := Parse(tt.alias)
		if err != nil {
			t.Fatalf("Parse(%q): unexpected error: %v", tt.alias, err)
		}
		want, err := Parse(tt.standard)
		if err != nil {
			t.Fatalf("Parse(%q): unexpected error: %v", tt.standard, err)
		}
		assertSameExpr(t, tt.alias, got, want)
	}
}

func assertSameExpr(t *testing.T, name string, got, want *Expr) {
	t.Helper()
	if !reflect.DeepEqual(got.Minute.Values, want.Minute.Values) ||
		!reflect.DeepEqual(got.Hour.Values, want.Hour.Values) ||
		!reflect.DeepEqual(got.Dom.Values, want.Dom.Values) ||
		!reflect.DeepEqual(got.Month.Values, want.Month.Values) ||
		!reflect.DeepEqual(got.Dow.Values, want.Dow.Values) {
		t.Errorf("%s: alias expansion mismatch:\n got minute=%v hour=%v dom=%v month=%v dow=%v\n want minute=%v hour=%v dom=%v month=%v dow=%v",
			name,
			got.Minute.Values, got.Hour.Values, got.Dom.Values, got.Month.Values, got.Dow.Values,
			want.Minute.Values, want.Hour.Values, want.Dom.Values, want.Month.Values, want.Dow.Values)
	}
}

func TestParseInvalid(t *testing.T) {
	tests := []struct {
		expr string
		want string // substring of the expected error
	}{
		{"60 * * * *", "minute"},
		{"0 24 * * *", "hour"},
		{"0 0 32 * *", "day-of-month"},
		{"0 0 * 13 *", "month"},
		{"0 0 * * 8", "day-of-week"},
		{"0 0 * * -1", "invalid range"},
		{"0 0 * * -2", "invalid range"},
		{"30-10 * * * *", "inverted range"},
		{"*/0 * * * *", "positive"},
		{"0 0 * * xyz", "invalid value"},
		{"a b c d e", "invalid value"},
		{"0 0 * * monday", "invalid value"},
		{"1,,3 * * * *", "empty list element"},
		{"0 0 *-1 * *", "invalid range"},
		{"0 0 * * * *", "5 fields"},
		{"0 0 * *", "5 fields"},
		{"", "empty"},
		{"@bogus", "unknown alias"},
		{"0 0 0 * *", "day-of-month"},
	}
	for _, tt := range tests {
		_, err := Parse(tt.expr)
		if err == nil {
			t.Fatalf("Parse(%q): expected error", tt.expr)
		}
		if tt.want != "" && !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Parse(%q) error %q, want substring %q", tt.expr, err.Error(), tt.want)
		}
	}
}

func TestParseValidEdgeBounds(t *testing.T) {
	// Boundary values that are legal on every field.
	for _, expr := range []string{
		"59 23 31 12 7",
		"0 0 1 1 0",
		"0 0 * * 7",
	} {
		if _, err := Parse(expr); err != nil {
			t.Errorf("Parse(%q): unexpected error: %v", expr, err)
		}
	}
}