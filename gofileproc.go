package gofileproc

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
)

type Config struct {
	Workers   int
	ChunkSize int64
}

func DefaultConfig() Config {
	workers := runtime.NumCPU()
	if workers > 32 {
		workers = 32
	}
	return Config{
		Workers:   workers,
		ChunkSize: 2 * 1024 * 1024,
	}
}

type Stats struct {
	LinesProcessed atomic.Uint64
	BytesRead      atomic.Uint64
	BytesWritten   atomic.Uint64
}

// CSVProcessor processes CSV files in parallel using mmap.
type CSVProcessor struct {
	config Config
	Stats  Stats
}

func NewCSVProcessor(config Config) *CSVProcessor {
	if config.Workers == 0 {
		config.Workers = runtime.NumCPU()
	}
	return &CSVProcessor{config: config}
}

func (p *CSVProcessor) Process(input, output string, transform func(int, [][]byte) [][]byte) error {
	data, err := mmapFile(input)
	if err != nil {
		return err
	}
	if len(data) == 0 {
		munmap(data)
		return nil
	}

	chunks := splitAtNewlines(data, p.config.Workers)
	results := make([]*chunkResult, len(chunks))
	var wg sync.WaitGroup

	for i, c := range chunks {
		wg.Add(1)
		chunkData := make([]byte, c.end-c.start)
		copy(chunkData, data[c.start:c.end])

		go func(id int, buf []byte) {
			defer wg.Done()
			results[id] = p.processChunk(buf, id, transform)
		}(i, chunkData)
	}

	wg.Wait()
	munmap(data)

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	for _, r := range results {
		if r != nil {
			out.Write(r.data)
			p.Stats.BytesWritten.Add(uint64(len(r.data)))
		}
	}
	return nil
}

type chunkResult struct {
	id    int
	data  []byte
	lines int64
}

func (p *CSVProcessor) processChunk(chunk []byte, id int, transform func(int, [][]byte) [][]byte) *chunkResult {
	out := make([]byte, 0, len(chunk)+len(chunk)/4)
	fields := make([][]byte, 0, 16)
	lineNum := 0
	pos := 0

	for pos < len(chunk) {
		end := pos
		for end < len(chunk) && chunk[end] != '\n' {
			end++
		}

		line := chunk[pos:end]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) > 0 {
			fields = parseCSVLine(line, fields[:0])
			transformed := transform(lineNum, fields)
			out = writeCSVLine(out, transformed)
			lineNum++
		}
		pos = end + 1
	}

	p.Stats.LinesProcessed.Add(uint64(lineNum))
	p.Stats.BytesRead.Add(uint64(len(chunk)))

	return &chunkResult{id: id, data: out, lines: int64(lineNum)}
}

func parseCSVLine(line []byte, fields [][]byte) [][]byte {
	start := 0
	inQuotes := false

	for i := 0; i < len(line); i++ {
		if line[i] == '"' {
			inQuotes = !inQuotes
		} else if line[i] == ',' && !inQuotes {
			fields = append(fields, line[start:i])
			start = i + 1
		}
	}
	return append(fields, line[start:])
}

func writeCSVLine(out []byte, fields [][]byte) []byte {
	for i, f := range fields {
		needQuote := false
		for _, c := range f {
			if c == ',' || c == '"' || c == '\n' {
				needQuote = true
				break
			}
		}

		if needQuote {
			out = append(out, '"')
			for _, c := range f {
				if c == '"' {
					out = append(out, '"', '"')
				} else {
					out = append(out, c)
				}
			}
			out = append(out, '"')
		} else {
			out = append(out, f...)
		}

		if i < len(fields)-1 {
			out = append(out, ',')
		}
	}
	return append(out, '\n')
}

// JSONProcessor processes JSON array files using streaming.
type JSONProcessor struct {
	config Config
	Stats  Stats
}

func NewJSONProcessor(config Config) *JSONProcessor {
	if config.Workers == 0 {
		config.Workers = runtime.NumCPU()
	}
	return &JSONProcessor{config: config}
}

// Process handles JSON array files with streaming (constant memory usage).
func (p *JSONProcessor) Process(input, output string, transform func(map[string]any) map[string]any) error {
	in, err := os.Open(input)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(output)
	if err != nil {
		return err
	}
	defer out.Close()

	dec := json.NewDecoder(in)

	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '[' {
		return fmt.Errorf("expected JSON array, got %v", tok)
	}

	out.WriteString("[\n")
	first := true

	for dec.More() {
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			continue
		}

		result := transform(obj)
		data, err := json.Marshal(result)
		if err != nil {
			continue
		}

		if !first {
			out.WriteString(",\n")
		}
		first = false
		out.Write(data)
		p.Stats.LinesProcessed.Add(1)
	}

	out.WriteString("\n]")
	return nil
}

// Helpers

type span struct {
	start, end int64
}

func splitAtNewlines(data []byte, n int) []span {
	if len(data) == 0 {
		return nil
	}

	size := int64(len(data)) / int64(n)
	if size < 1024 {
		return []span{{0, int64(len(data))}}
	}

	spans := make([]span, 0, n)
	start := int64(0)

	for i := 0; i < n-1; i++ {
		end := start + size
		for end < int64(len(data)) && data[end] != '\n' {
			end++
		}
		if end < int64(len(data)) {
			end++
		}
		spans = append(spans, span{start, end})
		start = end
	}

	if start < int64(len(data)) {
		spans = append(spans, span{start, int64(len(data))})
	}
	return spans
}

func mmapFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	size := stat.Size()
	if size == 0 {
		return nil, nil
	}

	data, err := syscall.Mmap(int(f.Fd()), 0, int(size), syscall.PROT_READ, syscall.MAP_PRIVATE)
	if err != nil {
		return nil, fmt.Errorf("mmap: %w", err)
	}
	return data, nil
}

func munmap(data []byte) {
	if data != nil {
		syscall.Munmap(data)
	}
}
