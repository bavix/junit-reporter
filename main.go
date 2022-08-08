package main

import (
	"errors"
	"flag"
	"github.com/joshdk/go-junit"
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
	Version string
	JUnit   junit.Test
}

func (u *Unit) FullName() string {
	return strings.TrimSuffix(u.Class, "Test") + ":" + strings.TrimPrefix(u.Method, "test")
}

func (u *Unit) Push(version string, t junit.Test) {
	u.t = append(u.t, UTest{Version: version, JUnit: t})
}

func (u *Unit) GetDuration(version string, ticks *bool) (time.Duration, error) {
	d := time.Duration(0)
	var i int64 = 0
	for _, c := range u.t {
		if c.Version == version {
			if c.JUnit.Status != "passed" {
				return 0, errors.New("-")
			}

			d = d + c.JUnit.Duration
			i++
		}
	}

	if *ticks && i > 0 {
		return time.Duration(int64(d) / i), nil
	}

	return d, nil
}

func NewUnit(version string, t junit.Test) Unit {
	namespaces := strings.Split(t.Classname, ".")
	className := namespaces[len(namespaces)-1]
	method := strings.Fields(t.Name)[0]
	uTest := UTest{Version: version, JUnit: t}

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

		var version string

		if *group {
			re, _ := regexp.Compile(`\d+\.((\d+|x)(\.(\d+|x))?)`)
			version = string(re.Find([]byte(_path)))
		} else {
			re := regexp.MustCompile(`junit-(.+).xml`)
			version = re.FindStringSubmatch(_path)[1]
		}

		if _, ok := verKeys[version]; !ok {
			versions = append(versions, version)
			verKeys[version] = true
		}

		for _, suite := range ingestFile {
			for _, test := range depthSuite(suite) {
				unit := NewUnit(version, test)
				if elem, ok := units[unit.FullName()]; ok {
					elem.Push(version, test)
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

	sort.Slice(unitList, func(i, j int) bool {
		return unitList[i] < unitList[j]
	})

	table := tablewriter.NewWriter(os.Stdout)

	if *rotate {
		columns = append(columns, "Version")
		columns = append(columns, unitList...)

		for _, version := range versions {
			var values []string
			values = append(values, version)
			for _, unitKey := range unitList {
				unit := units[unitKey]
				if dur, err := unit.GetDuration(version, ticks); err != nil {
					values = append(values, err.Error())
				} else {
					values = append(values, dur.String())
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
			for _, version := range versions {
				if dur, err := unit.GetDuration(version, ticks); err != nil {
					values = append(values, err.Error())
				} else {
					values = append(values, dur.String())
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
