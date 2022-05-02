package main

import (
	"github.com/fatih/color"
	"github.com/joshdk/go-junit"
	"github.com/rodaine/table"
	"regexp"
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
	return u.Class + "::" + u.Method
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
	directory := "./build"
	var filenames []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Add all regular files that end with ".xml"
		if info.Mode().IsRegular() && strings.HasSuffix(info.Name(), ".xml") && strings.HasPrefix(info.Name(), "junit-") {
			filenames = append(filenames, path)
		}

		return nil
	})

	if err != nil {
		log.Fatalln(err)
	}

	units := map[string]*Unit{}
	var versions []interface{}

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

	headerFmt := color.New(color.FgGreen, color.Underline).SprintfFunc()
	columnFmt := color.New(color.FgYellow).SprintfFunc()

	var columns []interface{}
	columns = append(columns, "Name")
	columns = append(columns, versions...)

	tbl := table.New(columns...)
	tbl.WithHeaderFormatter(headerFmt).WithFirstColumnFormatter(columnFmt)

	for _, unit := range units {
		var values []interface{}
		values = append(values, unit.FullName())
		for _, ver := range versions {
			if version, ok := ver.(string); ok {
				values = append(values, unit.GetDuration(version))
			}
		}

		tbl.AddRow(values...)
	}

	tbl.Print()
}
