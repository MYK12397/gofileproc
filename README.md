# gofileproc
 
Fast CSV file processor using mmap and parallel processing.

## Installation

```bash
go get github.com/MYK12397/gofileproc
```

## Usage
 
```go
// CSV
proc := gofileproc.NewCSVProcessor(gofileproc.DefaultConfig())
proc.Process("in.csv", "out.csv", func(line int, fields [][]byte) [][]byte {
    fields[1] = append(fields[1], "_modified"...)
    return fields
})
```

## Benchmark

```
goos: linux
goarch: amd64
pkg: github.com/MYK12397/gofileproc
cpu: Intel(R) Core(TM) i5-10210U CPU @ 1.60GHz
BenchmarkCSV/encoding_csv/100_records-8                    45068             26165 ns/op           14848 B/op        316 allocs/op
BenchmarkCSV/gofileproc/100_records-8                      16179             70562 ns/op            5816 B/op         17 allocs/op
BenchmarkCSV/encoding_csv/1000_records-8                    4790            256559 ns/op          106048 B/op       3016 allocs/op
BenchmarkCSV/gofileproc/1000_records-8                      7012            270345 ns/op           50753 B/op         59 allocs/op
BenchmarkCSV/encoding_csv/10000_records-8                    376           2724691 ns/op         1018054 B/op      30016 allocs/op
BenchmarkCSV/gofileproc/10000_records-8                      530           3268496 ns/op          463812 B/op         59 allocs/op
BenchmarkCSV/encoding_csv/100000_records-8                    43          27105206 ns/op        10138130 B/op     300016 allocs/op
BenchmarkCSV/gofileproc/100000_records-8                     100          14856413 ns/op         4330446 B/op         59 allocs/op
```
 
At 100k records: **1.8x faster**, **5000x fewer allocations**.
 
Small files have mmap overhead; gofileproc wins at scale.
 
## Run Benchmarks
 
```bash
go test -bench=. -benchmem
```