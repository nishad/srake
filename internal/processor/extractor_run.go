package processor

import (
	"context"
	"encoding/xml"
	"io"
	"strconv"
	"strings"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// ExtractRuns extracts comprehensive run data
func (ce *ComprehensiveExtractor) ExtractRuns(ctx context.Context, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var runSet parser.RunSet
		if err := decoder.Decode(&runSet); err != nil {
			if err == io.EOF {
				break
			}
			decoder = xml.NewDecoder(reader)
			var run parser.Run
			if err := decoder.Decode(&run); err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			runSet.Runs = []parser.Run{run}
		}

		for _, run := range runSet.Runs {
			ce.stats.RunsProcessed++
			dbRun := ce.extractRunData(run)

			if err := ce.db.InsertRun(dbRun); err != nil {
				ce.stats.Errors = append(ce.stats.Errors, err.Error())
			} else {
				ce.stats.RunsExtracted++
			}
		}
	}

	return nil
}

// extractRunData extracts data from a Run
func (ce *ComprehensiveExtractor) extractRunData(run parser.Run) *database.Run {
	dbRun := &database.Run{
		RunAccession:        run.Accession,
		Alias:               run.Alias,
		CenterName:          run.CenterName,
		BrokerName:          run.BrokerName,
		RunCenter:           run.RunCenter,
		ExperimentAccession: run.ExperimentRef.Accession,
		Title:               run.Title,
		Metadata:            "{}",
		DataFiles:           "[]",
		RunLinks:            "[]",
		RunAttributes:       "[]",
	}

	// Parse run date
	if run.RunDate != "" {
		if t := parser.ParseTime(run.RunDate); !t.IsZero() {
			dbRun.RunDate = &t
		}
	}

	// Extract statistics
	if run.Statistics != nil {
		dbRun.TotalSpots = run.Statistics.TotalSpots
		dbRun.TotalBases = run.Statistics.TotalBases
		dbRun.TotalSize = run.Statistics.TotalSize
		dbRun.LoadDone = run.Statistics.LoadDone
		if run.Statistics.Published != "" {
			dbRun.Published = run.Statistics.Published
		}
	}

	// Build metadata
	metadata := map[string]interface{}{
		"alias":       run.Alias,
		"center_name": run.CenterName,
		"run_center":  run.RunCenter,
		"run_date":    run.RunDate,
		"title":       run.Title,
	}

	// Extract file information
	if run.DataBlock != nil && len(run.DataBlock.Files) > 0 {
		files := []map[string]string{}
		for _, file := range run.DataBlock.Files {
			fileMap := map[string]string{
				"filename": file.Filename,
				"filetype": file.FileType,
			}
			if file.Checksum != "" {
				fileMap["checksum"] = file.Checksum
				fileMap["checksum_method"] = file.ChecksumMethod
			}
			files = append(files, fileMap)
		}
		dbRun.DataFiles = marshalJSON(files)
		metadata["files"] = files
	}

	// Extract links and attributes
	if ce.options.ExtractLinks && run.RunLinks != nil {
		links := ce.extractLinks(run.RunLinks.Links)
		dbRun.RunLinks = marshalJSON(links)
		metadata["links"] = links
	}

	if ce.options.ExtractAttributes && run.RunAttributes != nil {
		attrs := ce.extractAttributes(run.RunAttributes.Attributes)
		dbRun.RunAttributes = marshalJSON(attrs)
		metadata["attributes"] = attrs

		// Extract known quality metrics from attributes
		if ce.options.ExtractFromAttributes {
			for _, attr := range run.RunAttributes.Attributes {
				switch strings.ToLower(attr.Tag) {
				case "quality_score_mean":
					if val, err := strconv.ParseFloat(attr.Value, 64); err == nil {
						dbRun.QualityScoreMean = val
					}
				case "quality_score_std":
					if val, err := strconv.ParseFloat(attr.Value, 64); err == nil {
						dbRun.QualityScoreStd = val
					}
				case "read_count_r1":
					if val, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
						dbRun.ReadCountR1 = val
					}
				case "read_count_r2":
					if val, err := strconv.ParseInt(attr.Value, 10, 64); err == nil {
						dbRun.ReadCountR2 = val
					}
				}
			}
		}
	}

	dbRun.Metadata = marshalJSON(metadata)
	return dbRun
}
