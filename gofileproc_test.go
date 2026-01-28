package gofileproc

import (
	"os"
	"testing"
)

func TestCSVProcessor(t *testing.T) {
	// Create test file
	input := "id,name,value\n1,foo,100\n2,bar,200\n3,baz,300\n"
	os.WriteFile("test_input.csv", []byte(input), 0644)
	defer os.Remove("test_input.csv")
	defer os.Remove("test_output.csv")

	proc := NewCSVProcessor(DefaultConfig())
	err := proc.Process("test_input.csv", "test_output.csv", func(line int, fields [][]byte) [][]byte {
		if len(fields) > 1 {
			fields[1] = append(fields[1], []byte("_modified")...)
		}
		return fields
	})

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if proc.Stats.LinesProcessed.Load() != 4 {
		t.Errorf("Expected 4 lines, got %d", proc.Stats.LinesProcessed.Load())
	}

	output, _ := os.ReadFile("test_output.csv")
	if len(output) == 0 {
		t.Error("Output file is empty")
	}
}

func TestJSONProcessor(t *testing.T) {
	input := `[{"name":"foo"},{"name":"bar"}]`
	os.WriteFile("test_input.json", []byte(input), 0644)
	defer os.Remove("test_input.json")
	defer os.Remove("test_output.json")

	proc := NewJSONProcessor(DefaultConfig())
	err := proc.Process("test_input.json", "test_output.json", func(obj map[string]any) map[string]any {
		obj["processed"] = true
		return obj
	})

	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}

	if proc.Stats.LinesProcessed.Load() != 2 {
		t.Errorf("Expected 2 records, got %d", proc.Stats.LinesProcessed.Load())
	}
}

func TestEmptyFile(t *testing.T) {
	os.WriteFile("empty.csv", []byte(""), 0644)
	defer os.Remove("empty.csv")
	defer os.Remove("empty_out.csv")

	proc := NewCSVProcessor(DefaultConfig())
	err := proc.Process("empty.csv", "empty_out.csv", func(line int, fields [][]byte) [][]byte {
		return fields
	})

	if err != nil {
		t.Fatalf("Process failed on empty file: %v", err)
	}
}
