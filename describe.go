package cronex

import (
	"fmt"
	"strconv"
	"strings"
)

var describeMonthNames = []string{
	"", "January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}
var describeDowNames = []string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday",
	"Friday", "Saturday", "Sunday",
}

// Describe returns a concise human-readable summary of the expression.
// Examples: "every 5 minutes", "at minute 0, at hour 9, Monday through
// Friday", "at minute 0, at hour 0, on day 13 or Friday".
func (e *Expr) Describe() string {
	if e.Minute.star && e.Hour.star && e.Dom.star && e.Month.star && e.Dow.star {
		return "every minute"
	}
	if e.Minute.step > 0 && e.Hour.star && e.Dom.star && e.Month.star && e.Dow.star {
		return fmt.Sprintf("every %d minutes", e.Minute.step)
	}
	if len(e.Minute.Values) == 1 && e.Hour.star && e.Dom.star && e.Month.star && e.Dow.star {
		return fmt.Sprintf("at minute %d of every hour", e.Minute.Values[0])
	}

	var parts []string

	// Minute part.
	if e.Minute.star {
		parts = append(parts, "every minute")
	} else if e.Minute.step > 0 && len(e.Minute.Values) > 1 {
		parts = append(parts, fmt.Sprintf("every %d minutes", e.Minute.step))
	} else if len(e.Minute.Values) == 1 {
		parts = append(parts, fmt.Sprintf("at minute %d", e.Minute.Values[0]))
	} else {
		parts = append(parts, fmt.Sprintf("at minutes %s", joinInts(e.Minute.Values)))
	}

	// Hour part.
	if !e.Hour.star {
		if len(e.Hour.Values) == 1 {
			parts = append(parts, fmt.Sprintf("at hour %d", e.Hour.Values[0]))
		} else {
			parts = append(parts, fmt.Sprintf("at hours %s", joinInts(e.Hour.Values)))
		}
	}

	// Day and month parts.
	var day []string
	switch {
	case e.Dom.star && e.Dow.star:
		// "every day" is only informative once minute and hour are pinned.
		if e.Month.star && !e.Minute.star && !e.Hour.star {
			day = append(day, "every day")
		}
	case e.Dow.star:
		day = append(day, describeDom(e.Dom.Values))
	case e.Dom.star:
		day = append(day, describeDow(e.Dow.Values))
	default:
		day = append(day, describeDom(e.Dom.Values)+" or "+describeDow(e.Dow.Values))
	}
	if !e.Month.star {
		day = append(day, "in "+describeMonth(e.Month.Values))
	}
	parts = append(parts, day...)

	return strings.Join(parts, ", ")
}

func describeDom(vals []int) string {
	if len(vals) == 1 {
		return fmt.Sprintf("on day %d", vals[0])
	}
	if isContiguous(vals) {
		return fmt.Sprintf("on days %d through %d", vals[0], vals[len(vals)-1])
	}
	return fmt.Sprintf("on days %s", joinInts(vals))
}

func describeDow(vals []int) string {
	names := dowNames(vals)
	if len(names) == 1 {
		return names[0]
	}
	if isContiguous(vals) {
		return names[0] + " through " + names[len(names)-1]
	}
	return strings.Join(names, ", ")
}

func describeMonth(vals []int) string {
	if len(vals) == 1 {
		return describeMonthNames[vals[0]]
	}
	if isContiguous(vals) {
		return describeMonthNames[vals[0]] + " through " + describeMonthNames[vals[len(vals)-1]]
	}
	names := make([]string, len(vals))
	for i, v := range vals {
		names[i] = describeMonthNames[v]
	}
	return strings.Join(names, ", ")
}

func dowNames(vals []int) []string {
	var names []string
	prev := -99
	for _, v := range vals {
		if prev == 0 && v == 7 {
			continue // 7 and 0 are the same weekday (Sunday)
		}
		prev = v
		names = append(names, describeDowNames[v])
	}
	return names
}

func isContiguous(vals []int) bool {
	if len(vals) < 2 {
		return false
	}
	for i := 1; i < len(vals); i++ {
		if vals[i] != vals[i-1]+1 {
			return false
		}
	}
	return true
}

func joinInts(vals []int) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, ", ")
}