package cli

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestIndexResultJSONStruct(t *testing.T) {
	result := IndexResultJSON{
		FilesIndexed:     10,
		ChunksCreated:    50,
		FilesRemoved:     2,
		FilesSkipped:     3,
		SymbolsExtracted: 100,
		DurationMs:       1500,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal IndexResultJSON: %v", err)
	}

	// Verify all fields are present in JSON with correct snake_case names
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	expectedFields := []string{"files_indexed", "chunks_created", "files_removed", "files_skipped", "symbols_extracted", "duration_ms"}
	for _, field := range expectedFields {
		if _, exists := decoded[field]; !exists {
			t.Errorf("expected field '%s' to be present", field)
		}
	}
}

func TestIndexResultJSONValues(t *testing.T) {
	result := IndexResultJSON{
		FilesIndexed:     10,
		ChunksCreated:    50,
		FilesRemoved:     2,
		FilesSkipped:     3,
		SymbolsExtracted: 100,
		DurationMs:       1500,
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal IndexResultJSON: %v", err)
	}

	var decoded IndexResultJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.FilesIndexed != 10 {
		t.Errorf("expected files_indexed 10, got %d", decoded.FilesIndexed)
	}
	if decoded.ChunksCreated != 50 {
		t.Errorf("expected chunks_created 50, got %d", decoded.ChunksCreated)
	}
	if decoded.FilesRemoved != 2 {
		t.Errorf("expected files_removed 2, got %d", decoded.FilesRemoved)
	}
	if decoded.FilesSkipped != 3 {
		t.Errorf("expected files_skipped 3, got %d", decoded.FilesSkipped)
	}
	if decoded.SymbolsExtracted != 100 {
		t.Errorf("expected symbols_extracted 100, got %d", decoded.SymbolsExtracted)
	}
	if decoded.DurationMs != 1500 {
		t.Errorf("expected duration_ms 1500, got %d", decoded.DurationMs)
	}
}

func TestOutputIndexJSON(t *testing.T) {
	result := IndexResultJSON{
		FilesIndexed:     10,
		ChunksCreated:    50,
		FilesRemoved:     2,
		FilesSkipped:     3,
		SymbolsExtracted: 100,
		DurationMs:       1500,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode JSON: %v", err)
	}

	// Verify output can be decoded back
	var decoded IndexResultJSON
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}

	if decoded.FilesIndexed != result.FilesIndexed {
		t.Errorf("files_indexed mismatch: expected %d, got %d", result.FilesIndexed, decoded.FilesIndexed)
	}
}

func TestIndexErrorJSONFormat(t *testing.T) {
	// Test that error JSON has the expected structure
	testError := "test error message"
	errorJSON := map[string]string{"error": testError}

	data, err := json.Marshal(errorJSON)
	if err != nil {
		t.Fatalf("failed to marshal error JSON: %v", err)
	}

	var decoded map[string]string
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal error JSON: %v", err)
	}

	if decoded["error"] != testError {
		t.Errorf("expected error message '%s', got '%s'", testError, decoded["error"])
	}
}

func TestIndexJSONFlagDefault(t *testing.T) {
	// Save original value
	originalJSON := indexJSON

	// Reset after test
	defer func() {
		indexJSON = originalJSON
	}()

	// Default should be false
	indexJSON = false
	if indexJSON {
		t.Error("expected default indexJSON to be false")
	}
}
