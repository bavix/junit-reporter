package main

import (
	"errors"
	"flag"
	"github.com/hashicorp/go-version"
	"github.com/joshdk/go-junit"
	"github.com/montanaflynn/stats"
	"github.com/olekukonko/tablewriter"
	"path"
	"regexp"
	"sort"
	"time"

	"log"
	"os"
	"strings"
)

type Unit struct {
	Class  string
	Method string
	t      []UTest
}

type UTest struct {
	Ver   string
	JUnit junit.Test
}

func (u *Unit) FullName() string {
	return strings.TrimSuffix(u.Class, "Test") + ":" + strings.TrimPrefix(u.Method, "test")
}

func (u *Unit) Push(ver string, t junit.Test) {
	u.t = append(u.t, UTest{Ver: ver, JUnit: t})
}

func formatDuration(d time.Duration) string {
	scale := 100 * time.Second
	for scale > d {
		scale = scale / 10
	}

	return d.Round(scale / 100).String()
}

func (u *Unit) getDurationSum(input []time.Duration) time.Duration {
	d := time.Duration(0)
	for _, i := range input {
		d += i
	}

	return d
}

func (u *Unit) getDurationAverage(input []time.Duration) time.Duration {
	return time.Duration(int64(u.getDurationSum(input)) / int64(len(input)))
}

func (u *Unit) getDurationMedian(input []time.Duration) time.Duration {
	var results []float64
	for _, i := range input {
		results = append(results, float64(i))
	}

	median, err := stats.Median(results)
	if err != nil {
		return 0
	}

	return time.Duration(int64(median))
}

func (u *Unit) GetDuration(ver string, ticks *bool, median *bool) (time.Duration, error) {
	var results []time.Duration
	for _, c := range u.t {
		if c.Ver == ver {
			if c.JUnit.Status != "passed" {
				return 0, errors.New("-")
			}

			results = append(results, c.JUnit.Duration)
		}
	}

	if len(results) == 0 {
		return 0, errors.New("-")
	}

	if *ticks {
		if *median {
			return u.getDurationMedian(results), nil
		}

		return u.getDurationAverage(results), nil
	}

	return u.getDurationSum(results), nil
}

func NewUnit(ver string, t junit.Test) Unit {
	namespaces := strings.Split(t.Classname, ".")
	className := namespaces[len(namespaces)-1]
	method := strings.Fields(t.Name)[0]
	uTest := UTest{Ver: ver, JUnit: t}

	return Unit{Class: className, Method: method, t: []UTest{uTest}}
}

func depthSuite(suite junit.Suite) []junit.Test {
	tests := suite.Tests
	for _, one := range suite.Suites {
		tests = append(tests, depthSuite(one)...)
	}
	return tests
}

func main() {
	ticks := flag.Bool("ticks", false, "Time per ticks")
	group := flag.Bool("group", false, "Groups by version")
	major := flag.Bool("major", false, "Can only be used with a group")
	median := flag.Bool("median", false, "Median search")
	rotate := flag.Bool("rotate", false, "Swap versions and names")
	directory := flag.String("path", "./build", "Specify folder path")
	flag.Parse()

	var filenames []string

	files, err := os.ReadDir(*directory)
	if err != nil {
		log.Fatalln(err)
	}

	for _, info := range files {
		if info.Type().IsRegular() && strings.HasSuffix(info.Name(), ".xml") && strings.HasPrefix(info.Name(), "junit-") {
			filenames = append(filenames, path.Join(*directory, info.Name()))
		}
	}

	if len(filenames) == 0 {
		log.Fatalln("Files not found")
	}

	units := map[string]*Unit{}
	verKeys := map[string]bool{}
	var versions []string

	for _, _path := range filenames {
		ingestFile, err := junit.IngestFile(_path)
		if err != nil {
			log.Fatalf("failed to ingest JUnit xml %v", err)
		}

		var ver string

		if *group {
			re, _ := regexp.Compile(`\d+\.((\d+|x)(\.(\d+|x))?)`)
			ver = string(re.Find([]byte(_path)))
			if *major {
				ver = strings.Split(ver, ".")[0] + ".x"
			}
		} else {
			re := regexp.MustCompile(`junit-(.+).xml`)
			ver = re.FindStringSubmatch(_path)[1]
		}

		if _, ok := verKeys[ver]; !ok {
			versions = append(versions, ver)
			verKeys[ver] = true
		}

		for _, suite := range ingestFile {
			for _, test := range depthSuite(suite) {
				unit := NewUnit(ver, test)
				if elem, ok := units[unit.FullName()]; ok {
					elem.Push(ver, test)
					continue
				}

				units[unit.FullName()] = &unit
			}
		}
	}

	var columns []string
	var unitList []string
	for _, unit := range units {
		unitList = append(unitList, unit.FullName())
	}

	sort.Slice(versions, func(i, j int) bool {
		vi := strings.ReplaceAll(versions[i], "x", "0")
		vj := strings.ReplaceAll(versions[j], "x", "0")

		v1, err1 := version.NewVersion(vi)
		v2, err2 := version.NewVersion(vj)
		if err1 != nil || err2 != nil {
			return vi < vj
		}

		return v1.LessThan(v2)
	})

	sort.Slice(unitList, func(i, j int) bool {
		return unitList[i] < unitList[j]
	})

	table := tablewriter.NewWriter(os.Stdout)

	if *rotate {
		columns = append(columns, "Ver")
		columns = append(columns, unitList...)

		for _, ver := range versions {
			var values []string
			values = append(values, ver)
			for _, unitKey := range unitList {
				unit := units[unitKey]
				if dur, err := unit.GetDuration(ver, ticks, median); err != nil {
					values = append(values, err.Error())
				} else {
					values = append(values, formatDuration(dur))
				}
			}

			table.Append(values)
		}
	} else {
		columns = append(columns, "Name")
		columns = append(columns, versions...)

		for _, unitKey := range unitList {
			unit := units[unitKey]
			var values []string
			values = append(values, unit.FullName())
			for _, ver := range versions {
				if dur, err := unit.GetDuration(ver, ticks, median); err != nil {
					values = append(values, err.Error())
				} else {
					values = append(values, formatDuration(dur))
				}
			}

			table.Append(values)
		}
	}

	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetAutoFormatHeaders(false)
	table.SetAutoWrapText(false)
	table.SetCenterSeparator("|")
	table.SetHeader(columns)

	table.Render()
}
