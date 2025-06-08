package benchmark

import (
	"bytes"
	"encoding/json"
	"testing"
)

// BenchmarkStringVsBytes tests the performance difference between
// string and byte slice operations in our optimized functions

type TestData struct {
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

// BenchmarkJSONUnmarshalString benchmarks JSON unmarshalling with string conversion
func BenchmarkJSONUnmarshalString(b *testing.B) {
	data := `{"name": "test-client", "weight": 3}`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req TestData
		// Old approach: convert string to []byte
		json.Unmarshal([]byte(data), &req)
	}
}

// BenchmarkJSONUnmarshalBytes benchmarks JSON unmarshalling with direct byte slice
func BenchmarkJSONUnmarshalBytes(b *testing.B) {
	data := []byte(`{"name": "test-client", "weight": 3}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var req TestData
		// New approach: use []byte directly
		json.Unmarshal(data, &req)
	}
}

// BenchmarkBodySanitizeString benchmarks body sanitization with string operations
func BenchmarkBodySanitizeString(b *testing.B) {
	// Create a large string that needs truncation
	longString := string(bytes.Repeat([]byte("test data "), 200)) // 2000 chars

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Old approach: work with strings throughout
		maxLen := 1000
		if len(longString) > maxLen {
			_ = longString[:maxLen] + "..."
		}
	}
}

// BenchmarkBodySanitizeBytes benchmarks body sanitization with byte slice operations
func BenchmarkBodySanitizeBytes(b *testing.B) {
	// Create a large byte slice that needs truncation
	longBytes := bytes.Repeat([]byte("test data "), 200) // 2000 chars

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// New approach: work with []byte, convert to string only at the end
		maxLen := 1000
		if len(longBytes) > maxLen {
			_ = string(longBytes[:maxLen]) + "..."
		}
	}
}

// BenchmarkBufferToString benchmarks buffer to string conversion
func BenchmarkBufferToString(b *testing.B) {
	data := []byte("This is test response data that could be large")
	buffer := bytes.NewBuffer(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Old approach: convert buffer to string then back to processing
		_ = buffer.String()
	}
}

// BenchmarkBufferToBytes benchmarks buffer to bytes conversion
func BenchmarkBufferToBytes(b *testing.B) {
	data := []byte("This is test response data that could be large")
	buffer := bytes.NewBuffer(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// New approach: work with bytes directly
		_ = buffer.Bytes()
	}
}

// BenchmarkStringConcatenation benchmarks string concatenation operations
func BenchmarkStringConcatenation(b *testing.B) {
	parts := []string{"Method: GET", "Path: /test", "Status: 200"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Old approach: multiple string concatenations
		result := parts[0] + " | " + parts[1] + " | " + parts[2]
		_ = result
	}
}

// BenchmarkBytesBuffer benchmarks using bytes.Buffer for concatenation
func BenchmarkBytesBuffer(b *testing.B) {
	parts := [][]byte{[]byte("Method: GET"), []byte("Path: /test"), []byte("Status: 200")}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// New approach: use bytes.Buffer for efficient concatenation
		var buf bytes.Buffer
		buf.Write(parts[0])
		buf.WriteString(" | ")
		buf.Write(parts[1])
		buf.WriteString(" | ")
		buf.Write(parts[2])
		_ = buf.String()
	}
}
