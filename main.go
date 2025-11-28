package main

import (
	"bytes"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/bavix/junit-reporter/internal/reporter"
)

const baselinePerm = 0o600

func main() {
	ticks := flag.Bool("ticks", false, "Time per ticks")
	group := flag.Bool("group", false, "Groups by version")
	major := flag.Bool("major", false, "Can only be used with a group")
	median := flag.Bool("median", false, "Median search")
	rotate := flag.Bool("rotate", false, "Swap versions and names")
	directory := flag.String("path", "./build", "Specify folder path")
	compare := flag.String("compare", "", "Path to baseline file to compare output against")
	generate := flag.String("generate-baseline", "", "Write current output to given file path and exit")

	flag.Parse()

	opts := reporter.Options{
		Directory:    *directory,
		Ticks:        *ticks,
		Group:        *group,
		Major:        *major,
		Median:       *median,
		Rotate:       *rotate,
		OutputFormat: "",
		OutputFile:   "",
	}

	const exitCodeMismatch = 2

	handled, err := handleCompareGenerate(*compare, *generate, opts, exitCodeMismatch)
	if err != nil {
		log.Fatalln(err)
	}

	if handled {
		return
	}

	err = reporter.Run(os.Stdout, opts)
	if err != nil {
		log.Fatalln(err)
	}
}

func handleCompareGenerate(compare, generate string, opts reporter.Options, exitCode int) (bool, error) {
	if generate != "" {
		out, err := runToBytes(opts)
		if err != nil {
			return true, err
		}

		err = writeBaseline(generate, out)
		if err != nil {
			return true, err
		}

		fmt.Fprintln(os.Stdout, "wrote baseline:", generate)

		return true, nil
	}

	if compare != "" {
		out, err := runToBytes(opts)
		if err != nil {
			return true, err
		}

		var match bool

		match, err = compareBaseline(compare, out)
		if err != nil {
			return true, err
		}

		if match {
			fmt.Fprintln(os.Stdout, "OK: output matches baseline")

			return true, nil
		}

		os.Exit(exitCode)
	}

	return false, nil
}

func runToBytes(opts reporter.Options) ([]byte, error) {
	var buf bytes.Buffer

	err := reporter.Run(&buf, opts)
	if err != nil {
		return nil, fmt.Errorf("run reporter: %w", err)
	}

	return bytes.TrimSpace(buf.Bytes()), nil
}

func writeBaseline(path string, out []byte) error {
	err := os.WriteFile(path, append(out, '\n'), baselinePerm)
	if err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}

	return nil
}

func compareBaseline(path string, out []byte) (bool, error) {
	wantBytes, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read baseline: %w", err)
	}

	want := bytes.TrimSpace(wantBytes)
	if bytes.Equal(want, out) {
		return true, nil
	}

	fmt.Fprintln(os.Stderr, "output mismatch vs baseline:", path)
	fmt.Fprintln(os.Stderr, "---- baseline ----")
	fmt.Fprintln(os.Stderr, string(want))
	fmt.Fprintln(os.Stderr, "---- got ----")
	fmt.Fprintln(os.Stderr, string(out))

	return false, nil
}
