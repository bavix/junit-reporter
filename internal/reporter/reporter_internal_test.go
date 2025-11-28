package reporter

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func baselinePath(name string) string {
	// relative from package dir -> project root
	return filepath.Join("..", "..", "build", "runs", name)
}

func readBaseline(t *testing.T, name string) string {
	t.Helper()

	b, err := os.ReadFile(baselinePath(name))
	if err != nil {
		t.Fatalf("read baseline %s: %v", name, err)
	}

	return strings.TrimSpace(string(b))
}

func runAndCapture(opts Options) string {
	var buf bytes.Buffer
	// tests run from package dir (internal/reporter), make directory point to project build
	if opts.Directory == "" || opts.Directory == "./build" {
		opts.Directory = filepath.Join("..", "..", "build")
	}

	_ = Run(&buf, opts) // errors are surfaced by tests comparing output

	return strings.TrimSpace(buf.String())
}

func TestRun_DefaultMatchesBaseline(t *testing.T) {
	t.Parallel()

	got := runAndCapture(Options{
		Directory:    "./build",
		Ticks:        false,
		Group:        false,
		Major:        false,
		Median:       false,
		Rotate:       false,
		OutputFormat: "",
		OutputFile:   "",
	})

	want := readBaseline(t, "run-default.txt")

	if got != want {
		t.Fatalf("default output mismatch\n--- want\n%s\n--- got\n%s\n", want, got)
	}
}

func TestRun_TicksMatchesBaseline(t *testing.T) {
	t.Parallel()

	got := runAndCapture(Options{
		Directory:    "./build",
		Ticks:        true,
		Group:        false,
		Major:        false,
		Median:       false,
		Rotate:       false,
		OutputFormat: "",
		OutputFile:   "",
	})

	want := readBaseline(t, "run-ticks.txt")

	if got != want {
		t.Fatalf("ticks output mismatch\n--- want\n%s\n--- got\n%s\n", want, got)
	}
}

func TestRun_RotateMatchesBaseline(t *testing.T) {
	t.Parallel()

	got := runAndCapture(Options{
		Directory:    "./build",
		Ticks:        false,
		Group:        false,
		Major:        false,
		Median:       false,
		Rotate:       true,
		OutputFormat: "",
		OutputFile:   "",
	})

	want := readBaseline(t, "run-rotate.txt")

	if got != want {
		t.Fatalf("rotate output mismatch\n--- want\n%s\n--- got\n%s\n", want, got)
	}
}

func TestRun_GroupMatchesBaseline(t *testing.T) {
	t.Parallel()

	got := runAndCapture(Options{
		Directory:    "./build",
		Ticks:        false,
		Group:        true,
		Major:        false,
		Median:       false,
		Rotate:       false,
		OutputFormat: "",
		OutputFile:   "",
	})

	want := readBaseline(t, "run-group.txt")

	if got != want {
		t.Fatalf("group output mismatch\n--- want\n%s\n--- got\n%s\n", want, got)
	}
}

func TestRun_GroupMajorMatchesBaseline(t *testing.T) {
	t.Parallel()

	got := runAndCapture(Options{
		Directory:    "./build",
		Ticks:        false,
		Group:        true,
		Major:        true,
		Median:       false,
		Rotate:       false,
		OutputFormat: "",
		OutputFile:   "",
	})

	want := readBaseline(t, "run-group-major.txt")

	if got != want {
		t.Fatalf("group-major output mismatch\n--- want\n%s\n--- got\n%s\n", want, got)
	}
}

func TestRun_MedianMatchesBaseline(t *testing.T) {
	t.Parallel()

	got := runAndCapture(Options{
		Directory:    "./build",
		Ticks:        false,
		Group:        false,
		Major:        false,
		Median:       true,
		Rotate:       false,
		OutputFormat: "",
		OutputFile:   "",
	})

	want := readBaseline(t, "run-median.txt")

	if got != want {
		t.Fatalf("median output mismatch\n--- want\n%s\n--- got\n%s\n", want, got)
	}
}
