package processor

import (
	"fmt"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// IdentifierHandler manages structured identifier and link storage
type IdentifierHandler struct {
	db Database
}

// NewIdentifierHandler creates a new identifier handler
func NewIdentifierHandler(db Database) *IdentifierHandler {
	return &IdentifierHandler{
		db: db,
	}
}

// StructuredIdentifier represents a normalized identifier
type StructuredIdentifier struct {
	RecordType      string // study, sample, experiment, run, analysis
	RecordAccession string
	IDType          string // primary, secondary, external, submitter, uuid
	IDNamespace     string
	IDValue         string
	IDLabel         string
}

// StructuredLink represents a normalized link
type StructuredLink struct {
	RecordType      string
	RecordAccession string
	LinkType        string // url, xref, entrez
	DB              string
	ID              string
	Label           string
	URL             string
	Query           string
}

// ExtractIdentifiers extracts all identifiers from a record
func (ih *IdentifierHandler) ExtractIdentifiers(identifiers *parser.Identifiers, recordType, recordAccession string) []StructuredIdentifier {
	if identifiers == nil {
		return nil
	}

	var structuredIDs []StructuredIdentifier

	// Extract primary ID
	if identifiers.PrimaryID != nil {
		structuredIDs = append(structuredIDs, StructuredIdentifier{
			RecordType:      recordType,
			RecordAccession: recordAccession,
			IDType:          "primary",
			IDValue:         identifiers.PrimaryID.Value,
			IDLabel:         identifiers.PrimaryID.Label,
		})
	}

	// Extract secondary IDs
	for _, id := range identifiers.SecondaryIDs {
		structuredIDs = append(structuredIDs, StructuredIdentifier{
			RecordType:      recordType,
			RecordAccession: recordAccession,
			IDType:          "secondary",
			IDValue:         id.Value,
			IDLabel:         id.Label,
		})
	}

	// Extract external IDs
	for _, id := range identifiers.ExternalIDs {
		structuredIDs = append(structuredIDs, StructuredIdentifier{
			RecordType:      recordType,
			RecordAccession: recordAccession,
			IDType:          "external",
			IDNamespace:     id.Namespace,
			IDValue:         id.Value,
			IDLabel:         id.Label,
		})
	}

	// Extract submitter IDs
	for _, id := range identifiers.SubmitterIDs {
		structuredIDs = append(structuredIDs, StructuredIdentifier{
			RecordType:      recordType,
			RecordAccession: recordAccession,
			IDType:          "submitter",
			IDNamespace:     id.Namespace,
			IDValue:         id.Value,
			IDLabel:         id.Label,
		})
	}

	// Extract UUIDs
	for _, id := range identifiers.UUIDs {
		structuredIDs = append(structuredIDs, StructuredIdentifier{
			RecordType:      recordType,
			RecordAccession: recordAccession,
			IDType:          "uuid",
			IDValue:         id.Value,
			IDLabel:         id.Label,
		})
	}

	return structuredIDs
}

// ExtractLinks extracts all links from a record
func (ih *IdentifierHandler) ExtractLinks(links []parser.Link, recordType, recordAccession string) []StructuredLink {
	var structuredLinks []StructuredLink

	for _, link := range links {
		sl := StructuredLink{
			RecordType:      recordType,
			RecordAccession: recordAccession,
		}

		// Extract URL link
		if link.URLLink != nil {
			sl.LinkType = "url"
			sl.URL = link.URLLink.URL
			sl.Label = link.URLLink.Label
		}

		// Extract XRef link
		if link.XRefLink != nil {
			sl.LinkType = "xref"
			sl.DB = link.XRefLink.DB
			sl.ID = link.XRefLink.ID
			sl.Label = link.XRefLink.Label
		}

		structuredLinks = append(structuredLinks, sl)
	}

	return structuredLinks
}

// StoreIdentifiers stores structured identifiers in the database
func (ih *IdentifierHandler) StoreIdentifiers(identifiers []StructuredIdentifier) error {
	for _, id := range identifiers {
		dbID := &database.Identifier{
			RecordType:      id.RecordType,
			RecordAccession: id.RecordAccession,
			IDType:          id.IDType,
			IDNamespace:     id.IDNamespace,
			IDValue:         id.IDValue,
			IDLabel:         id.IDLabel,
		}

		if err := ih.db.InsertIdentifier(dbID); err != nil {
			return fmt.Errorf("failed to insert identifier: %w", err)
		}
	}

	return nil
}

// StoreLinks stores structured links in the database
func (ih *IdentifierHandler) StoreLinks(links []StructuredLink) error {
	for _, link := range links {
		dbLink := &database.Link{
			RecordType:      link.RecordType,
			RecordAccession: link.RecordAccession,
			LinkType:        link.LinkType,
			DB:              link.DB,
			ID:              link.ID,
			Label:           link.Label,
			URL:             link.URL,
		}

		if err := ih.db.InsertLink(dbLink); err != nil {
			return fmt.Errorf("failed to insert link: %w", err)
		}
	}

	return nil
}

// GetIdentifiers retrieves all identifiers for a record
func (ih *IdentifierHandler) GetIdentifiers(recordType, recordAccession string) ([]StructuredIdentifier, error) {
	dbIDs, err := ih.db.GetIdentifiers(recordType, recordAccession)
	if err != nil {
		return nil, fmt.Errorf("failed to get identifiers: %w", err)
	}

	var identifiers []StructuredIdentifier
	for _, dbID := range dbIDs {
		identifiers = append(identifiers, StructuredIdentifier{
			RecordType:      dbID.RecordType,
			RecordAccession: dbID.RecordAccession,
			IDType:          dbID.IDType,
			IDNamespace:     dbID.IDNamespace,
			IDValue:         dbID.IDValue,
			IDLabel:         dbID.IDLabel,
		})
	}

	return identifiers, nil
}

// GetLinks retrieves all links for a record
func (ih *IdentifierHandler) GetLinks(recordType, recordAccession string) ([]StructuredLink, error) {
	dbLinks, err := ih.db.GetLinks(recordType, recordAccession)
	if err != nil {
		return nil, fmt.Errorf("failed to get links: %w", err)
	}

	var links []StructuredLink
	for _, dbLink := range dbLinks {
		links = append(links, StructuredLink{
			RecordType:      dbLink.RecordType,
			RecordAccession: dbLink.RecordAccession,
			LinkType:        dbLink.LinkType,
			DB:              dbLink.DB,
			ID:              dbLink.ID,
			Label:           dbLink.Label,
			URL:             dbLink.URL,
		})
	}

	return links, nil
}

// FindRecordsByIdentifier finds all records with a specific identifier value
func (ih *IdentifierHandler) FindRecordsByIdentifier(idValue string) ([]StructuredIdentifier, error) {
	dbIDs, err := ih.db.FindRecordsByIdentifier(idValue)
	if err != nil {
		return nil, fmt.Errorf("failed to find records by identifier: %w", err)
	}

	var identifiers []StructuredIdentifier
	for _, dbID := range dbIDs {
		identifiers = append(identifiers, StructuredIdentifier{
			RecordType:      dbID.RecordType,
			RecordAccession: dbID.RecordAccession,
			IDType:          dbID.IDType,
			IDNamespace:     dbID.IDNamespace,
			IDValue:         dbID.IDValue,
			IDLabel:         dbID.IDLabel,
		})
	}

	return identifiers, nil
}

// GetCrossReferences gets all cross-references for a record
func (ih *IdentifierHandler) GetCrossReferences(recordType, recordAccession string) (map[string][]string, error) {
	links, err := ih.GetLinks(recordType, recordAccession)
	if err != nil {
		return nil, err
	}

	crossRefs := make(map[string][]string)
	for _, link := range links {
		if link.LinkType == "xref" && link.DB != "" {
			crossRefs[link.DB] = append(crossRefs[link.DB], link.ID)
		}
	}

	return crossRefs, nil
}
