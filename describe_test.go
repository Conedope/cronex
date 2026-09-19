package cronex

import "testing"

func TestDescribe(t *testing.T) {
	tests := []struct {
		expr string
		want string
	}{
		{"* * * * *", "every minute"},
		{"*/5 * * * *", "every 5 minutes"},
		{"*/30 * * * *", "every 30 minutes"},
		{"0 * * * *", "at minute 0 of every hour"},
		{"0 9 * * *", "at minute 0, at hour 9, every day"},
		{"0 9 * * 1-5", "at minute 0, at hour 9, Monday through Friday"},
		{"30 8 * * 1", "at minute 30, at hour 8, Monday"},
		{"0 0 13 * 5", "at minute 0, at hour 0, on day 13 or Friday"},
		{"0 0 1 1 *", "at minute 0, at hour 0, on day 1, in January"},
		{"0 0 * * 0", "at minute 0, at hour 0, Sunday"},
		{"0 0 * * 7", "at minute 0, at hour 0, Sunday"},
		{"15 14 1 * *", "at minute 15, at hour 14, on day 1"},
		{"0 12 * 1-3 *", "at minute 0, at hour 12, in January through March"},
		{"1,5,9 * * * *", "at minutes 1, 5, 9"},
		{"0 0 1-5 * *", "at minute 0, at hour 0, on days 1 through 5"},
		{"0 0 * * sat-sun", "at minute 0, at hour 0, Saturday through Sunday"},
		{"0 0 15 6,12 *", "at minute 0, at hour 0, on day 15, in June, December"},
	}
	for _, tt := range tests {
		e, err := Parse(tt.expr)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.expr, err)
		}
		if got := e.Describe(); got != tt.want {
			t.Errorf("Describe(%q) = %q, want %q", tt.expr, got, tt.want)
		}
	}
}

func TestDescribeAliases(t *testing.T) {
	// Aliases describe exactly like their standard expansions.
	tests := []struct {
		alias string
		want  string
	}{
		{"@hourly", "at minute 0 of every hour"},
		{"@daily", "at minute 0, at hour 0, every day"},
		{"@weekly", "at minute 0, at hour 0, Sunday"},
		{"@monthly", "at minute 0, at hour 0, on day 1"},
		{"@yearly", "at minute 0, at hour 0, on day 1, in January"},
	}
	for _, tt := range tests {
		e, err := Parse(tt.alias)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tt.alias, err)
		}
		if got := e.Describe(); got != tt.want {
			t.Errorf("Describe(%q) = %q, want %q", tt.alias, got, tt.want)
		}
	}
}