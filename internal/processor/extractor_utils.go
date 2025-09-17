package processor

import (
	"encoding/json"
	"strings"

	"github.com/nishad/srake/internal/parser"
)

// extractAttributes converts attributes to a map
func (ce *ComprehensiveExtractor) extractAttributes(attrs []parser.Attribute) []map[string]string {
	var attributes []map[string]string
	for _, attr := range attrs {
		attrMap := map[string]string{
			"tag":   attr.Tag,
			"value": attr.Value,
		}
		if attr.Units != "" {
			attrMap["units"] = attr.Units
		}
		attributes = append(attributes, attrMap)
	}
	return attributes
}

// extractLinks converts links to a map
func (ce *ComprehensiveExtractor) extractLinks(links []parser.Link) []map[string]string {
	var result []map[string]string
	for _, link := range links {
		linkMap := make(map[string]string)
		if link.URLLink != nil {
			linkMap["type"] = "URL"
			linkMap["label"] = link.URLLink.Label
			linkMap["url"] = link.URLLink.URL
		} else if link.XRefLink != nil {
			linkMap["type"] = "XREF"
			linkMap["db"] = link.XRefLink.DB
			linkMap["id"] = link.XRefLink.ID
			if link.XRefLink.Label != "" {
				linkMap["label"] = link.XRefLink.Label
			}
		}
		if len(linkMap) > 0 {
			result = append(result, linkMap)
		}
	}
	return result
}

// extractLink converts a single link to a map
func (ce *ComprehensiveExtractor) extractLink(link parser.Link) map[string]string {
	linkMap := make(map[string]string)
	if link.URLLink != nil {
		linkMap["type"] = "URL"
		linkMap["label"] = link.URLLink.Label
		linkMap["url"] = link.URLLink.URL
	} else if link.XRefLink != nil {
		linkMap["type"] = "XREF"
		linkMap["db"] = link.XRefLink.DB
		linkMap["id"] = link.XRefLink.ID
		if link.XRefLink.Label != "" {
			linkMap["label"] = link.XRefLink.Label
		}
	}
	return linkMap
}

// extractIdentifiers extracts identifiers to a structured format
func (ce *ComprehensiveExtractor) extractIdentifiers(ids *parser.Identifiers) map[string]interface{} {
	if ids == nil {
		return nil
	}

	result := make(map[string]interface{})

	if ids.PrimaryID != nil {
		result["primary_id"] = ids.PrimaryID.Value
	}

	var secondaryIDs []string
	for _, id := range ids.SecondaryIDs {
		secondaryIDs = append(secondaryIDs, id.Value)
	}
	if len(secondaryIDs) > 0 {
		result["secondary_ids"] = secondaryIDs
	}

	var externalIDs []map[string]string
	for _, id := range ids.ExternalIDs {
		extID := map[string]string{
			"namespace": id.Namespace,
			"value":     id.Value,
		}
		externalIDs = append(externalIDs, extID)
	}
	if len(externalIDs) > 0 {
		result["external_ids"] = externalIDs
	}

	var submitterIDs []map[string]string
	for _, id := range ids.SubmitterIDs {
		subID := map[string]string{
			"namespace": id.Namespace,
			"value":     id.Value,
		}
		submitterIDs = append(submitterIDs, subID)
	}
	if len(submitterIDs) > 0 {
		result["submitter_ids"] = submitterIDs
	}

	return result
}

// normalizeOrganism normalizes organism names to standard format
func (ce *ComprehensiveExtractor) normalizeOrganism(organism string) string {
	if !ce.options.NormalizeOrganism {
		return organism
	}

	// Common normalizations
	normalizations := map[string]string{
		"homo sapiens":             "Homo sapiens",
		"human":                    "Homo sapiens",
		"mouse":                    "Mus musculus",
		"mus musculus":             "Mus musculus",
		"rat":                      "Rattus norvegicus",
		"rattus norvegicus":        "Rattus norvegicus",
		"e. coli":                  "Escherichia coli",
		"e.coli":                   "Escherichia coli",
		"escherichia coli":         "Escherichia coli",
		"yeast":                    "Saccharomyces cerevisiae",
		"saccharomyces cerevisiae": "Saccharomyces cerevisiae",
		"fruit fly":                "Drosophila melanogaster",
		"drosophila":               "Drosophila melanogaster",
		"drosophila melanogaster":  "Drosophila melanogaster",
		"zebrafish":                "Danio rerio",
		"danio rerio":              "Danio rerio",
		"c. elegans":               "Caenorhabditis elegans",
		"c.elegans":                "Caenorhabditis elegans",
		"caenorhabditis elegans":   "Caenorhabditis elegans",
	}

	lower := strings.ToLower(strings.TrimSpace(organism))
	if normalized, ok := normalizations[lower]; ok {
		return normalized
	}

	// If not found, return with proper capitalization
	parts := strings.Fields(organism)
	if len(parts) > 0 {
		// Capitalize genus
		parts[0] = strings.Title(strings.ToLower(parts[0]))
		// Species in lowercase
		for i := 1; i < len(parts); i++ {
			parts[i] = strings.ToLower(parts[i])
		}
		return strings.Join(parts, " ")
	}

	return organism
}

// marshalJSON safely marshals data to JSON string
func marshalJSON(data interface{}) string {
	if data == nil {
		return "{}"
	}
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "{}"
	}
	return string(jsonData)
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// extractPlatformName extracts the platform name from a Platform struct
func extractPlatformName(platform parser.Platform) string {
	if platform.Illumina != nil {
		return "ILLUMINA"
	} else if platform.IonTorrent != nil {
		return "ION_TORRENT"
	} else if platform.PacBio != nil {
		return "PACBIO_SMRT"
	} else if platform.OxfordNanopore != nil {
		return "OXFORD_NANOPORE"
	} else if platform.LS454 != nil {
		return "LS454"
	} else if platform.Solid != nil {
		return "ABI_SOLID"
	} else if platform.Helicos != nil {
		return "HELICOS"
	} else if platform.CompleteGenomics != nil {
		return "COMPLETE_GENOMICS"
	} else if platform.Capillary != nil {
		return "CAPILLARY"
	}
	// Add more platforms as needed
	return "UNKNOWN"
}

// extractInstrumentModel extracts the instrument model from a Platform struct
func extractInstrumentModel(platform parser.Platform) string {
	if platform.Illumina != nil {
		return platform.Illumina.InstrumentModel
	} else if platform.IonTorrent != nil {
		return platform.IonTorrent.InstrumentModel
	} else if platform.PacBio != nil {
		return platform.PacBio.InstrumentModel
	} else if platform.OxfordNanopore != nil {
		return platform.OxfordNanopore.InstrumentModel
	} else if platform.LS454 != nil {
		return platform.LS454.InstrumentModel
	} else if platform.Solid != nil {
		return platform.Solid.InstrumentModel
	} else if platform.Helicos != nil {
		return platform.Helicos.InstrumentModel
	} else if platform.CompleteGenomics != nil {
		return platform.CompleteGenomics.InstrumentModel
	} else if platform.Capillary != nil {
		return platform.Capillary.InstrumentModel
	}
	return ""
}

// extractLibraryLayout extracts the library layout type
func extractLibraryLayout(layout parser.LibraryLayout) string {
	if layout.Single != nil {
		return "SINGLE"
	} else if layout.Paired != nil {
		return "PAIRED"
	}
	return ""
}

// extractSpotDescriptor extracts spot descriptor information
func extractSpotDescriptor(spotDesc *parser.SpotDescriptor) map[string]interface{} {
	if spotDesc == nil || spotDesc.SpotDecodeSpec == nil {
		return nil
	}

	result := map[string]interface{}{
		"spot_length": spotDesc.SpotDecodeSpec.SpotLength,
	}

	var readSpecs []map[string]interface{}
	for _, spec := range spotDesc.SpotDecodeSpec.ReadSpecs {
		readSpec := map[string]interface{}{
			"read_index": spec.ReadIndex,
			"read_class": spec.ReadClass,
			"read_type":  spec.ReadType,
			"base_coord": spec.BaseCoord,
		}
		if spec.ReadLength > 0 {
			readSpec["read_length"] = spec.ReadLength
		}
		readSpecs = append(readSpecs, readSpec)
	}
	if len(readSpecs) > 0 {
		result["read_specs"] = readSpecs
	}

	return result
}
