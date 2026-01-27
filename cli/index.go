package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yoanbernabeu/grepai/config"
	"github.com/yoanbernabeu/grepai/indexer"
	"github.com/yoanbernabeu/grepai/trace"
)

var indexJSON bool

// IndexResultJSON is the JSON output structure for the index command
type IndexResultJSON struct {
	FilesIndexed     int   `json:"files_indexed"`
	ChunksCreated    int   `json:"chunks_created"`
	FilesRemoved     int   `json:"files_removed"`
	FilesSkipped     int   `json:"files_skipped"`
	SymbolsExtracted int   `json:"symbols_extracted"`
	DurationMs       int64 `json:"duration_ms"`
}

var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Index the codebase once and exit",
	Long: `Perform a one-shot index of the codebase and exit.

Unlike 'grepai watch', this command indexes all files once and terminates,
making it suitable for CI/CD pipelines and AI agents that need deterministic,
finite operations.

Examples:
  grepai index              Index codebase and print human-readable summary
  grepai index --json       Index codebase and output JSON result`,
	RunE: runIndex,
}

func init() {
	indexCmd.Flags().BoolVarP(&indexJSON, "json", "j", false, "Output result in JSON format (for AI agents)")
}

func runIndex(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	startTime := time.Now()

	// Find project root
	projectRoot, err := config.FindProjectRoot()
	if err != nil {
		return outputIndexError(err)
	}

	// Load configuration
	cfg, err := config.Load(projectRoot)
	if err != nil {
		return outputIndexError(fmt.Errorf("failed to load configuration: %w", err))
	}

	// Initialize embedder
	emb, err := initializeEmbedder(ctx, cfg)
	if err != nil {
		return outputIndexError(err)
	}
	defer emb.Close()

	// Initialize store
	st, err := initializeStore(ctx, cfg, projectRoot)
	if err != nil {
		return outputIndexError(err)
	}
	defer st.Close()

	// Initialize ignore matcher
	ignoreMatcher, err := indexer.NewIgnoreMatcher(projectRoot, cfg.Ignore, cfg.ExternalGitignore)
	if err != nil {
		return outputIndexError(fmt.Errorf("failed to initialize ignore matcher: %w", err))
	}

	// Initialize scanner
	scanner := indexer.NewScanner(projectRoot, ignoreMatcher)

	// Initialize chunker
	chunker := indexer.NewChunker(cfg.Chunking.Size, cfg.Chunking.Overlap)

	// Initialize indexer
	idx := indexer.NewIndexer(projectRoot, st, emb, chunker, scanner, cfg.Watch.LastIndexTime)

	// Initialize symbol store and extractor
	symbolStore := trace.NewGOBSymbolStore(config.GetSymbolIndexPath(projectRoot))
	if err := symbolStore.Load(ctx); err != nil {
		// Log warning but continue - symbol index is optional
		if !indexJSON {
			fmt.Fprintf(os.Stderr, "Warning: failed to load symbol index: %v\n", err)
		}
	}
	defer symbolStore.Close()

	extractor := trace.NewRegexExtractor()

	// Use default trace languages if not configured
	tracedLanguages := cfg.Trace.EnabledLanguages
	if len(tracedLanguages) == 0 {
		tracedLanguages = []string{".go", ".js", ".ts", ".jsx", ".tsx", ".py", ".php", ".java", ".cs"}
	}

	// Perform indexing
	if !indexJSON {
		fmt.Println("Indexing codebase...")
	}

	var stats *indexer.IndexStats
	stats, err = idx.IndexAllWithBatchProgress(ctx, nil, nil)
	if err != nil {
		return outputIndexError(fmt.Errorf("indexing failed: %w", err))
	}

	// Update lastIndexTime in config if files were indexed
	if stats.FilesIndexed > 0 || stats.ChunksCreated > 0 {
		cfg.Watch.LastIndexTime = time.Now()
		if err := cfg.Save(projectRoot); err != nil {
			if !indexJSON {
				fmt.Fprintf(os.Stderr, "Warning: failed to save config: %v\n", err)
			}
		}
	}

	// Save index
	if err := st.Persist(ctx); err != nil {
		return outputIndexError(fmt.Errorf("failed to persist index: %w", err))
	}

	// Build symbol index
	symbolCount := 0
	files, _, _ := scanner.Scan()
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Path))
		if !isTracedLanguage(ext, tracedLanguages) {
			continue
		}
		symbols, refs, err := extractor.ExtractAll(ctx, file.Path, file.Content)
		if err != nil {
			if !indexJSON {
				fmt.Fprintf(os.Stderr, "Warning: failed to extract symbols from %s: %v\n", file.Path, err)
			}
			continue
		}
		if err := symbolStore.SaveFile(ctx, file.Path, symbols, refs); err != nil {
			if !indexJSON {
				fmt.Fprintf(os.Stderr, "Warning: failed to save symbols for %s: %v\n", file.Path, err)
			}
		}
		symbolCount += len(symbols)
	}
	if err := symbolStore.Persist(ctx); err != nil {
		if !indexJSON {
			fmt.Fprintf(os.Stderr, "Warning: failed to persist symbol index: %v\n", err)
		}
	}

	duration := time.Since(startTime)

	// Output result
	if indexJSON {
		return outputIndexJSON(IndexResultJSON{
			FilesIndexed:     stats.FilesIndexed,
			ChunksCreated:    stats.ChunksCreated,
			FilesRemoved:     stats.FilesRemoved,
			FilesSkipped:     stats.FilesSkipped,
			SymbolsExtracted: symbolCount,
			DurationMs:       duration.Milliseconds(),
		})
	}

	// Human-readable output
	fmt.Printf("Indexing complete:\n")
	fmt.Printf("  Files indexed:     %d\n", stats.FilesIndexed)
	fmt.Printf("  Chunks created:    %d\n", stats.ChunksCreated)
	fmt.Printf("  Files removed:     %d\n", stats.FilesRemoved)
	fmt.Printf("  Files skipped:     %d\n", stats.FilesSkipped)
	fmt.Printf("  Symbols extracted: %d\n", symbolCount)
	fmt.Printf("  Duration:          %s\n", duration.Round(time.Millisecond))

	return nil
}

// outputIndexJSON outputs the result in JSON format
func outputIndexJSON(result IndexResultJSON) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// outputIndexError outputs an error in JSON format if --json is set, otherwise returns the error
func outputIndexError(err error) error {
	if indexJSON {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(map[string]string{"error": err.Error()})
		os.Exit(1)
	}
	return err
}
