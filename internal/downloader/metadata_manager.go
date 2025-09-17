package downloader

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// NCBI FTP base URLs
	NCBIMetadataBaseURL = "https://ftp.ncbi.nlm.nih.gov/sra/reports/Metadata/"

	// File patterns
	DailyFilePattern   = `NCBI_SRA_Metadata_(\d{8})\.tar\.gz`
	MonthlyFilePattern = `NCBI_SRA_Metadata_Full_(\d{8})\.tar\.gz`
)

// MetadataFile represents an available metadata file
type MetadataFile struct {
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Size         int64     `json:"size"`
	Date         time.Time `json:"date"`
	Type         FileType  `json:"type"`
	IsCompressed bool      `json:"is_compressed"`
}

// FileType represents the type of metadata file
type FileType string

const (
	FileTypeDaily   FileType = "daily"
	FileTypeMonthly FileType = "monthly"
	FileTypeUnknown FileType = "unknown"
)

// MetadataManager discovers and manages metadata files from NCBI
type MetadataManager struct {
	client  *http.Client
	baseURL string
}

// NewMetadataManager creates a new metadata manager
func NewMetadataManager() *MetadataManager {
	return &MetadataManager{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: NCBIMetadataBaseURL,
	}
}

// ListAvailableFiles discovers all available metadata files
func (mm *MetadataManager) ListAvailableFiles(ctx context.Context) ([]MetadataFile, error) {
	// Fetch directory listing
	req, err := http.NewRequestWithContext(ctx, "GET", mm.baseURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := mm.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch directory listing: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %s", resp.Status)
	}

	// Read HTML content
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse HTML to extract file information
	files := mm.parseDirectoryListing(string(body))

	// Sort files by date (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].Date.After(files[j].Date)
	})

	return files, nil
}

// parseDirectoryListing parses the HTML directory listing from NCBI FTP
func (mm *MetadataManager) parseDirectoryListing(html string) []MetadataFile {
	var files []MetadataFile

	// Regular expressions for parsing HTML table rows
	// NCBI FTP uses Apache directory listing format
	rowPattern := regexp.MustCompile(`<a href="([^"]+\.tar\.gz)"[^>]*>([^<]+)</a>\s+(\d{2}-\w{3}-\d{4}\s+\d{2}:\d{2})\s+(\d+[KMG]?)`)

	// Alternative pattern for different formatting
	altPattern := regexp.MustCompile(`href="([^"]+\.tar\.gz)"[^>]*>.*?</a>.*?(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2})\s+(\d+)`)

	// Find all matches
	matches := rowPattern.FindAllStringSubmatch(html, -1)
	if len(matches) == 0 {
		matches = altPattern.FindAllStringSubmatch(html, -1)
	}

	dailyRegex := regexp.MustCompile(DailyFilePattern)
	monthlyRegex := regexp.MustCompile(MonthlyFilePattern)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		filename := match[1]

		// Skip if not a metadata file we're interested in
		if !strings.Contains(filename, "NCBI_SRA_Metadata") {
			continue
		}

		// Determine file type
		var fileType FileType
		var date time.Time

		if monthlyRegex.MatchString(filename) {
			fileType = FileTypeMonthly
			dateMatch := monthlyRegex.FindStringSubmatch(filename)
			if len(dateMatch) > 1 {
				date = parseDate(dateMatch[1])
			}
		} else if dailyRegex.MatchString(filename) {
			fileType = FileTypeDaily
			dateMatch := dailyRegex.FindStringSubmatch(filename)
			if len(dateMatch) > 1 {
				date = parseDate(dateMatch[1])
			}
		} else {
			fileType = FileTypeUnknown
			// Try to parse date from file modification time
			date = parseModTime(match[2])
		}

		// Parse size
		size := parseSize(match[3])

		file := MetadataFile{
			Name:         filename,
			URL:          mm.baseURL + filename,
			Size:         size,
			Date:         date,
			Type:         fileType,
			IsCompressed: strings.HasSuffix(filename, ".gz"),
		}

		files = append(files, file)
	}

	return files
}

// GetLatestFile returns the most recent metadata file
func (mm *MetadataManager) GetLatestFile(ctx context.Context, fileType FileType) (*MetadataFile, error) {
	files, err := mm.ListAvailableFiles(ctx)
	if err != nil {
		return nil, err
	}

	// Filter by type if specified
	if fileType != FileTypeUnknown {
		filtered := []MetadataFile{}
		for _, f := range files {
			if f.Type == fileType {
				filtered = append(filtered, f)
			}
		}
		files = filtered
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files found")
	}

	// Files are already sorted by date (newest first)
	return &files[0], nil
}

// AutoSelectFile implements smart auto-selection logic
func (mm *MetadataManager) AutoSelectFile(ctx context.Context) (*MetadataFile, error) {
	files, err := mm.ListAvailableFiles(ctx)
	if err != nil {
		return nil, err
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no files available")
	}

	now := time.Now()

	// Strategy:
	// 1. If a monthly file exists from the current month, use it
	// 2. Otherwise, use the most recent daily file
	// 3. If no daily files from last 7 days, use most recent monthly

	var latestMonthly *MetadataFile
	var latestDaily *MetadataFile

	for i := range files {
		file := &files[i]

		switch file.Type {
		case FileTypeMonthly:
			if latestMonthly == nil {
				latestMonthly = file
			}
			// Check if it's from current month
			if file.Date.Year() == now.Year() && file.Date.Month() == now.Month() {
				return file, nil
			}
		case FileTypeDaily:
			if latestDaily == nil {
				latestDaily = file
				// If daily file is from last 7 days, prefer it
				if now.Sub(file.Date) < 7*24*time.Hour {
					return file, nil
				}
			}
		}
	}

	// Fallback: use most recent monthly if available
	if latestMonthly != nil {
		return latestMonthly, nil
	}

	// Otherwise use most recent daily
	if latestDaily != nil {
		return latestDaily, nil
	}

	// Last resort: return first file
	return &files[0], nil
}

// GetFileByName returns a specific file by name
func (mm *MetadataManager) GetFileByName(ctx context.Context, name string) (*MetadataFile, error) {
	files, err := mm.ListAvailableFiles(ctx)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if f.Name == name {
			return &f, nil
		}
	}

	return nil, fmt.Errorf("file not found: %s", name)
}

// Helper functions

// parseDate parses date from YYYYMMDD format
func parseDate(dateStr string) time.Time {
	if len(dateStr) != 8 {
		return time.Time{}
	}

	year, _ := strconv.Atoi(dateStr[0:4])
	month, _ := strconv.Atoi(dateStr[4:6])
	day, _ := strconv.Atoi(dateStr[6:8])

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

// parseModTime parses modification time from various formats
func parseModTime(timeStr string) time.Time {
	// Try different date formats
	formats := []string{
		"02-Jan-2006 15:04",
		"2006-01-02 15:04",
		"Jan 2, 2006 3:04 PM",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// parseSize parses file size from string (e.g., "1.5G", "500M", "1024")
func parseSize(sizeStr string) int64 {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0
	}

	// Check for unit suffix
	multiplier := int64(1)
	lastChar := sizeStr[len(sizeStr)-1]

	switch lastChar {
	case 'K', 'k':
		multiplier = 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case 'M', 'm':
		multiplier = 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	case 'G', 'g':
		multiplier = 1024 * 1024 * 1024
		sizeStr = sizeStr[:len(sizeStr)-1]
	}

	// Parse numeric part (can be float)
	if strings.Contains(sizeStr, ".") {
		if val, err := strconv.ParseFloat(sizeStr, 64); err == nil {
			return int64(val * float64(multiplier))
		}
	} else {
		if val, err := strconv.ParseInt(sizeStr, 10, 64); err == nil {
			return val * multiplier
		}
	}

	return 0
}

// FormatSize formats bytes as human-readable string
func FormatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration in human-readable form
func FormatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	}
	if d < time.Hour {
		return fmt.Sprintf("%.1f minutes", d.Minutes())
	}
	return fmt.Sprintf("%.1f hours", d.Hours())
}
