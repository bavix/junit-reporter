package main

import (
	"flag"
	"github.com/joshdk/go-junit"
	"github.com/olekukonko/tablewriter"
	"regexp"
	"sort"
	"time"

	"log"
	"os"
	"path/filepath"
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

func (u *Unit) GetDuration(version string) time.Duration {
	d := time.Duration(0)
	for _, c := range u.t {
		if c.Version == version {
			d = d + c.JUnit.Duration
		}
	}

	return d
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
	rotate := flag.Bool("rotate", false, "Swap versions and names")
	directory := flag.String("path", "./build", "Specify folder path")
	flag.Parse()

	var filenames []string

	err := filepath.Walk(*directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Add all regular files that end with ".xml"
		if info.Mode().IsRegular() && strings.HasSuffix(info.Name(), ".xml") && strings.HasPrefix(info.Name(), "junit-") {
			filenames = append(filenames, path)
		}

		return nil
	})

	if len(filenames) == 0 {
		log.Fatalln("Files not found")
	}

	if err != nil {
		log.Fatalln(err)
	}

	units := map[string]*Unit{}
	var versions []string

	for _, path := range filenames {
		ingestFile, err := junit.IngestFile(path)
		if err != nil {
			log.Fatalf("failed to ingest JUnit xml %v", err)
		}

		re := regexp.MustCompile(`junit-(.+).xml`)
		version := re.FindStringSubmatch(path)[1]
		versions = append(versions, version)

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
				values = append(values, unit.GetDuration(version).String())
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
				values = append(values, unit.GetDuration(version).String())
			}

			table.Append(values)
		}
	}

	table.SetBorders(tablewriter.Border{Left: true, Top: false, Right: true, Bottom: false})
	table.SetCenterSeparator("|")
	table.SetHeader(columns)

	table.Render()
}
