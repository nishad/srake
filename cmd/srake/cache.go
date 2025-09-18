package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/nishad/srake/internal/paths"
	"github.com/spf13/cobra"
)

var cacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage srake cache",
	Long:  `Manage cache files used by srake including downloads, search indices, and embeddings.`,
}

var cacheCleanCmd = &cobra.Command{
	Use:   "clean",
	Short: "Clean cache files",
	Long: `Remove cache files to free up disk space.

By default, this removes only old download files. Use --all to remove
all cache including search indices which will need to be rebuilt.`,
	Example: `  # Remove downloads older than 30 days
  srake cache clean --older 30d

  # Remove all cache including indices
  srake cache clean --all

  # Remove only search cache
  srake cache clean --search`,
	RunE: runCacheClean,
}

var cacheInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show cache information",
	Long:  `Display information about cache directories and their sizes.`,
	RunE:  runCacheInfo,
}

var (
	cleanAll      bool
	cleanOlder    string
	cleanSearch   bool
	cleanDownload bool
	cleanIndex    bool
)

func init() {
	cacheCleanCmd.Flags().BoolVar(&cleanAll, "all", false, "Remove all cache including indices")
	cacheCleanCmd.Flags().StringVar(&cleanOlder, "older", "", "Remove files older than duration (e.g. 30d, 24h)")
	cacheCleanCmd.Flags().BoolVar(&cleanSearch, "search", false, "Remove search result cache")
	cacheCleanCmd.Flags().BoolVar(&cleanDownload, "downloads", false, "Remove downloaded files")
	cacheCleanCmd.Flags().BoolVar(&cleanIndex, "index", false, "Remove search index (will need rebuild)")

	cacheCmd.AddCommand(cacheCleanCmd)
	cacheCmd.AddCommand(cacheInfoCmd)
}

func runCacheClean(cmd *cobra.Command, args []string) error {
	cacheDir := paths.GetPaths().CacheDir

	if cleanAll {
		printWarning("Cleaning all cache in %s", cacheDir)
		fmt.Print("This will remove all cached data including search indices.\nContinue? [y/N]: ")

		if !yes {
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		if err := os.RemoveAll(cacheDir); err != nil {
			return fmt.Errorf("failed to clean cache: %w", err)
		}

		// Recreate the directory structure
		if err := paths.EnsureDirectories(); err != nil {
			return fmt.Errorf("failed to recreate directories: %w", err)
		}

		printSuccess("All cache cleaned")
		return nil
	}

	// Clean specific cache types
	var cleaned int64

	if cleanSearch {
		searchCache := filepath.Join(cacheDir, "search")
		if size, err := cleanDirectory(searchCache); err == nil {
			cleaned += size
			printInfo("Cleaned search cache: %.2f MB", float64(size)/(1024*1024))
		}
	}

	if cleanDownload || cleanOlder != "" {
		downloadsDir := filepath.Join(cacheDir, "downloads")

		if cleanOlder != "" {
			duration, err := parseDuration(cleanOlder)
			if err != nil {
				return fmt.Errorf("invalid duration: %w", err)
			}

			cutoff := time.Now().Add(-duration)
			size, count, err := cleanOldFiles(downloadsDir, cutoff)
			if err != nil {
				return fmt.Errorf("failed to clean old files: %w", err)
			}

			cleaned += size
			printInfo("Removed %d files older than %s (%.2f MB)",
				count, cleanOlder, float64(size)/(1024*1024))
		} else if cleanDownload {
			if size, err := cleanDirectory(downloadsDir); err == nil {
				cleaned += size
				printInfo("Cleaned downloads: %.2f MB", float64(size)/(1024*1024))
			}
		}
	}

	if cleanIndex {
		indexPath := paths.GetIndexPath()
		indexDir := filepath.Dir(indexPath)

		printWarning("Removing search index at %s", indexPath)
		fmt.Println("You will need to rebuild the index with: srake search index --build")

		if !yes {
			fmt.Print("Continue? [y/N]: ")
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Cancelled")
				return nil
			}
		}

		if size, err := cleanDirectory(indexDir); err == nil {
			cleaned += size
			printInfo("Cleaned index: %.2f MB", float64(size)/(1024*1024))
		}
	}

	if cleaned == 0 {
		fmt.Println("No cache to clean. Use --help for options.")
		return nil
	}

	printSuccess("Total cleaned: %.2f MB", float64(cleaned)/(1024*1024))
	return nil
}

func runCacheInfo(cmd *cobra.Command, args []string) error {
	p := paths.GetPaths()

	printInfo("Cache Directories")
	fmt.Println(colorize(colorGray, "────────────────────────────────────────"))

	// Show base directories
	fmt.Printf("%s\n", colorize(colorBold, "Base Paths:"))
	fmt.Printf("  Config:     %s\n", p.ConfigDir)
	fmt.Printf("  Data:       %s\n", p.DataDir)
	fmt.Printf("  Cache:      %s\n", p.CacheDir)
	fmt.Printf("  State:      %s\n", p.StateDir)

	fmt.Println()
	fmt.Printf("%s\n", colorize(colorBold, "Cache Subdirectories:"))

	// Calculate sizes for each cache directory
	type dirInfo struct {
		path  string
		name  string
		size  int64
		count int
	}

	dirs := []dirInfo{
		{filepath.Join(p.CacheDir, "downloads"), "Downloads", 0, 0},
		{filepath.Join(p.CacheDir, "index"), "Search Index", 0, 0},
		{filepath.Join(p.CacheDir, "embeddings"), "Embeddings", 0, 0},
		{filepath.Join(p.CacheDir, "search"), "Search Cache", 0, 0},
	}

	var totalSize int64
	for i, dir := range dirs {
		size, count := getDirStats(dir.path)
		dirs[i].size = size
		dirs[i].count = count
		totalSize += size

		if size > 0 {
			fmt.Printf("  %-15s %s (%.2f MB, %d files)\n",
				dir.name+":",
				colorize(colorCyan, dir.path),
				float64(size)/(1024*1024),
				count)
		} else {
			fmt.Printf("  %-15s %s (empty)\n",
				dir.name+":",
				colorize(colorGray, dir.path))
		}
	}

	fmt.Println()
	fmt.Printf("%s %.2f MB\n",
		colorize(colorBold, "Total cache size:"),
		float64(totalSize)/(1024*1024))

	// Show environment variables if set
	envVars := []string{
		"SRAKE_CONFIG_HOME",
		"SRAKE_DATA_HOME",
		"SRAKE_CACHE_HOME",
		"SRAKE_STATE_HOME",
		"SRAKE_DB_PATH",
		"SRAKE_INDEX_PATH",
		"SRAKE_MODELS_PATH",
	}

	hasEnv := false
	for _, env := range envVars {
		if os.Getenv(env) != "" {
			hasEnv = true
			break
		}
	}

	if hasEnv {
		fmt.Println()
		fmt.Printf("%s\n", colorize(colorBold, "Environment Variables:"))
		for _, env := range envVars {
			if val := os.Getenv(env); val != "" {
				fmt.Printf("  %s=%s\n", env, colorize(colorCyan, val))
			}
		}
	}

	return nil
}

// Helper functions

func cleanDirectory(dir string) (int64, error) {
	var totalSize int64

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			totalSize += info.Size()
			if removeErr := os.Remove(path); removeErr != nil {
				// Log but continue
				if verbose {
					fmt.Printf("Warning: could not remove %s: %v\n", path, removeErr)
				}
			}
		}
		return nil
	})

	// Try to remove empty directories
	os.RemoveAll(dir)

	return totalSize, err
}

func cleanOldFiles(dir string, cutoff time.Time) (int64, int, error) {
	var totalSize int64
	var count int

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		if !info.IsDir() && info.ModTime().Before(cutoff) {
			totalSize += info.Size()
			count++

			if verbose {
				fmt.Printf("Removing %s (%.2f MB, %s old)\n",
					path,
					float64(info.Size())/(1024*1024),
					time.Since(info.ModTime()).Round(time.Hour))
			}

			if removeErr := os.Remove(path); removeErr != nil {
				if verbose {
					fmt.Printf("Warning: could not remove %s: %v\n", path, removeErr)
				}
			}
		}
		return nil
	})

	return totalSize, count, err
}

func getDirStats(dir string) (int64, int) {
	var size int64
	var count int

	filepath.Walk(dir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
			count++
		}
		return nil
	})

	return size, count
}

func parseDuration(s string) (time.Duration, error) {
	// Handle common suffixes like "30d" for days
	if len(s) > 1 && s[len(s)-1] == 'd' {
		days := s[:len(s)-1]
		var d int
		if _, err := fmt.Sscanf(days, "%d", &d); err != nil {
			return 0, err
		}
		return time.Duration(d) * 24 * time.Hour, nil
	}

	// Try standard duration parsing
	return time.ParseDuration(s)
}