package reporter

import (
	"testing"
	"time"

	"github.com/joshdk/go-junit"
)

func makeTest(name, class string, status junit.Status, dur time.Duration) junit.Test {
	var testCase junit.Test

	testCase.Name = name
	testCase.Classname = class
	testCase.Status = status
	testCase.Duration = dur

	return testCase
}

func TestNewUnitAndFullName(t *testing.T) {
	t.Parallel()

	unitVal := newUnit("1.2.3", makeTest("testSomething arg", "a.b.CThingTest", junit.StatusPassed, 100*time.Millisecond))
	if unitVal.Class != "CThingTest" {
		t.Fatalf("unexpected class: %s", unitVal.Class)
	}

	if unitVal.Method != "testSomething" {
		t.Fatalf("unexpected method: %s", unitVal.Method)
	}

	if got := unitVal.FullName(); got != "CThing:Something" {
		t.Fatalf("FullName mismatch: %s", got)
	}
}

func TestUnitPushAndGetDurationSumAverageMedian(t *testing.T) {
	t.Parallel()

	unitVal := newUnit("v1", makeTest("testA", "pkg.TestA", junit.StatusPassed, 100*time.Millisecond))
	unitVal.Push("v1", makeTest("testA", "pkg.TestA", junit.StatusPassed, 300*time.Millisecond))
	unitVal.Push("v1", makeTest("testA", "pkg.TestA", junit.StatusPassed, 200*time.Millisecond))

	// sum
	duration, err := unitVal.GetDuration("v1", false, false)
	if err != nil {
		t.Fatalf("GetDuration sum error: %v", err)
	}

	if duration != 600*time.Millisecond {
		t.Fatalf("expected sum 600ms, got %v", duration)
	}

	// average (ticks=false would return sum; ticks=true gives average)
	duration, err = unitVal.GetDuration("v1", true, false)
	if err != nil {
		t.Fatalf("GetDuration avg error: %v", err)
	}

	if duration != 200*time.Millisecond {
		t.Fatalf("expected avg 200ms, got %v", duration)
	}

	// median
	duration, err = unitVal.GetDuration("v1", true, true)
	if err != nil {
		t.Fatalf("GetDuration median error: %v", err)
	}

	if duration != 200*time.Millisecond {
		t.Fatalf("expected median 200ms, got %v", duration)
	}
}

func TestGetDurationErrors(t *testing.T) {
	t.Parallel()

	unitVal := newUnit("v1", makeTest("testA", "pkg.TestA", junit.StatusFailed, 100*time.Millisecond))

	_, err := unitVal.GetDuration("v1", false, false)
	if err == nil {
		t.Fatalf("expected error for non-passed test")
	}

	_, err = unitVal.GetDuration("v2", false, false)
	if err == nil {
		t.Fatalf("expected error for missing version")
	}
}

func TestDepthSuite(t *testing.T) {
	t.Parallel()

	suiteVal := junit.Suite{
		Tests: []junit.Test{makeTest("t1", "pkg.S1", junit.StatusPassed, 10*time.Millisecond)},
		Suites: []junit.Suite{
			{
				Tests: []junit.Test{makeTest("t2", "pkg.S2", junit.StatusPassed, 20*time.Millisecond)},
				Suites: []junit.Suite{
					{
						Tests:      []junit.Test{makeTest("t3", "pkg.S3", junit.StatusPassed, 30*time.Millisecond)},
						Suites:     nil,
						Name:       "",
						Package:    "",
						Properties: nil,
						SystemOut:  "",
						SystemErr:  "",
						Totals:     junit.Totals{Tests: 0, Passed: 0, Skipped: 0, Failed: 0, Error: 0, Duration: 0},
					},
				},
				Name:       "",
				Package:    "",
				Properties: nil,
				SystemOut:  "",
				SystemErr:  "",
				Totals:     junit.Totals{Tests: 0, Passed: 0, Skipped: 0, Failed: 0, Error: 0, Duration: 0},
			},
		},
		Name:       "",
		Package:    "",
		Properties: nil,
		SystemOut:  "",
		SystemErr:  "",
		Totals:     junit.Totals{Tests: 0, Passed: 0, Skipped: 0, Failed: 0, Error: 0, Duration: 0},
	}

	out := depthSuite(suiteVal)
	if len(out) != 3 {
		t.Fatalf("expected 3 tests flattened, got %d", len(out))
	}
}
