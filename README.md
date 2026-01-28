# gofileproc
 
Fast CSV and JSON file processor using mmap and parallel processing.
 
## Benchmark
 
```
goos: darwin
goarch: arm64
cpu: Apple M4 Pro
 
BenchmarkCSV/encoding_csv/100_records-12       92715      12062 ns/op    14848 B/op    316 allocs/op
BenchmarkCSV/gofileproc/100_records-12          2144     663030 ns/op     5816 B/op     17 allocs/op
 
BenchmarkCSV/encoding_csv/1000_records-12      10000     106799 ns/op   106049 B/op   3016 allocs/op
BenchmarkCSV/gofileproc/1000_records-12          326    3423879 ns/op    53230 B/op     83 allocs/op
 
BenchmarkCSV/encoding_csv/10000_records-12      1081    1071220 ns/op  1018067 B/op  30016 allocs/op
BenchmarkCSV/gofileproc/10000_records-12         444    5559482 ns/op   449797 B/op     83 allocs/op
 
BenchmarkCSV/encoding_csv/100000_records-12      100   10866230 ns/op 10138185 B/op 300016 allocs/op
BenchmarkCSV/gofileproc/100000_records-12        248    5797796 ns/op  4431146 B/op     83 allocs/op
```
 
At 100k records: **1.9x faster**, **3600x fewer allocations**.
 
Small files have mmap overhead; gofileproc wins at scale.
 
## Usage
 
```go
// CSV
proc := gofileproc.NewCSVProcessor(gofileproc.DefaultConfig())
proc.Process("in.csv", "out.csv", func(line int, fields [][]byte) [][]byte {
    fields[1] = append(fields[1], "_modified"...)
    return fields
})
 
// JSON
jp := gofileproc.NewJSONProcessor(gofileproc.DefaultConfig())
jp.Process("in.json", "out.json", func(obj map[string]any) map[string]any {
    obj["done"] = true
    return obj
})
```
 
## Run Benchmarks
 
```bash
go test -bench=. -benchmem
```