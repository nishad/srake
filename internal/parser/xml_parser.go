package parser

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/nishad/srake/internal/models"
)

// XMLParser handles streaming XML parsing of SRA metadata
type XMLParser struct {
	decoder *xml.Decoder
}

// NewXMLParser creates a new XML parser
func NewXMLParser(reader io.Reader) *XMLParser {
	decoder := xml.NewDecoder(reader)
	decoder.Strict = false // Handle malformed XML
	decoder.AutoClose = xml.HTMLAutoClose

	return &XMLParser{decoder: decoder}
}

// Parse streams through XML and returns parsed entities
func (p *XMLParser) Parse() (<-chan interface{}, <-chan error) {
	results := make(chan interface{}, 100)
	errors := make(chan error, 1)

	go func() {
		defer close(results)
		defer close(errors)

		for {
			token, err := p.decoder.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				errors <- err
				return
			}

			switch t := token.(type) {
			case xml.StartElement:
				switch strings.ToUpper(t.Name.Local) {
				case "EXPERIMENT":
					if exp, err := p.parseExperiment(t); err == nil && exp != nil {
						results <- exp
					}
				case "RUN":
					if run, err := p.parseRun(t); err == nil && run != nil {
						results <- run
					}
				case "SAMPLE":
					if sample, err := p.parseSample(t); err == nil && sample != nil {
						results <- sample
					}
				case "STUDY":
					if study, err := p.parseStudy(t); err == nil && study != nil {
						results <- study
					}
				}
			}
		}
	}()

	return results, errors
}

func (p *XMLParser) parseExperiment(start xml.StartElement) (*models.Experiment, error) {
	exp := &models.Experiment{}

	// Extract attributes
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "accession":
			exp.Accession = attr.Value
		case "alias":
			exp.Alias = attr.Value
		}
	}

	// Parse nested elements
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "TITLE":
				if title, err := p.parseText(); err == nil {
					exp.Title = title
				}
			case "STUDY_REF":
				for _, attr := range t.Attr {
					if attr.Name.Local == "accession" {
						exp.StudyAccession = attr.Value
					}
				}
			case "DESIGN":
				p.parseDesign(exp)
			case "PLATFORM":
				p.parsePlatform(exp)
			case "SAMPLE_REF":
				for _, attr := range t.Attr {
					if attr.Name.Local == "accession" {
						exp.SampleAccession = attr.Value
					}
				}
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "EXPERIMENT" {
				return exp, nil
			}
		}
	}
}

func (p *XMLParser) parseDesign(exp *models.Experiment) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if strings.ToUpper(t.Name.Local) == "LIBRARY_DESCRIPTOR" {
				return p.parseLibraryDescriptor(exp)
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "DESIGN" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parseLibraryDescriptor(exp *models.Experiment) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "LIBRARY_STRATEGY":
				if text, err := p.parseText(); err == nil {
					exp.LibraryStrategy = text
				}
			case "LIBRARY_SOURCE":
				if text, err := p.parseText(); err == nil {
					exp.LibrarySource = text
				}
			case "LIBRARY_SELECTION":
				if text, err := p.parseText(); err == nil {
					exp.LibrarySelection = text
				}
			case "LIBRARY_LAYOUT":
				p.parseLibraryLayout(exp)
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "LIBRARY_DESCRIPTOR" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parseLibraryLayout(exp *models.Experiment) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "SINGLE":
				exp.LibraryLayout = "SINGLE"
			case "PAIRED":
				exp.LibraryLayout = "PAIRED"
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "LIBRARY_LAYOUT" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parsePlatform(exp *models.Experiment) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			platformType := strings.ToUpper(t.Name.Local)
			switch platformType {
			case "ILLUMINA", "PACBIO_SMRT", "ION_TORRENT", "OXFORD_NANOPORE", "CAPILLARY", "LS454", "HELICOS", "ABI_SOLID":
				exp.Platform = platformType
				p.parseInstrumentModel(exp)
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "PLATFORM" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parseInstrumentModel(exp *models.Experiment) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			if strings.ToUpper(t.Name.Local) == "INSTRUMENT_MODEL" {
				if text, err := p.parseText(); err == nil {
					exp.InstrumentModel = text
				}
			}
		case xml.EndElement:
			// Return when we exit the platform-specific element
			return nil
		}
	}
}

func (p *XMLParser) parseRun(start xml.StartElement) (*models.Run, error) {
	run := &models.Run{}

	// Extract attributes
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "accession":
			run.Accession = attr.Value
		case "alias":
			run.Alias = attr.Value
		case "spots":
			fmt.Sscanf(attr.Value, "%d", &run.Spots)
		case "bases":
			fmt.Sscanf(attr.Value, "%d", &run.Bases)
		case "spot_length":
			fmt.Sscanf(attr.Value, "%d", &run.SpotLength)
		case "published":
			if t, err := time.Parse(time.RFC3339, attr.Value); err == nil {
				run.PublishedDate = t
			}
		}
	}

	// Parse nested elements
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "EXPERIMENT_REF":
				for _, attr := range t.Attr {
					if attr.Name.Local == "accession" {
						run.ExperimentAccession = attr.Value
					}
				}
			case "STATISTICS":
				p.parseRunStatistics(run)
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "RUN" {
				return run, nil
			}
		}
	}
}

func (p *XMLParser) parseRunStatistics(run *models.Run) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			for _, attr := range t.Attr {
				switch attr.Name.Local {
				case "spots":
					fmt.Sscanf(attr.Value, "%d", &run.Spots)
				case "bases":
					fmt.Sscanf(attr.Value, "%d", &run.Bases)
				}
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "STATISTICS" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parseSample(start xml.StartElement) (*models.Sample, error) {
	sample := &models.Sample{}

	// Extract attributes
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "accession":
			sample.Accession = attr.Value
		case "alias":
			sample.Alias = attr.Value
		}
	}

	// Parse nested elements
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "DESCRIPTION":
				if text, err := p.parseText(); err == nil {
					sample.Description = text
				}
			case "SAMPLE_NAME":
				p.parseSampleName(sample)
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "SAMPLE" {
				return sample, nil
			}
		}
	}
}

func (p *XMLParser) parseSampleName(sample *models.Sample) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "SCIENTIFIC_NAME":
				if text, err := p.parseText(); err == nil {
					sample.ScientificName = text
				}
			case "TAXON_ID":
				if text, err := p.parseText(); err == nil {
					fmt.Sscanf(text, "%d", &sample.TaxonID)
				}
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "SAMPLE_NAME" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parseStudy(start xml.StartElement) (*models.Study, error) {
	study := &models.Study{}

	// Extract attributes
	for _, attr := range start.Attr {
		switch attr.Name.Local {
		case "accession":
			study.Accession = attr.Value
		}
	}

	// Parse nested elements
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "DESCRIPTOR":
				p.parseStudyDescriptor(study)
			case "SUBMISSION_REF":
				for _, attr := range t.Attr {
					if attr.Name.Local == "accession" {
						study.SubmissionAccession = attr.Value
					}
				}
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "STUDY" {
				return study, nil
			}
		}
	}
}

func (p *XMLParser) parseStudyDescriptor(study *models.Study) error {
	for {
		token, err := p.decoder.Token()
		if err != nil {
			return err
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch strings.ToUpper(t.Name.Local) {
			case "STUDY_TITLE":
				if text, err := p.parseText(); err == nil {
					study.Title = text
				}
			case "STUDY_TYPE":
				for _, attr := range t.Attr {
					if attr.Name.Local == "existing_study_type" {
						study.Type = attr.Value
					}
				}
			case "STUDY_ABSTRACT":
				if text, err := p.parseText(); err == nil {
					study.Abstract = text
				}
			}
		case xml.EndElement:
			if strings.ToUpper(t.Name.Local) == "DESCRIPTOR" {
				return nil
			}
		}
	}
}

func (p *XMLParser) parseText() (string, error) {
	token, err := p.decoder.Token()
	if err != nil {
		return "", err
	}

	if charData, ok := token.(xml.CharData); ok {
		return strings.TrimSpace(string(charData)), nil
	}
	return "", nil
}
