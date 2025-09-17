package processor

import (
	"context"
	"encoding/xml"
	"io"
	"strings"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// ExtractSamples extracts comprehensive sample data
func (ce *ComprehensiveExtractor) ExtractSamples(ctx context.Context, reader io.Reader) error {
	decoder := xml.NewDecoder(reader)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var sampleSet parser.SampleSet
		if err := decoder.Decode(&sampleSet); err != nil {
			if err == io.EOF {
				break
			}
			decoder = xml.NewDecoder(reader)
			var sample parser.Sample
			if err := decoder.Decode(&sample); err != nil {
				if err == io.EOF {
					break
				}
				continue
			}
			sampleSet.Samples = []parser.Sample{sample}
		}

		for _, sample := range sampleSet.Samples {
			ce.stats.SamplesProcessed++
			dbSample := ce.extractSampleData(sample)

			if err := ce.db.InsertSample(dbSample); err != nil {
				ce.stats.Errors = append(ce.stats.Errors, err.Error())
			} else {
				ce.stats.SamplesExtracted++
			}
		}
	}

	return nil
}

// extractSampleData extracts data from a Sample
func (ce *ComprehensiveExtractor) extractSampleData(sample parser.Sample) *database.Sample {
	dbSample := &database.Sample{
		SampleAccession: sample.Accession,
		Alias:           sample.Alias,
		CenterName:      sample.CenterName,
		BrokerName:      sample.BrokerName,
		Title:           sample.Title,
		TaxonID:         sample.SampleName.TaxonID,
		ScientificName:  ce.normalizeOrganism(sample.SampleName.ScientificName),
		CommonName:      sample.SampleName.CommonName,
		Description:     sample.Description,
		Metadata:        "{}",
	}

	// Use scientific name as organism
	dbSample.Organism = dbSample.ScientificName

	// Build metadata
	metadata := map[string]interface{}{
		"alias":           sample.Alias,
		"taxon_id":        sample.SampleName.TaxonID,
		"scientific_name": sample.SampleName.ScientificName,
		"common_name":     sample.SampleName.CommonName,
	}

	// Extract attributes
	if ce.options.ExtractAttributes && sample.SampleAttributes != nil {
		attrs := ce.extractAttributes(sample.SampleAttributes.Attributes)
		metadata["attributes"] = attrs
		dbSample.SampleAttributes = marshalJSON(attrs)

		// Extract known fields from attributes
		if ce.options.ExtractFromAttributes {
			for _, attr := range sample.SampleAttributes.Attributes {
				switch strings.ToLower(attr.Tag) {
				case "strain":
					dbSample.Strain = attr.Value
				case "sex", "gender":
					dbSample.Sex = attr.Value
				case "age":
					if attr.Units != "" {
						dbSample.Age = attr.Value + " " + attr.Units
					} else {
						dbSample.Age = attr.Value
					}
				case "disease", "disease_state":
					dbSample.Disease = attr.Value
				case "treatment":
					dbSample.Treatment = attr.Value
				case "geo_loc_name", "geographic_location":
					dbSample.GeoLocName = attr.Value
				case "lat_lon":
					dbSample.LatLon = attr.Value
				case "collection_date":
					dbSample.CollectionDate = attr.Value
				case "env_biome":
					dbSample.EnvBiome = attr.Value
				case "env_feature":
					dbSample.EnvFeature = attr.Value
				case "env_material":
					dbSample.EnvMaterial = attr.Value
				case "biosample":
					dbSample.BiosampleAccession = attr.Value
				case "bioproject":
					dbSample.BioprojectAccession = attr.Value
				}
			}
		}
	}

	// Extract links
	if ce.options.ExtractLinks && sample.SampleLinks != nil {
		links := ce.extractLinks(sample.SampleLinks.Links)
		metadata["links"] = links
		dbSample.SampleLinks = marshalJSON(links)
	}

	dbSample.Metadata = marshalJSON(metadata)
	return dbSample
}
