package processor

import (
	"context"
	"encoding/xml"
	"io"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// ExtractStudies extracts comprehensive study data
func (ce *ComprehensiveExtractor) ExtractStudies(ctx context.Context, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)
	var batch []database.Study

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Try to decode as StudySet first
		var studySet parser.StudySet
		if err := decoder.Decode(&studySet); err != nil {
			if err == io.EOF {
				break
			}
			// Try single study
			decoder = xml.NewDecoder(reader)
			var study parser.Study
			if err := decoder.Decode(&study); err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			studySet.Studies = []parser.Study{study}
		}

		for _, study := range studySet.Studies {
			ce.stats.StudiesProcessed++

			dbStudy := ce.extractStudyData(study)
			batch = append(batch, *dbStudy)

			if len(batch) >= ce.options.BatchSize {
				if err := ce.insertStudyBatch(batch); err != nil {
					ce.stats.Errors = append(ce.stats.Errors, err.Error())
				} else {
					ce.stats.StudiesExtracted += len(batch)
				}
				batch = batch[:0]
			}
		}
	}

	// Insert remaining batch
	if len(batch) > 0 {
		if err := ce.insertStudyBatch(batch); err != nil {
			ce.stats.Errors = append(ce.stats.Errors, err.Error())
		} else {
			ce.stats.StudiesExtracted += len(batch)
		}
	}

	return nil
}

// extractStudyData extracts comprehensive data from a Study
func (ce *ComprehensiveExtractor) extractStudyData(study parser.Study) *database.Study {
	dbStudy := &database.Study{
		StudyAccession:   study.Accession,
		Alias:            study.Alias,
		CenterName:       study.CenterName,
		BrokerName:       study.BrokerName,
		StudyTitle:       study.Descriptor.StudyTitle,
		StudyAbstract:    study.Descriptor.StudyAbstract,
		StudyDescription: study.Descriptor.StudyDescription,
		StudyType:        "",
		Metadata:         "{}",
		// Initialize JSON fields as empty arrays
		SecondaryIDs:    "[]",
		ExternalIDs:     "[]",
		SubmitterIDs:    "[]",
		StudyLinks:      "[]",
		StudyAttributes: "[]",
		RelatedStudies:  "[]",
	}

	// Extract study type
	if study.Descriptor.StudyType != nil {
		dbStudy.StudyType = study.Descriptor.StudyType.ExistingStudyType
		if dbStudy.StudyType == "Other" && study.Descriptor.StudyType.NewStudyType != "" {
			dbStudy.StudyType = study.Descriptor.StudyType.NewStudyType
		}
	}

	// Build comprehensive metadata
	metadata := map[string]interface{}{
		"alias":               study.Alias,
		"center_name":         study.CenterName,
		"broker_name":         study.BrokerName,
		"center_project_name": study.Descriptor.CenterProjectName,
		"study_description":   study.Descriptor.StudyDescription,
	}

	// Extract and store identifiers
	if study.Identifiers != nil {
		ids := ce.extractIdentifiers(study.Identifiers)
		metadata["identifiers"] = ids

		// Store individual ID fields
		if study.Identifiers.PrimaryID != nil {
			dbStudy.PrimaryID = study.Identifiers.PrimaryID.Value
		}

		// Secondary IDs
		if len(study.Identifiers.SecondaryIDs) > 0 {
			secondary := []map[string]string{}
			for _, id := range study.Identifiers.SecondaryIDs {
				secondary = append(secondary, map[string]string{
					"label": id.Label,
					"value": id.Value,
				})
			}
			dbStudy.SecondaryIDs = marshalJSON(secondary)
		}

		// External IDs
		if len(study.Identifiers.ExternalIDs) > 0 {
			external := []map[string]string{}
			for _, id := range study.Identifiers.ExternalIDs {
				external = append(external, map[string]string{
					"namespace": id.Namespace,
					"label":     id.Label,
					"value":     id.Value,
				})
			}
			dbStudy.ExternalIDs = marshalJSON(external)
		}

		// Submitter IDs
		if len(study.Identifiers.SubmitterIDs) > 0 {
			submitter := []map[string]string{}
			for _, id := range study.Identifiers.SubmitterIDs {
				submitter = append(submitter, map[string]string{
					"namespace": id.Namespace,
					"label":     id.Label,
					"value":     id.Value,
				})
			}
			dbStudy.SubmitterIDs = marshalJSON(submitter)
		}
	}

	// Extract related studies
	if study.Descriptor.RelatedStudies != nil {
		related := []map[string]interface{}{}
		for _, rs := range study.Descriptor.RelatedStudies.RelatedStudy {
			related = append(related, map[string]interface{}{
				"db":         rs.RelatedLink.DB,
				"id":         rs.RelatedLink.ID,
				"is_primary": rs.IsPrimary,
			})
		}
		metadata["related_studies"] = related
		dbStudy.RelatedStudies = marshalJSON(related)
	}

	// Extract links
	if ce.options.ExtractLinks && study.StudyLinks != nil {
		links := ce.extractLinks(study.StudyLinks.Links)
		if len(links) > 0 {
			metadata["links"] = links
			dbStudy.StudyLinks = marshalJSON(links)
		}
	}

	// Extract attributes
	if ce.options.ExtractAttributes && study.StudyAttributes != nil {
		attrs := ce.extractAttributes(study.StudyAttributes.Attributes)
		if len(attrs) > 0 {
			metadata["attributes"] = attrs
			dbStudy.StudyAttributes = marshalJSON(attrs)
		}

		// Try to extract organism from attributes if enabled
		if ce.options.ExtractFromAttributes {
			for _, attr := range study.StudyAttributes.Attributes {
				if attr.Tag == "organism" || attr.Tag == "scientific_name" {
					dbStudy.Organism = ce.normalizeOrganism(attr.Value)
					break
				}
			}
		}
	}

	// Convert metadata to JSON
	dbStudy.Metadata = marshalJSON(metadata)

	return dbStudy
}

// insertStudyBatch inserts a batch of studies
func (ce *ComprehensiveExtractor) insertStudyBatch(batch []database.Study) error {
	for _, study := range batch {
		if err := ce.db.InsertStudy(&study); err != nil {
			return err
		}
	}
	return nil
}
