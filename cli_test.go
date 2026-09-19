package cronex_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var binPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "cronex-cli")
	if err != nil {
		panic(err)
	}
	binPath = filepath.Join(dir, "cronex")
	cmd := exec.Command("go", "build", "-o", binPath, "./cmd/cronex")
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("building cronex binary: " + err.Error() + "\n" + string(out))
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type result struct {
	stdout, stderr string
	code           int
}

func run(t *testing.T, args ...string) result {
	t.Helper()
	cmd := exec.Command(binPath, args...)
	var out, errb bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()
	code := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else {
			t.Fatalf("running cronex %v: %v", args, err)
		}
	}
	return result{stdout: out.String(), stderr: errb.String(), code: code}
}

func TestCLINextTimes(t *testing.T) {
	r := run(t, "--at", "2026-01-01 00:00:00", "--tz", "UTC",
		"--format", "2006-01-02 15:04:05", "-n", "3", "*/15 * * * *")
	if r.code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", r.code, r.stderr)
	}
	want := "2026-01-01 00:15:00\n2026-01-01 00:30:00\n2026-01-01 00:45:00\n"
	if r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

func TestCLINextBusinessHours(t *testing.T) {
	r := run(t, "--at", "2026-01-02 17:30:00", "--tz", "UTC",
		"--format", "2006-01-02 15:04:05", "-n", "2", "*/15 9-17 * * 1-5")
	if r.code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", r.code, r.stderr)
	}
	// 2026-01-02 is a Friday; 17:45 is still inside the working day.
	want := "2026-01-02 17:45:00\n2026-01-05 09:00:00\n"
	if r.stdout != want {
		t.Errorf("stdout = %q, want %q", r.stdout, want)
	}
}

func TestCLIDescribe(t *testing.T) {
	r := run(t, "--describe", "0 9 * * 1-5")
	if r.code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", r.code, r.stderr)
	}
	if r.stdout != "at minute 0, at hour 9, Monday through Friday\n" {
		t.Errorf("stdout = %q", r.stdout)
	}
}

func TestCLIParse(t *testing.T) {
	r := run(t, "--parse", "*/5 * * * *")
	if r.code != 0 {
		t.Fatalf("exit code = %d, stderr = %q", r.code, r.stderr)
	}
	lines := strings.Split(strings.TrimRight(r.stdout, "\n"), "\n")
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d: %q", len(lines), r.stdout)
	}
	if lines[0] != "minute: 0 5 10 15 20 25 30 35 40 45 50 55" {
		t.Errorf("minute line = %q", lines[0])
	}
	if lines[1] != "hour: 0 1 2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 19 20 21 22 23" {
		t.Errorf("hour line = %q", lines[1])
	}
	if lines[4] != "dow: 0 1 2 3 4 5 6 7" {
		t.Errorf("dow line = %q", lines[4])
	}
}

func TestCLIValidate(t *testing.T) {
	r := run(t, "--validate", "0 9 * * 1-5")
	if r.code != 0 || r.stdout != "valid\n" {
		t.Errorf("valid expr: code=%d stdout=%q stderr=%q", r.code, r.stdout, r.stderr)
	}

	r = run(t, "--validate", "0 60 * * *")
	if r.code != 1 {
		t.Errorf("invalid expr: code=%d, want 1", r.code)
	}
	if !strings.Contains(r.stderr, "60 out of range") {
		t.Errorf("invalid expr stderr = %q", r.stderr)
	}
}

func TestCLIInvalidExpression(t *testing.T) {
	r := run(t, "0 25 * * *")
	if r.code != 1 {
		t.Errorf("code=%d, want 1", r.code)
	}
	if !strings.Contains(r.stderr, "out of range") {
		t.Errorf("stderr = %q", r.stderr)
	}
}

func TestCLIBadFlags(t *testing.T) {
	r := run(t, "--bogus", "0 9 * * *")
	if r.code != 2 {
		t.Errorf("unknown flag: code=%d, want 2", r.code)
	}

	r = run(t)
	if r.code != 2 {
		t.Errorf("missing expression: code=%d, want 2", r.code)
	}

	r = run(t, "--at", "not-a-time", "0 9 * * *")
	if r.code != 2 {
		t.Errorf("bad --at: code=%d, want 2", r.code)
	}
}

func TestCLIVersion(t *testing.T) {
	r := run(t, "--version")
	if r.code != 0 || !strings.HasPrefix(r.stdout, "cronex ") {
		t.Errorf("code=%d stdout=%q", r.code, r.stdout)
	}
}

func TestCLIHelp(t *testing.T) {
	r := run(t, "--help")
	if r.code != 0 || !strings.Contains(r.stdout, "usage:") {
		t.Errorf("code=%d stdout=%q", r.code, r.stdout)
	}
}