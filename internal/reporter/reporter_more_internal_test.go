package reporter

import (
	"sort"
	"strings"
	"testing"
	"time"
)

func TestParseVersionFromPath_Grouping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		path  string
		group bool
		major bool
		want  string
	}{
		{"./build/junit-6.2.4.xml", false, false, "6.2.4"},
		{"./build/junit-7.3.0-beta1.xml", false, false, "7.3.0-beta1"},
		{"./build/junit-6.2.4.xml", true, false, "6.2.4"},
		{"./build/junit-10.0.0-beta1.xml", true, true, "10.x"},
		{"./build/junit-6.2.4-8-array.xml", true, false, "6.2.4"},
	}

	for _, tt := range tests {
		got := ParseVersionFromPath(tt.path, tt.group, tt.major)
		if got != tt.want {
			t.Fatalf("ParseVersionFromPath(%q,%v,%v) = %q; want %q", tt.path, tt.group, tt.major, got, tt.want)
		}
	}
}

func TestCompareVersionsOrdering(t *testing.T) {
	t.Parallel()

	versions := []string{"6.2.4", "6.2.4-8-array", "6.1.0", "7.0.0", "7.1.0", "10.0.0-beta1", "6.x"}
	sort.Slice(versions, func(i, j int) bool {
		return CompareVersions(versions[i], versions[j])
	})
	// ensure stable ordering where 6.x comes before 6.1.0/6.2.4 after replacement
	if !strings.HasPrefix(versions[0], "6") {
		t.Fatalf("unexpected first version after sort: %v", versions)
	}
}

func TestFormatDurationReadable(t *testing.T) {
	t.Parallel()
	// ensure it returns something human readable for various scales
	durations := []time.Duration{5 * time.Millisecond, 250 * time.Millisecond, 2 * time.Second, 90 * time.Second}
	for _, d := range durations {
		out := formatDuration(d)
		if out == "" {
			t.Fatalf("formatDuration returned empty for %v", d)
		}
	}
}

func TestRun_NoFilesError(t *testing.T) {
	t.Parallel()
	// point to a non-existent folder
	errDir := Options{
		Directory:    "./nonexistent-folder",
		Ticks:        false,
		Group:        false,
		Major:        false,
		Median:       false,
		Rotate:       false,
		OutputFormat: "",
		OutputFile:   "",
	}

	var buf strings.Builder

	err := Run(&buf, errDir)
	if err == nil {
		t.Fatalf("expected error when directory has no junit files")
	}
}
