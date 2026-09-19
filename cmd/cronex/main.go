// Command cronex is a thin command-line interface over the cronex library:
// it validates cron expressions, prints the next run times, expands the
// parsed fields, or describes an expression in plain English.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	_ "time/tzdata" // embed IANA zoneinfo so --tz works without host tzdata

	"github.com/Conedope/cronex"
)

const version = "0.1.0"

const atLayout = "2006-01-02 15:04:05"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cronex", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.Usage = func() {}
	n := fs.Int("n", 5, "number of next run times to print (default 5)")
	at := fs.String("at", "now", `reference time "2006-01-02 15:04:05" or "now" (default now)`)
	format := fs.String("format", time.RFC3339, "time layout for the printed run times")
	describe := fs.Bool("describe", false, "print a human description of the expression")
	parse := fs.Bool("parse", false, "dump the expanded field values")
	validate := fs.Bool("validate", false, "validate the expression (exit 0 valid, 1 invalid)")
	showVersion := fs.Bool("version", false, "print the version")
	tz := fs.String("tz", "", "time zone name (default: local)"+
		"; example: --tz Europe/Oslo")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printUsage(stdout)
			return 0
		}
		fmt.Fprintf(stderr, "cronex: %v\n", err)
		printUsage(stderr)
		return 2
	}

	if *showVersion {
		fmt.Fprintln(stdout, "cronex", version)
		return 0
	}

	if fs.NArg() != 1 {
		fmt.Fprintln(stderr, "cronex: exactly one cron expression argument is required")
		printUsage(stderr)
		return 2
	}
	expr := fs.Arg(0)

	loc, err := loadLocation(*tz)
	if err != nil {
		fmt.Fprintf(stderr, "cronex: %v\n", err)
		return 2
	}

	if *validate {
		if _, err := cronex.Parse(expr); err != nil {
			fmt.Fprintf(stderr, "cronex: %v\n", err)
			return 1
		}
		fmt.Fprintln(stdout, "valid")
		return 0
	}

	e, err := cronex.Parse(expr)
	if err != nil {
		fmt.Fprintf(stderr, "cronex: %v\n", err)
		return 1
	}

	switch {
	case *describe:
		fmt.Fprintln(stdout, e.Describe())
		return 0
	case *parse:
		dumpFields(stdout, e)
		return 0
	}

	after, err := resolveAt(*at, loc)
	if err != nil {
		fmt.Fprintf(stderr, "cronex: %v\n", err)
		return 2
	}
	if *n < 1 {
		fmt.Fprintf(stderr, "cronex: -n must be at least 1, got %d\n", *n)
		return 2
	}

	times, err := e.NextN(after, *n)
	if err != nil {
		fmt.Fprintf(stderr, "cronex: %v\n", err)
		return 1
	}
	for _, t := range times {
		fmt.Fprintln(stdout, t.Format(*format))
	}
	return 0
}

func loadLocation(name string) (*time.Location, error) {
	if name == "" || strings.EqualFold(name, "Local") || strings.EqualFold(name, "local") {
		return time.Local, nil
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, fmt.Errorf("invalid time zone %q: %v", name, err)
	}
	return loc, nil
}

func resolveAt(s string, loc *time.Location) (time.Time, error) {
	if s == "now" {
		return time.Now().In(loc), nil
	}
	t, err := time.ParseInLocation(atLayout, s, loc)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --at %q: %v", s, err)
	}
	return t, nil
}

func dumpFields(w io.Writer, e *cronex.Expr) {
	fmt.Fprintf(w, "minute: %s\n", joinSpace(e.Minute.Values))
	fmt.Fprintf(w, "hour: %s\n", joinSpace(e.Hour.Values))
	fmt.Fprintf(w, "dom: %s\n", joinSpace(e.Dom.Values))
	fmt.Fprintf(w, "month: %s\n", joinSpace(e.Month.Values))
	fmt.Fprintf(w, "dow: %s\n", joinSpace(e.Dow.Values))
}

func joinSpace(vals []int) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = strconv.Itoa(v)
	}
	return strings.Join(parts, " ")
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `cronex %s - a cron expression parser and scheduler

usage:
  cronex [flags] "cron-expression"

modes (default: print the next run times):
  --validate        exit 0 if valid, 1 if not
  --describe        print a human-readable description
  --parse           dump the expanded field values

flags:
  -n N              number of next run times (default 5)
  --at TIME         reference time "2006-01-02 15:04:05" or "now" (default now)
  --format LAYOUT   Go time layout for run times (default RFC3339)
  --tz ZONE         time zone name (default: local), e.g. Europe/Oslo
  --version         print the version
  --help            show this help

examples:
  cronex "0 0 * * *"
  cronex -n 10 "*/15 9-17 * * 1-5"
  cronex --at "2026-01-01 00:00:00" --format "2006-01-02 15:04:05" "@daily"
  cronex --describe "0 9 * * 1-5"
  cronex --parse "*/5 * * * *"
  cronex --validate "0 60 * * *"
`, version)
}