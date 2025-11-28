package reporter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_ExportCSV(t *testing.T) {
	t.Parallel()
	td := t.TempDir()
	out := filepath.Join(td, "out.csv")
	opts := Options{
		Directory:    filepath.Join("..", "..", "build"),
		Ticks:        false,
		Group:        false,
		Major:        false,
		Median:       false,
		Rotate:       false,
		OutputFormat: "csv",
		OutputFile:   out,
	}

	var b strings.Builder

	err := Run(&b, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	// file should exist and contain header
	data, err := os.ReadFile(out)
	if err != nil {
		t.Fatalf("read out csv: %v", err)
	}

	if !strings.Contains(string(data), "Name") && !strings.Contains(string(data), "Ver") {
		t.Fatalf("csv content does not contain expected header")
	}
}

func TestRun_ExportJSON(t *testing.T) {
	t.Parallel()
	td := t.TempDir()
	out := filepath.Join(td, "out.json")
	opts := Options{
		Directory:    filepath.Join("..", "..", "build"),
		Ticks:        false,
		Group:        false,
		Major:        false,
		Median:       false,
		Rotate:       false,
		OutputFormat: "json",
		OutputFile:   out,
	}

	var b strings.Builder

	err := Run(&b, opts)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	// file should exist
	_, err = os.Stat(out)
	if err != nil {
		t.Fatalf("json file not created: %v", err)
	}
}
