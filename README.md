# cronex

A cron expression parser and scheduler for Go, implemented with the
standard library only (`time`, `strconv`, `fmt`). It validates classic
5-field cron expressions, expands ranges and lists, computes the next run
times from a reference time, and ships with a thin CLI.

```
module github.com/Conedope/cronex   (requires Go >= 1.22)
```

## Grammar

An expression has five whitespace-separated fields:

| Field        | Allowed values | Names |
|--------------|----------------|-------|
| `minute`     | 0-59           |       |
| `hour`       | 0-23           |       |
| day-of-month | 1-31           |       |
| month        | 1-12           | `jan`–`dec` (case-insensitive) |
| day-of-week  | 0-7            | `mon`–`sun` (case-insensitive) |

Day-of-week `0` and `7` both mean Sunday. The three-letter month/day names
are accepted case-insensitively anywhere a numeric value is legal, e.g.
`jan-mar`, `mon-fri`, `SAT-SUN`.

Each field supports:

| Syntax     | Meaning                                    | Example          |
|------------|--------------------------------------------|------------------|
| `*`        | every value                                | `*`              |
| `*/n`      | every n-th value from the field minimum    | `*/15`           |
| `a-b`      | inclusive range                            | `9-17`           |
| `a-b/n`    | range with a step                          | `2-20/4`         |
| `a,b,c`    | list of values/ranges/steps                | `0,30`           |
| `a`        | a single value                             | `5`              |

Inverted ranges, out-of-range values, a zero step, unknown tokens, and a
wrong field count are rejected with descriptive errors.

### Aliases

`Parse`, and therefore the CLI, accepts these shorthands:

| Alias            | Expands to      |
|------------------|-----------------|
| `@yearly`, `@annually` | `0 0 1 1 *` |
| `@monthly`       | `0 0 1 * *`     |
| `@weekly`        | `0 0 * * 0`     |
| `@daily`, `@midnight` | `0 0 * * *` |
| `@hourly`        | `0 * * * *`     |

### day-of-month / day-of-week semantics

Classic (Vixie) cron semantics apply: when **both** `day-of-month` and
`day-of-week` are restricted (neither is `*`), a day matches if **either**
field matches. When one of them is `*`, that field is ignored. For example
`0 0 13 * 5` runs at midnight on the 13th **and** on every Friday.

### Time zones

`Next`/`NextErr`/`NextN` always operate in the location of the `after`
argument and return times in that same location, so DST-aware zones work
correctly. The tests use `time.FixedZone` / UTC so expectations stay
deterministic. The CLI defaults to the local zone and can be overridden
with `--tz` (IANA names such as `Europe/Oslo`; the CLI embeds the `tzdata`
package so named zones work even without host zoneinfo files).

## Library API

```go
import "github.com/Conedope/cronex"

e, err := cronex.Parse("*/15 9-17 * * 1-5")      // every quarter-hour, workdays
next, err := e.NextErr(time.Now())                // earliest match strictly after
times, err := e.NextN(time.Now(), 10)             // next 10 run times, ascending
valid, err := cronex.Parse("0 60 * * *")          // err: value 60 out of range
e.Describe()                                       // "at minute 0, at hour 9, Monday through Friday"
```

- `Expr` exposes `Minute`, `Hour`, `Dom`, `Month`, `Dow` as `Field`
  structs holding the sorted, expanded `Values []int`.
- `Next(after)` returns the earliest matching `time.Time` strictly after
  `after`, or the zero time if none exists; `NextErr` returns the error
  instead. Impossible expressions (e.g. `0 0 30 2 *`, 30 February) fail
  after the five-year scan window rather than looping forever.
- `NextN(after, n)` returns `n` matches, strictly increasing and all
  strictly after `after`.

## CLI

```
cronex [flags] "cron-expression"
```

| Flag            | Default                | Meaning |
|-----------------|------------------------|---------|
| `-n N`          | `5`                    | number of next run times |
| `--at TIME`     | `now`                  | reference time `2006-01-02 15:04:05` |
| `--format LAYOUT` | RFC3339             | Go time layout for printed run times |
| `--tz ZONE`     | local                  | time zone, e.g. `Europe/Oslo` |
| `--describe`    |                        | print a human description |
| `--parse`       |                        | dump expanded field values |
| `--validate`    |                        | exit 0 valid / 1 invalid |
| `--version`, `--help` |                  |                       |

Exit codes: `0` success, `1` invalid expression, `2` bad flags/arguments.

### Verified examples

```
$ cronex "*/5 * * * *"
2026-09-19T05:45:00Z
2026-09-19T05:50:00Z
2026-09-19T05:55:00Z
2026-09-19T06:00:00Z
2026-09-19T06:05:00Z
```

```
$ cronex --at "2026-01-01 00:00:00" --format "2006-01-02 15:04:05" -n 4 "*/15 9-17 * * 1-5"
2026-01-01 09:00:00
2026-01-01 09:15:00
2026-01-01 09:30:00
2026-01-01 09:45:00
```

```
$ cronex --at "2026-01-01 00:00:00" --format "2006-01-02 15:04:05" -n 4 "0 0 13 * 5"
2026-01-02 00:00:00
2026-01-09 00:00:00
2026-01-13 00:00:00
2026-01-16 00:00:00
```

```
$ cronex --describe "0 9 * * 1-5"
at minute 0, at hour 9, Monday through Friday

$ cronex --parse "*/5 * * * *"
minute: 0 5 10 15 20 25 30 35 40 45 50 55
hour: 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23
dom: 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23 24 25 26 27 28 29 30 31
month: 1 2 3 4 5 6 7 8 9 10 11 12
dow: 0 1 2 3 4 5 6 7
```

```
$ cronex --validate "0 9 * * 1-5"
valid

$ cronex --validate "0 60 * * *"; echo "exit=$?"
cronex: hour field: value 60 out of range (0-23)
exit=1
```

```
$ cronex --tz Europe/Oslo --at "2026-01-01 00:00:00" --format "2006-01-02 15:04:05 MST" -n 2 "@daily"
2026-01-02 00:00:00 CET
2026-01-03 00:00:00 CET
```

## Building and testing

```
CGO_ENABLED=0 go vet ./...
CGO_ENABLED=0 go test ./...
go build -o cronex ./cmd/cronex
```

## License

MIT — see [LICENSE](LICENSE). © 2026 Conedope.