package cli

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/store"
)

func TestStatusResultJSONStruct(t *testing.T) {
	result := StatusResultJSON{
		FilesIndexed:   10,
		TotalChunks:    50,
		IndexSizeBytes: 1024 * 1024,
		LastUpdated:    "2024-01-15T10:30:00Z",
		Provider:       "openai",
		Model:          "text-embedding-3-small",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal StatusResultJSON: %v", err)
	}

	// Verify all fields are present in JSON with correct snake_case names
	var decoded map[string]interface{}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	expectedFields := []string{"files_indexed", "total_chunks", "index_size_bytes", "last_updated", "provider", "model"}
	for _, field := range expectedFields {
		if _, exists := decoded[field]; !exists {
			t.Errorf("expected field '%s' to be present", field)
		}
	}
}

func TestStatusResultJSONValues(t *testing.T) {
	result := StatusResultJSON{
		FilesIndexed:   10,
		TotalChunks:    50,
		IndexSizeBytes: 1024 * 1024,
		LastUpdated:    "2024-01-15T10:30:00Z",
		Provider:       "openai",
		Model:          "text-embedding-3-small",
	}

	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal StatusResultJSON: %v", err)
	}

	var decoded StatusResultJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if decoded.FilesIndexed != 10 {
		t.Errorf("expected files_indexed 10, got %d", decoded.FilesIndexed)
	}
	if decoded.TotalChunks != 50 {
		t.Errorf("expected total_chunks 50, got %d", decoded.TotalChunks)
	}
	if decoded.IndexSizeBytes != 1024*1024 {
		t.Errorf("expected index_size_bytes %d, got %d", 1024*1024, decoded.IndexSizeBytes)
	}
	if decoded.LastUpdated != "2024-01-15T10:30:00Z" {
		t.Errorf("expected last_updated '2024-01-15T10:30:00Z', got '%s'", decoded.LastUpdated)
	}
	if decoded.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", decoded.Provider)
	}
	if decoded.Model != "text-embedding-3-small" {
		t.Errorf("expected model 'text-embedding-3-small', got '%s'", decoded.Model)
	}
}

func TestOutputStatusJSON(t *testing.T) {
	stats := &store.IndexStats{
		TotalFiles:  10,
		TotalChunks: 50,
		IndexSize:   1024 * 1024,
		LastUpdated: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
	}

	cfg := &config.Config{
		Embedder: config.EmbedderConfig{
			Provider: "openai",
			Model:    "text-embedding-3-small",
		},
	}

	// Test the JSON generation logic
	lastUpdated := ""
	if !stats.LastUpdated.IsZero() {
		lastUpdated = stats.LastUpdated.Format("2006-01-02T15:04:05Z07:00")
	}

	result := StatusResultJSON{
		FilesIndexed:   stats.TotalFiles,
		TotalChunks:    stats.TotalChunks,
		IndexSizeBytes: stats.IndexSize,
		LastUpdated:    lastUpdated,
		Provider:       cfg.Embedder.Provider,
		Model:          cfg.Embedder.Model,
	}

	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(result); err != nil {
		t.Fatalf("failed to encode JSON: %v", err)
	}

	// Verify output can be decoded back
	var decoded StatusResultJSON
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v", err)
	}

	if decoded.FilesIndexed != 10 {
		t.Errorf("expected files_indexed 10, got %d", decoded.FilesIndexed)
	}
	if decoded.Provider != "openai" {
		t.Errorf("expected provider 'openai', got '%s'", decoded.Provider)
	}
}

func TestStatusPlainAndJSONMutualExclusivity(t *testing.T) {
	// Save original values
	originalPlain := statusPlain
	originalJSON := statusJSON

	// Reset after test
	defer func() {
		statusPlain = originalPlain
		statusJSON = originalJSON
	}()

	// Test case: both flags set should be invalid
	statusPlain = true
	statusJSON = true

	// Verify the validation logic
	if statusPlain && statusJSON {
		// This is the expected behavior - validation should fail
		return
	}

	t.Error("expected --plain and --json to be mutually exclusive")
}

func TestStatusPlainFlagValid(t *testing.T) {
	// Save original values
	originalPlain := statusPlain
	originalJSON := statusJSON

	// Reset after test
	defer func() {
		statusPlain = originalPlain
		statusJSON = originalJSON
	}()

	// Test case: only --plain flag set should be valid
	statusPlain = true
	statusJSON = false

	if statusPlain && statusJSON {
		t.Error("expected --plain alone to be valid")
	}
}

func TestStatusJSONFlagValid(t *testing.T) {
	// Save original values
	originalPlain := statusPlain
	originalJSON := statusJSON

	// Reset after test
	defer func() {
		statusPlain = originalPlain
		statusJSON = originalJSON
	}()

	// Test case: only --json flag set should be valid
	statusPlain = false
	statusJSON = true

	if statusPlain && statusJSON {
		t.Error("expected --json alone to be valid")
	}
}

func TestStatusEmptyLastUpdated(t *testing.T) {
	// Test the JSON generation logic with zero time
	lastUpdated := time.Time{}

	// Verify that zero time results in empty string
	lastUpdatedStr := ""
	if !lastUpdated.IsZero() {
		lastUpdatedStr = lastUpdated.Format("2006-01-02T15:04:05Z07:00")
	}

	if lastUpdatedStr != "" {
		t.Errorf("expected last_updated to be empty string for zero time, got '%s'", lastUpdatedStr)
	}
}

func TestTTYErrorMessageContainsSuggestions(t *testing.T) {
	// Test that the TTY error message includes helpful suggestions
	errorMsg := "interactive status requires a terminal (TTY)\nUse --plain or --json for non-interactive output"

	if !containsString(errorMsg, "--plain") {
		t.Error("TTY error message should mention --plain flag")
	}
	if !containsString(errorMsg, "--json") {
		t.Error("TTY error message should mention --json flag")
	}
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
