# junit-reporter

## Install

```bash
go install github.com/bavix/junit-reporter@latest
```

#### Alternative installation

```bash
cd /tmp
git clone https://github.com/bavix/junit-reporter.git
cd junit-reporter
go install -ldflags "-s -w"
```

## Usage

```bash
junit-reporter
```

## CLI flags

Supported flags:

- `-ticks` : show per-tick times (averages) instead of total durations  
- `-group` : group input files by semantic version pattern extracted from filenames  
- `-major` : when used with `-group`, collapse to major.x (e.g. 7.x)  
- `-median` : use median instead of average for tick mode  
- `-rotate` : swap rows and columns (versions as rows)  
- `-path` : specify input directory (default `./build`)  
- `-output-format` : optional export format, `csv` or `json` (writes additional file)  
- `-output-file` : optional path to write exported CSV/JSON (defaults to `<path>/report.<format>`)  

Examples:

```bash
# default table printed to stdout
junit-reporter -path ./build

# export CSV alongside printing
junit-reporter -path ./build -output-format csv -output-file ./build/report.csv
```

## Examples

Basic runs:

```bash
# print totals per version (default)
junit-reporter -path ./build

# print per-tick averages (faster numbers)
junit-reporter -path ./build -ticks

# use median instead of average in ticks mode
junit-reporter -path ./build -ticks -median

# group files by semantic version extracted from filenames
junit-reporter -path ./build -group

# group and collapse to major versions (e.g. 7.x)
junit-reporter -path ./build -group -major

# rotate output: versions as rows, tests as columns
junit-reporter -path ./build -rotate
```

Exporting:

```bash
# export CSV to a file (and still print table)
junit-reporter -path ./build -output-format csv -output-file ./build/report.csv

# export JSON (useful for automated processing)
junit-reporter -path ./build -output-format json -output-file ./build/report.json
```

Integration / regression workflow:

```bash
# save baseline outputs for later comparison
junit-reporter -path ./build > build/runs/run-default.txt
junit-reporter -path ./build -ticks > build/runs/run-ticks.txt
junit-reporter -path ./build -group > build/runs/run-group.txt

# after refactor/change, regenerate and diff
junit-reporter -path ./build > build/runs/post/run-default.txt
diff -u build/runs/run-default.txt build/runs/post/run-default.txt
```

Running tests:

```bash
go test ./...         # run unit & integration tests
go test ./... -cover  # show coverage
```
