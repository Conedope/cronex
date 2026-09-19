// Package cronex parses, matches, and describes classic 5-field cron
// expressions. It supports step values, ranges, lists, named months and
// weekdays, and the common "@" aliases. It is implemented with the Go
// standard library only.
package cronex

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// aliases maps the common @ shorthand expressions to their standard
// 5-field equivalents.
//
//	@yearly / @annually -> 0 0 1 1 *
//	@monthly           -> 0 0 1 * *
//	@weekly            -> 0 0 * * 0
//	@daily / @midnight -> 0 0 * * *
//	@hourly            -> 0 * * * *
var aliases = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// Field is an expanded set of allowed values for one cron field, plus
// parsing metadata recording whether the source token was a bare "*" and
// whether it used an explicit step.
type Field struct {
	Values []int

	star bool
	step int
}

// Expr is a parsed cron expression. Minute, Hour, Dom, Month and Dow hold
// the expanded allowed values for each of the five standard fields.
type Expr struct {
	Minute, Hour, Dom, Month, Dow Field

	spec string
}

// fieldKind identifies which cron field is being parsed so that the correct
// bounds and name tables apply.
type fieldKind int

const (
	fminute fieldKind = iota
	fhour
	fdom
	fmonth
	fdow
)

type fieldDesc struct {
	name  string
	min   int
	max   int
	names map[string]int
}

var fieldDescs = map[fieldKind]fieldDesc{
	fminute: {name: "minute", min: 0, max: 59},
	fhour:   {name: "hour", min: 0, max: 23},
	fdom:    {name: "day-of-month", min: 1, max: 31},
	fmonth:  {name: "month", min: 1, max: 12, names: monthNameValues},
	fdow:    {name: "day-of-week", min: 0, max: 7, names: dowNameValues},
}

// monthNameValues maps the three-letter English month abbreviations
// (case-insensitive) to their month numbers.
var monthNameValues = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4, "may": 5, "jun": 6,
	"jul": 7, "aug": 8, "sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// dowNameValues maps the three-letter English weekday abbreviations
// (case-insensitive) to their weekday numbers. Sunday is additionally
// accepted as 7 so that range expansion like "sat-sun" == 6-7 works.
var dowNameValues = map[string]int{
	"sun": 7, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6,
}

// Parse parses a cron expression into an Expr. It accepts the standard
// five fields (minute hour day-of-month month day-of-week) separated by
// whitespace, plus the @yearly/@annually, @monthly, @weekly, @daily,
// @midnight and @hourly aliases.
//
// Each field supports:
//
//	*           every value
//	*/n         every n-th value from the field minimum
//	1,5,9       a list of values
//	3-7         an inclusive range
//	2-20/4      a range with a step
//	5           a single value
//
// Minute ranges are 0-59, hour 0-23, day-of-month 1-31, month 1-12, and
// day-of-week 0-7 (where both 0 and 7 mean Sunday). Months (jan-dec) and
// weekdays (mon-sun) may be given by their explicit three-letter short
// names, case-insensitively. Inverted ranges, out-of-range values, a zero
// step, and unrecognized tokens produce descriptive errors.
func Parse(s string) (*Expr, error) {
	input := strings.TrimSpace(s)
	if input == "" {
		return nil, fmt.Errorf("empty cron expression")
	}
	fields := strings.Fields(input)
	if len(fields) == 1 && strings.HasPrefix(fields[0], "@") {
		standard, ok := aliases[strings.ToLower(fields[0])]
		if !ok {
			return nil, fmt.Errorf("unknown alias %q", fields[0])
		}
		fields = strings.Fields(standard)
	}
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields, got %d: %q", len(fields), input)
	}

	e := &Expr{spec: input}
	var err error
	if e.Minute, err = parseField(fields[0], fminute); err != nil {
		return nil, fmt.Errorf("minute field: %v", err)
	}
	if e.Hour, err = parseField(fields[1], fhour); err != nil {
		return nil, fmt.Errorf("hour field: %v", err)
	}
	if e.Dom, err = parseField(fields[2], fdom); err != nil {
		return nil, fmt.Errorf("day-of-month field: %v", err)
	}
	if e.Month, err = parseField(fields[3], fmonth); err != nil {
		return nil, fmt.Errorf("month field: %v", err)
	}
	if e.Dow, err = parseField(fields[4], fdow); err != nil {
		return nil, fmt.Errorf("day-of-week field: %v", err)
	}
	return e, nil
}

func parseField(token string, kind fieldKind) (Field, error) {
	fd := fieldDescs[kind]
	var f Field

	for _, elem := range strings.Split(token, ",") {
		if elem == "" {
			return Field{}, fmt.Errorf("empty list element in %q", token)
		}
		vals, star, step, err := parseElem(elem, kind)
		if err != nil {
			return Field{}, err
		}
		if star {
			f.star = true
		}
		if step > 0 {
			f.step = step
		}
		f.Values = append(f.Values, vals...)
	}

	sort.Ints(f.Values)
	f.Values = dedupe(f.Values)
	if len(f.Values) == 0 {
		return Field{}, fmt.Errorf("field %q expands to no values", fd.name)
	}
	return f, nil
}

// parseElem parses a single comma-delimited element such as "*", "*/5",
// "3-7", "2-20/4" or "5".
func parseElem(elem string, kind fieldKind) ([]int, bool, int, error) {
	var rng string
	var step int
	slashed := strings.Split(elem, "/")
	switch len(slashed) {
	case 1:
		rng = elem
	case 2:
		rng = slashed[0]
		n, err := strconv.Atoi(slashed[1])
		if err != nil {
			return nil, false, 0, fmt.Errorf("invalid step %q in %q", slashed[1], elem)
		}
		if n <= 0 {
			return nil, false, 0, fmt.Errorf("step must be positive, got %d", n)
		}
		step = n
	default:
		return nil, false, 0, fmt.Errorf("invalid element %q", elem)
	}

	lo, hi, star, err := rangeBounds(rng, kind)
	if err != nil {
		return nil, false, 0, err
	}
	explicit := step
	if star && explicit > 0 {
		star = false // "*/n" is restricted even though its range is "*"
	}
	eff := explicit
	if eff == 0 {
		eff = 1
	}

	var vals []int
	for v := lo; v <= hi; v += eff {
		vals = append(vals, v)
	}
	return vals, star, explicit, nil
}

// rangeBounds resolves a range token ("*", "3-7", "jan-mar", "5") to its
// inclusive low and high bounds.
func rangeBounds(rng string, kind fieldKind) (int, int, bool, error) {
	fd := fieldDescs[kind]
	if rng == "*" {
		return fd.min, fd.max, true, nil
	}

	if strings.Contains(rng, "-") {
		parts := strings.Split(rng, "-")
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" || parts[0] == "*" || parts[1] == "*" {
			return 0, 0, false, fmt.Errorf("invalid range %q", rng)
		}
		lo, err := parseValue(parts[0], kind)
		if err != nil {
			return 0, 0, false, err
		}
		hi, err := parseValue(parts[1], kind)
		if err != nil {
			return 0, 0, false, err
		}
		if lo > hi {
			return 0, 0, false, fmt.Errorf("inverted range %d-%d", lo, hi)
		}
		return lo, hi, false, nil
	}

	v, err := parseValue(rng, kind)
	if err != nil {
		return 0, 0, false, err
	}
	return v, v, false, nil
}

// parseValue resolves a single value, which may be numeric or a name.
func parseValue(v string, kind fieldKind) (int, error) {
	fd := fieldDescs[kind]
	if name, ok := fd.names[strings.ToLower(v)]; ok {
		return name, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", v)
	}
	if n < fd.min || n > fd.max {
		return 0, fmt.Errorf("value %d out of range (%d-%d)", n, fd.min, fd.max)
	}
	return n, nil
}

func dedupe(vals []int) []int {
	out := vals[:0]
	for i, v := range vals {
		if i == 0 || vals[i-1] != v {
			out = append(out, v)
		}
	}
	return out
}

// Next returns the earliest time strictly after after that matches the
// expression, in after's location. If no such time exists (within the
// five-year scan window) the zero time.Time is returned.
func (e *Expr) Next(after time.Time) time.Time {
	t, err := e.NextErr(after)
	if err != nil {
		return time.Time{}
	}
	return t
}

const maxScanYears = 5

// NextErr is like Next but reports an error when no matching time can be
// found within maxScanYears of after. Impossible expressions such as
// "0 0 30 2 *" (30 February) therefore fail rather than loop forever.
func (e *Expr) NextErr(after time.Time) (time.Time, error) {
	loc := after.Location()
	base := time.Date(after.Year(), after.Month(), after.Day(), after.Hour(), after.Minute(), 0, 0, loc)
	t := base.Add(1 * time.Minute)
	limit := after.AddDate(maxScanYears, 0, 0)

	for t.Before(limit) {
		if e.matches(t) {
			return t, nil
		}
		if !contains(e.Month, int(t.Month())) {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, loc)
			continue
		}
		if !e.dayMatches(t) {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			continue
		}
		if !contains(e.Hour, t.Hour()) {
			next := nextAfter(e.Hour, t.Hour())
			if next < 0 {
				t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, loc)
			} else {
				t = time.Date(t.Year(), t.Month(), t.Day(), next, 0, 0, 0, loc)
			}
			continue
		}
		t = t.Add(1 * time.Minute)
	}
	return time.Time{}, fmt.Errorf("no matching time for %q within 5 years of %s", e.spec, after.Format(time.RFC3339))
}

// nextAfter returns the smallest value in f strictly greater than v, or -1
// when no such value exists.
func nextAfter(f Field, v int) int {
	best := -1
	for _, x := range f.Values {
		if x > v && (best < 0 || x < best) {
			best = x
		}
	}
	return best
}

// NextN returns the next n matching times strictly after after, in ascending
// order.
func (e *Expr) NextN(after time.Time, n int) ([]time.Time, error) {
	if n < 1 {
		return nil, fmt.Errorf("NextN: n must be positive, got %d", n)
	}
	times := make([]time.Time, 0, n)
	cur := after
	for i := 0; i < n; i++ {
		t, err := e.NextErr(cur)
		if err != nil {
			return nil, err
		}
		times = append(times, t)
		cur = t
	}
	return times, nil
}

// matches reports whether t satisfies every field of the expression.
func (e *Expr) matches(t time.Time) bool {
	return contains(e.Minute, t.Minute()) &&
		contains(e.Hour, t.Hour()) &&
		e.dayMatches(t) &&
		contains(e.Month, int(t.Month()))
}

// dayMatches applies classic cron semantics: when both day-of-month and
// day-of-week are restricted the command runs when either matches; when one
// is unrestricted it is ignored entirely.
func (e *Expr) dayMatches(t time.Time) bool {
	domM := contains(e.Dom, t.Day())
	dowM := dowMatches(e.Dow, t)
	switch {
	case e.Dom.star && e.Dow.star:
		return true
	case e.Dom.star:
		return dowM
	case e.Dow.star:
		return domM
	default:
		return domM || dowM
	}
}

func contains(f Field, v int) bool {
	if f.star {
		return true
	}
	for _, x := range f.Values {
		if x == v {
			return true
		}
	}
	return false
}

// dowMatches matches the day-of-week field, honouring the equivalence of 0
// and 7 (both Sunday).
func dowMatches(f Field, t time.Time) bool {
	if f.star {
		return true
	}
	wd := int(t.Weekday())
	for _, x := range f.Values {
		if x == wd || (wd == 0 && x == 7) {
			return true
		}
	}
	return false
}