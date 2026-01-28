package gofileproc

import (
	"bytes"
	"encoding/csv"
	"io"
	"os"
	"testing"
)

func BenchmarkCSV(b *testing.B) {
	sizes := []struct {
		name    string
		records int
	}{
		{"100_records", 100},
		{"1000_records", 1000},
		{"10000_records", 10000},
		{"100000_records", 100000},
	}

	for _, size := range sizes {
		data := genCSVData(size.records)

		b.Run("encoding_csv/"+size.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				r := csv.NewReader(bytes.NewReader(data))
				for {
					rec, err := r.Read()
					if err == io.EOF {
						break
					}
					if len(rec) > 1 {
						rec[1] = rec[1] + "_x"
					}
				}
			}
		})

		b.Run("gofileproc/"+size.name, func(b *testing.B) {
			b.ReportAllocs()
			tmpIn := "bench_in.csv"
			tmpOut := "bench_out.csv"
			os.WriteFile(tmpIn, data, 0644)
			defer os.Remove(tmpIn)
			defer os.Remove(tmpOut)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				proc := NewCSVProcessor(DefaultConfig())
				proc.Process(tmpIn, tmpOut, func(_ int, fields [][]byte) [][]byte {
					if len(fields) > 1 {
						fields[1] = append(fields[1], "_x"...)
					}
					return fields
				})
			}
		})
	}
}

func genCSVData(records int) []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	w.Write([]string{"a", "b", "c", "d", "e"})
	for i := 0; i < records; i++ {
		w.Write([]string{"1", "foo", "bar", "100", "true"})
	}
	w.Flush()
	return buf.Bytes()
}
