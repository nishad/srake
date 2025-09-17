package processor

import (
	"context"
	"encoding/xml"
	"io"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// ExtractExperiments extracts comprehensive experiment data
func (ce *ComprehensiveExtractor) ExtractExperiments(ctx context.Context, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	var batch []database.Experiment

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var expSet parser.ExperimentSet
		if err := decoder.Decode(&expSet); err != nil {
			if err == io.EOF {
				break
			}
			decoder = xml.NewDecoder(reader)
			var exp parser.Experiment
			if err := decoder.Decode(&exp); err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			expSet.Experiments = []parser.Experiment{exp}
		}

		for _, exp := range expSet.Experiments {
			ce.stats.ExperimentsProcessed++
			dbExp := ce.extractExperimentData(exp)
			batch = append(batch, *dbExp)

			// Extract and store pool relationships if present
			if ce.poolHandler != nil {
				poolRels, err := ce.poolHandler.ExtractPoolRelationships(exp)
				if err != nil {
					ce.stats.Errors = append(ce.stats.Errors, err.Error())
				} else if len(poolRels) > 0 {
					if err := ce.poolHandler.StorePoolRelationships(poolRels); err != nil {
						ce.stats.Errors = append(ce.stats.Errors, err.Error())
					}
				}
			}

			if len(batch) >= ce.options.BatchSize {
				if err := ce.db.BatchInsertExperiments(batch); err != nil {
					ce.stats.Errors = append(ce.stats.Errors, err.Error())
				} else {
					ce.stats.ExperimentsExtracted += len(batch)
				}
				batch = batch[:0]
			}
		}
	}

	if len(batch) > 0 {
		if err := ce.db.BatchInsertExperiments(batch); err != nil {
			ce.stats.Errors = append(ce.stats.Errors, err.Error())
		} else {
			ce.stats.ExperimentsExtracted += len(batch)
		}
	}

	return nil
}

// extractExperimentData extracts data from an Experiment
func (ce *ComprehensiveExtractor) extractExperimentData(exp parser.Experiment) *database.Experiment {
	// Extract platform details using the platform handler
	platformName, instrumentModel := ce.platformHandler.ExtractPlatformDetails(exp.Platform)

	dbExp := &database.Experiment{
		ExperimentAccession:         exp.Accession,
		Alias:                       exp.Alias,
		CenterName:                  exp.CenterName,
		BrokerName:                  exp.BrokerName,
		Title:                       exp.Title,
		StudyAccession:              exp.StudyRef.Accession,
		SampleAccession:             exp.Design.SampleDescriptor.Accession,
		DesignDescription:           exp.Design.DesignDescription,
		Platform:                    platformName,
		InstrumentModel:             instrumentModel,
		LibraryName:                 exp.Design.LibraryDescriptor.LibraryName,
		LibraryStrategy:             exp.Design.LibraryDescriptor.LibraryStrategy,
		LibrarySource:               exp.Design.LibraryDescriptor.LibrarySource,
		LibrarySelection:            exp.Design.LibraryDescriptor.LibrarySelection,
		LibraryLayout:               extractLibraryLayout(exp.Design.LibraryDescriptor.LibraryLayout),
		LibraryConstructionProtocol: exp.Design.LibraryDescriptor.LibraryConstructionProtocol,
		Metadata:                    "{}",
	}

	// Extract paired-end info
	if exp.Design.LibraryDescriptor.LibraryLayout.Paired != nil {
		dbExp.NominalLength = exp.Design.LibraryDescriptor.LibraryLayout.Paired.NominalLength
		dbExp.NominalSdev = exp.Design.LibraryDescriptor.LibraryLayout.Paired.NominalSdev
	}

	// Extract spot descriptor
	if exp.Design.SpotDescriptor != nil {
		spotDesc := extractSpotDescriptor(exp.Design.SpotDescriptor)
		dbExp.SpotDecodeSpec = marshalJSON(spotDesc)
		if exp.Design.SpotDescriptor.SpotDecodeSpec != nil {
			dbExp.SpotLength = exp.Design.SpotDescriptor.SpotDecodeSpec.SpotLength
		}
	}

	// Extract pool information
	if exp.Design.SampleDescriptor.Pool != nil {
		poolInfo := extractPoolInfo(exp.Design.SampleDescriptor.Pool)
		dbExp.PoolInfo = marshalJSON(poolInfo)
		dbExp.PoolMemberCount = len(exp.Design.SampleDescriptor.Pool.Members)
		if exp.Design.SampleDescriptor.Pool.DefaultMember != nil {
			dbExp.PoolMemberCount++
		}
	}

	// Build metadata
	metadata := map[string]interface{}{
		"title":             exp.Title,
		"platform":          dbExp.Platform,
		"instrument_model":  dbExp.InstrumentModel,
		"library_strategy":  dbExp.LibraryStrategy,
		"library_source":    dbExp.LibrarySource,
		"library_selection": dbExp.LibrarySelection,
		"library_layout":    dbExp.LibraryLayout,
	}

	// Extract links and attributes
	if ce.options.ExtractLinks && exp.ExperimentLinks != nil {
		links := ce.extractLinks(exp.ExperimentLinks.Links)
		dbExp.ExperimentLinks = marshalJSON(links)
		metadata["links"] = links
	}

	if ce.options.ExtractAttributes && exp.ExperimentAttributes != nil {
		attrs := ce.extractAttributes(exp.ExperimentAttributes.Attributes)
		dbExp.ExperimentAttributes = marshalJSON(attrs)
		metadata["attributes"] = attrs
	}

	dbExp.Metadata = marshalJSON(metadata)
	return dbExp
}

// extractPoolInfo extracts pool information
func extractPoolInfo(pool *parser.Pool) map[string]interface{} {
	info := make(map[string]interface{})

	var members []map[string]interface{}
	if pool.DefaultMember != nil {
		member := map[string]interface{}{
			"accession": pool.DefaultMember.Accession,
			"name":      pool.DefaultMember.MemberName,
		}
		if pool.DefaultMember.Proportion > 0 {
			member["proportion"] = pool.DefaultMember.Proportion
		}
		members = append(members, member)
	}

	for _, m := range pool.Members {
		member := map[string]interface{}{
			"accession": m.Accession,
			"name":      m.MemberName,
		}
		if m.Proportion > 0 {
			member["proportion"] = m.Proportion
		}
		members = append(members, member)
	}

	if len(members) > 0 {
		info["members"] = members
	}

	return info
}
