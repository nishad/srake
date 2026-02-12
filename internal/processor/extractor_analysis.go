package processor

import (
	"context"
	"encoding/xml"
	"io"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// ExtractAnalyses extracts analysis data
func (ce *ComprehensiveExtractor) ExtractAnalyses(ctx context.Context, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var analysisSet parser.AnalysisSet
		if err := decoder.Decode(&analysisSet); err != nil {
			if err == io.EOF {
				break
			}
			decoder = xml.NewDecoder(reader)
			var analysis parser.Analysis
			if err := decoder.Decode(&analysis); err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			analysisSet.Analyses = []parser.Analysis{analysis}
		}

		for _, analysis := range analysisSet.Analyses {
			ce.stats.AnalysesProcessed++
			dbAnalysis := ce.extractAnalysisData(analysis)

			if err := ce.db.InsertAnalysis(dbAnalysis); err != nil {
				ce.stats.Errors = append(ce.stats.Errors, err.Error())
			} else {
				ce.stats.AnalysesExtracted++
			}
		}
	}

	return nil
}

// extractAnalysisData extracts data from an Analysis
func (ce *ComprehensiveExtractor) extractAnalysisData(analysis parser.Analysis) *database.Analysis {
	dbAnalysis := &database.Analysis{
		AnalysisAccession:  analysis.Accession,
		Alias:              analysis.Alias,
		CenterName:         analysis.CenterName,
		BrokerName:         analysis.BrokerName,
		AnalysisCenter:     analysis.AnalysisCenter,
		Title:              analysis.Title,
		Description:        analysis.Description,
		StudyAccession:     analysis.StudyRef.Accession,
		AnalysisType:       ce.extractAnalysisTypeName(analysis.AnalysisType),
		Metadata:           "{}",
		Targets:            "[]",
		DataBlocks:         "[]",
		AssemblyRef:        "{}",
		RunLabels:          "[]",
		SeqLabels:          "[]",
		Processing:         "{}",
		AnalysisLinks:      "[]",
		AnalysisAttributes: "[]",
	}

	// Parse analysis date
	if analysis.AnalysisDate != "" {
		if t := parser.ParseTime(analysis.AnalysisDate); !t.IsZero() {
			dbAnalysis.AnalysisDate = &t
		}
	}

	// Extract targets
	if analysis.Targets != nil && len(analysis.Targets.Targets) > 0 {
		targets := []map[string]interface{}{}
		for _, target := range analysis.Targets.Targets {
			targetMap := map[string]interface{}{
				"sra_object_type": target.SraObjectType,
				"accession":       target.Accession,
				"refname":         target.RefName,
				"refcenter":       target.RefCenter,
			}
			targets = append(targets, targetMap)
		}
		dbAnalysis.Targets = marshalJSON(targets)
	}

	// Extract data blocks
	if len(analysis.DataBlocks) > 0 {
		dataBlocks := []map[string]interface{}{}
		for _, block := range analysis.DataBlocks {
			blockMap := map[string]interface{}{
				"name":   block.Name,
				"serial": block.Serial,
				"member": block.Member,
			}
			files := []map[string]string{}
			for _, file := range block.Files {
				fileMap := map[string]string{
					"filename":        file.Filename,
					"filetype":        file.FileType,
					"checksum":        file.Checksum,
					"checksum_method": file.ChecksumMethod,
				}
				files = append(files, fileMap)
			}
			blockMap["files"] = files
			dataBlocks = append(dataBlocks, blockMap)
		}
		dbAnalysis.DataBlocks = marshalJSON(dataBlocks)
	}

	// Extract processing information
	processingInfo := ce.extractProcessingInfo(analysis)
	dbAnalysis.Processing = marshalJSON(processingInfo)

	// Extract assembly reference for reference alignment
	if analysis.AnalysisType.ReferenceAlignment != nil {
		assemblyRef := ce.extractAssemblyRef(analysis.AnalysisType.ReferenceAlignment.Assembly)
		dbAnalysis.AssemblyRef = marshalJSON(assemblyRef)

		// Extract run labels
		if analysis.AnalysisType.ReferenceAlignment.RunLabels != nil {
			runLabels := []map[string]string{}
			for _, rl := range analysis.AnalysisType.ReferenceAlignment.RunLabels.Runs {
				runLabels = append(runLabels, map[string]string{
					"accession":        rl.Accession,
					"read_group_label": rl.ReadGroupLabel,
				})
			}
			dbAnalysis.RunLabels = marshalJSON(runLabels)
		}

		// Extract seq labels
		if analysis.AnalysisType.ReferenceAlignment.SeqLabels != nil {
			seqLabels := []map[string]string{}
			for _, sl := range analysis.AnalysisType.ReferenceAlignment.SeqLabels.Sequences {
				seqLabels = append(seqLabels, map[string]string{
					"accession": sl.Accession,
					"seq_label": sl.SeqLabel,
				})
			}
			dbAnalysis.SeqLabels = marshalJSON(seqLabels)
		}
	}

	// Build metadata
	metadata := map[string]interface{}{
		"alias":           analysis.Alias,
		"center_name":     analysis.CenterName,
		"analysis_center": analysis.AnalysisCenter,
		"analysis_date":   analysis.AnalysisDate,
		"analysis_type":   dbAnalysis.AnalysisType,
	}

	dbAnalysis.Metadata = marshalJSON(metadata)
	return dbAnalysis
}

// extractAssemblyRef extracts assembly reference information
func (ce *ComprehensiveExtractor) extractAssemblyRef(assembly parser.Assembly) map[string]interface{} {
	ref := map[string]interface{}{}

	if assembly.Standard != nil {
		ref["type"] = "standard"
		ref["ref_name"] = assembly.Standard.ShortName
		if len(assembly.Standard.Names) > 0 {
			names := []map[string]string{}
			for _, n := range assembly.Standard.Names {
				names = append(names, map[string]string{
					"db": n.DB,
					"id": n.ID,
				})
			}
			ref["names"] = names
		}
	} else if assembly.Custom != nil {
		ref["type"] = "CUSTOM"
		ref["description"] = assembly.Custom.Description
	}

	return ref
}

// extractAnalysisTypeName determines the analysis type name
func (ce *ComprehensiveExtractor) extractAnalysisTypeName(analysisType parser.AnalysisType) string {
	if analysisType.DeNovoAssembly != nil {
		return "DE_NOVO_ASSEMBLY"
	} else if analysisType.ReferenceAlignment != nil {
		return "REFERENCE_ALIGNMENT"
	} else if analysisType.SequenceAnnotation != nil {
		return "SEQUENCE_ANNOTATION"
	} else if analysisType.AbundanceMeasurement != nil {
		return "ABUNDANCE_MEASUREMENT"
	}
	return "UNKNOWN"
}

// extractProcessingInfo extracts processing/pipeline information
func (ce *ComprehensiveExtractor) extractProcessingInfo(analysis parser.Analysis) map[string]interface{} {
	processingInfo := map[string]interface{}{}

	if analysis.AnalysisType.DeNovoAssembly != nil {
		processingInfo["type"] = "de_novo_assembly"
		programs := ce.extractPipeline(analysis.AnalysisType.DeNovoAssembly.Processing.Pipeline)
		processingInfo["programs"] = programs
		if len(programs) > 0 {
			processingInfo["pipeline_name"] = programs[0]["name"]
		}
	} else if analysis.AnalysisType.ReferenceAlignment != nil {
		processingInfo["type"] = "reference_alignment"
		programs := ce.extractPipeline(analysis.AnalysisType.ReferenceAlignment.Processing.Pipeline)
		processingInfo["programs"] = programs
		if len(programs) > 0 {
			processingInfo["pipeline_name"] = programs[0]["name"]
		}
	} else if analysis.AnalysisType.SequenceAnnotation != nil {
		processingInfo["type"] = "sequence_annotation"
		programs := ce.extractPipeline(analysis.AnalysisType.SequenceAnnotation.Processing.Pipeline)
		processingInfo["programs"] = programs
		if len(programs) > 0 {
			processingInfo["pipeline_name"] = programs[0]["name"]
		}
	} else if analysis.AnalysisType.AbundanceMeasurement != nil {
		processingInfo["type"] = "abundance_measurement"
		programs := ce.extractPipeline(analysis.AnalysisType.AbundanceMeasurement.Processing.Pipeline)
		processingInfo["programs"] = programs
		if len(programs) > 0 {
			processingInfo["pipeline_name"] = programs[0]["name"]
		}
	}

	return processingInfo
}

// extractPipeline extracts pipeline information
func (ce *ComprehensiveExtractor) extractPipeline(pipeline parser.PipelineType) []map[string]string {
	var programs []map[string]string
	for _, program := range pipeline.Programs {
		programs = append(programs, map[string]string{
			"name":    program.Name,
			"version": program.Version,
		})
	}
	return programs
}
