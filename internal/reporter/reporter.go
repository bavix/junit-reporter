package reporter

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/joshdk/go-junit"
	"github.com/montanaflynn/stats"
	"github.com/olekukonko/tablewriter"
	"github.com/olekukonko/tablewriter/tw"
)

const (
	roundPrecision = 100
	roundBase      = 100
)

// Precompiled regex for extracting versions from filenames.
var (
	versionRE = regexp.MustCompile(`\d+\.((\d+|x)(\.(\d+|x))?)`)
	junitRE   = regexp.MustCompile(`junit-(.+).xml`)
)

type Options struct {
	Directory    string
	Ticks        bool
	Group        bool
	Major        bool
	Median       bool
	Rotate       bool
	OutputFormat string
	OutputFile   string
}

type unit struct {
	Class  string
	Method string
	t      []uTest
}

type uTest struct {
	Ver   string
	JUnit junit.Test
}

var (
	ErrDash              = errors.New("-")
	ErrFilesNotFound     = errors.New("files not found")
	ErrUnsupportedFormat = errors.New("unsupported output format")
)

func (u *unit) FullName() string {
	return strings.TrimSuffix(u.Class, "Test") + ":" + strings.TrimPrefix(u.Method, "test")
}

func (u *unit) Push(ver string, t junit.Test) {
	u.t = append(u.t, uTest{Ver: ver, JUnit: t})
}

func formatDuration(dur time.Duration) string {
	scale := roundBase * time.Second
	for scale > dur {
		scale /= 10
	}

	return dur.Round(scale / roundPrecision).String()
}

func (u *unit) GetDuration(ver string, ticks bool, median bool) (time.Duration, error) {
	var results []time.Duration

	for _, testCase := range u.t {
		if testCase.Ver == ver {
			if testCase.JUnit.Status != "passed" {
				return 0, ErrDash
			}

			results = append(results, testCase.JUnit.Duration)
		}
	}

	if len(results) == 0 {
		return 0, ErrDash
	}

	if ticks {
		if median {
			return u.getDurationMedian(results), nil
		}

		return u.getDurationAverage(results), nil
	}

	return u.getDurationSum(results), nil
}

func (u *unit) getDurationSum(input []time.Duration) time.Duration {
	durTotal := time.Duration(0)
	for _, i := range input {
		durTotal += i
	}

	return durTotal
}

func (u *unit) getDurationAverage(input []time.Duration) time.Duration {
	return time.Duration(int64(u.getDurationSum(input)) / int64(len(input)))
}

func (u *unit) getDurationMedian(input []time.Duration) time.Duration {
	results := make([]float64, 0, len(input))
	for _, i := range input {
		results = append(results, float64(i))
	}

	median, err := stats.Median(results)
	if err != nil {
		return 0
	}

	return time.Duration(int64(median))
}

func newUnit(ver string, t junit.Test) unit {
	namespaces := strings.Split(t.Classname, ".")
	className := namespaces[len(namespaces)-1]
	method := strings.Fields(t.Name)[0]
	ut := uTest{Ver: ver, JUnit: t}

	return unit{Class: className, Method: method, t: []uTest{ut}}
}

func depthSuite(suite junit.Suite) []junit.Test {
	tests := suite.Tests
	for _, one := range suite.Suites {
		tests = append(tests, depthSuite(one)...)
	}

	return tests
}

// ParseVersionFromPath extracts the version string from a filename path using the same
// rules as Run: when group==true it extracts numeric version-like pattern, optionally
// collapsing to major.x when major==true. When group==false it extracts the substring
// matched by `junit-(.+).xml`.
func ParseVersionFromPath(_path string, group bool, major bool) string {
	if group {
		ver := string(versionRE.Find([]byte(_path)))
		if major {
			parts := strings.Split(ver, ".")
			if len(parts) > 0 {
				return parts[0] + ".x"
			}
		}

		return ver
	}

	m := junitRE.FindStringSubmatch(_path)
	if len(m) > 1 {
		return m[1]
	}

	return ""
}

func discoverJUnitFiles(dir string) ([]string, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var filenames []string

	for _, info := range files {
		if info.Type().IsRegular() && strings.HasSuffix(info.Name(), ".xml") && strings.HasPrefix(info.Name(), "junit-") {
			filenames = append(filenames, path.Join(dir, info.Name()))
		}
	}

	if len(filenames) == 0 {
		return nil, fmt.Errorf("%w: %s", ErrFilesNotFound, dir)
	}

	return filenames, nil
}

func ingestFilesToUnits(filenames []string, opts Options) (map[string]*unit, []string, error) {
	units := map[string]*unit{}
	verKeys := map[string]bool{}

	var versions []string

	for _, filePath := range filenames {
		ingestFile, err := junit.IngestFile(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to ingest JUnit xml %s: %w", filePath, err)
		}

		var ver string
		if opts.Group {
			ver = string(versionRE.Find([]byte(filePath)))
			if opts.Major {
				ver = strings.Split(ver, ".")[0] + ".x"
			}
		} else {
			m := junitRE.FindStringSubmatch(filePath)
			if len(m) > 1 {
				ver = m[1]
			}
		}

		if _, ok := verKeys[ver]; !ok {
			versions = append(versions, ver)
			verKeys[ver] = true
		}

		for _, suite := range ingestFile {
			for _, test := range depthSuite(suite) {
				unitVal := newUnit(ver, test)

				if elem, ok := units[unitVal.FullName()]; ok {
					elem.Push(ver, test)

					continue
				}

				units[unitVal.FullName()] = &unitVal
			}
		}
	}

	return units, versions, nil
}

func buildTableData(units map[string]*unit, versions []string, opts Options) ([]string, [][]string) {
	columns := []string{}

	unitList := make([]string, 0, len(units))

	for _, unitVal := range units {
		unitList = append(unitList, unitVal.FullName())
	}

	slices.Sort(unitList)

	rows := [][]string{}

	if opts.Rotate {
		columns = append(columns, "Ver")
		columns = append(columns, unitList...)

		for _, ver := range versions {
			values := make([]string, 0, 1+len(unitList))
			values = append(values, ver)

			for _, unitKey := range unitList {
				unitVal := units[unitKey]

				dur, err := unitVal.GetDuration(ver, opts.Ticks, opts.Median)
				if err != nil {
					values = append(values, err.Error())
				} else {
					values = append(values, formatDuration(dur))
				}
			}

			rows = append(rows, values)
		}

		return columns, rows
	}

	columns = append(columns, "Name")
	columns = append(columns, versions...)

	for _, unitKey := range unitList {
		unitVal := units[unitKey]

		values := make([]string, 0, 1+len(versions))
		values = append(values, unitVal.FullName())

		for _, ver := range versions {
			dur, err := unitVal.GetDuration(ver, opts.Ticks, opts.Median)
			if err != nil {
				values = append(values, err.Error())
			} else {
				values = append(values, formatDuration(dur))
			}
		}

		rows = append(rows, values)
	}

	return columns, rows
}

// exportRows writes CSV/JSON exports if requested in options.
func exportCSV(columns []string, rows [][]string, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	defer outFile.Close()

	csvWriter := csv.NewWriter(outFile)

	err = csvWriter.Write(columns)
	if err != nil {
		return fmt.Errorf("write csv header: %w", err)
	}

	for _, row := range rows {
		err = csvWriter.Write(row)
		if err != nil {
			return fmt.Errorf("write csv row: %w", err)
		}
	}

	csvWriter.Flush()

	err = csvWriter.Error()
	if err != nil {
		return fmt.Errorf("flush csv: %w", err)
	}

	return nil
}

func exportJSON(columns []string, rows [][]string, outPath string) error {
	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}
	defer outFile.Close()

	objs := make([]map[string]string, 0, len(rows))

	for _, row := range rows {
		obj := map[string]string{}

		for i, header := range columns {
			val := ""
			if i < len(row) {
				val = row[i]
			}

			obj[header] = val
		}

		objs = append(objs, obj)
	}

	enc := json.NewEncoder(outFile)
	enc.SetIndent("", "  ")

	err = enc.Encode(objs)
	if err != nil {
		return fmt.Errorf("encode json: %w", err)
	}

	return nil
}

func exportRows(columns []string, rows [][]string, opts Options) error {
	if opts.OutputFormat == "" {
		return nil
	}

	outPath := opts.OutputFile
	if outPath == "" {
		outPath = path.Join(opts.Directory, "report."+opts.OutputFormat)
	}

	switch opts.OutputFormat {
	case "csv":
		return exportCSV(columns, rows, outPath)
	case "json":
		return exportJSON(columns, rows, outPath)
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedFormat, opts.OutputFormat)
	}
}

// renderTable configures the table writer, writes header and rows, and renders output.
func renderTable(w io.Writer, columns []string, rows [][]string) error {
	tbl := tablewriter.NewWriter(w)
	tbl.Options(
		tablewriter.WithHeaderAutoFormat(tw.Off),
		tablewriter.WithRowAutoFormat(tw.Off),
		tablewriter.WithRowAutoWrap(tw.WrapNone),
		tablewriter.WithRendition(tw.Rendition{
			Borders: tw.Border{Left: tw.On, Top: tw.Off, Right: tw.On, Bottom: tw.Off, Overwrite: false},
			Symbols: tw.NewSymbols(tw.StyleMarkdown),
			Settings: tw.Settings{
				Separators: tw.Separators{
					ShowHeader:     tw.Off,
					ShowFooter:     tw.Off,
					BetweenRows:    tw.Off,
					BetweenColumns: tw.On,
				},
				Lines: tw.Lines{
					ShowTop:        tw.Off,
					ShowBottom:     tw.Off,
					ShowHeaderLine: tw.On,
					ShowFooterLine: tw.Off,
				},
				CompactMode: tw.Off,
			},
			Streaming: false,
		}),
	)

	hdr := make([]any, len(columns))
	for i, c := range columns {
		hdr[i] = c
	}

	tbl.Header(hdr...)

	for _, values := range rows {
		err := tbl.Append(values)
		if err != nil {
			return fmt.Errorf("append table row: %w", err)
		}
	}

	err := tbl.Render()
	if err != nil {
		return fmt.Errorf("render table: %w", err)
	}

	return nil
}

// CompareVersions implements the same comparison used in Run to sort version strings.
// Returns true if a < b.
func CompareVersions(a, b string) bool {
	normalizedA := strings.ReplaceAll(a, "x", "0")
	normalizedB := strings.ReplaceAll(b, "x", "0")
	ver1, err1 := version.NewVersion(normalizedA)

	ver2, err2 := version.NewVersion(normalizedB)
	if err1 != nil || err2 != nil {
		return normalizedA < normalizedB
	}

	return ver1.LessThan(ver2)
}

// Run parses junit xml files from the provided directory according to options
// and renders a table to the provided writer.
func Run(writer io.Writer, opts Options) error {
	filenames, err := discoverJUnitFiles(opts.Directory)
	if err != nil {
		return err
	}

	units, versions, err := ingestFilesToUnits(filenames, opts)
	if err != nil {
		return err
	}

	// sort versions semantically, treating 'x' as zero
	sort.Slice(versions, func(i, j int) bool {
		normalizedI := strings.ReplaceAll(versions[i], "x", "0")
		normalizedJ := strings.ReplaceAll(versions[j], "x", "0")

		verI, errI := version.NewVersion(normalizedI)

		verJ, errJ := version.NewVersion(normalizedJ)
		if errI != nil || errJ != nil {
			return normalizedI < normalizedJ
		}

		return verI.LessThan(verJ)
	})

	columns, rows := buildTableData(units, versions, opts)

	// render and export
	err = renderTable(writer, columns, rows)
	if err != nil {
		return err
	}

	err = exportRows(columns, rows, opts)
	if err != nil {
		return err
	}

	return nil
}
