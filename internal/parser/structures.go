package parser

// ExperimentPackage represents an experiment from SRA XML
type ExperimentPackage struct {
	Accession string `xml:"accession,attr"`
	Title     string `xml:"EXPERIMENT>TITLE"`
	Study     struct {
		Accession string `xml:"accession,attr"`
	} `xml:"EXPERIMENT>STUDY_REF"`
	Design struct {
		SampleDescriptor struct {
			Accession string `xml:"accession,attr"`
		} `xml:"SAMPLE_DESCRIPTOR"`
	} `xml:"EXPERIMENT>DESIGN"`
	Platform struct {
		InstrumentModel string `xml:",innerxml"`
	} `xml:"EXPERIMENT>PLATFORM"`
	Library struct {
		Strategy  string `xml:"LIBRARY_STRATEGY"`
		Source    string `xml:"LIBRARY_SOURCE"`
		Selection string `xml:"LIBRARY_SELECTION"`
	} `xml:"EXPERIMENT>DESIGN>LIBRARY_DESCRIPTOR"`
}

// StudyPackage represents a study from SRA XML
type StudyPackage struct {
	Accession  string `xml:"accession,attr"`
	Descriptor struct {
		StudyTitle    string `xml:"STUDY_TITLE"`
		StudyAbstract string `xml:"STUDY_ABSTRACT"`
		StudyType     struct {
			ExistingStudyType string `xml:"existing_study_type,attr"`
		} `xml:"STUDY_TYPE"`
	} `xml:"STUDY>DESCRIPTOR"`
}

// SamplePackage represents a sample from SRA XML
type SamplePackage struct {
	Accession   string `xml:"accession,attr"`
	Description string `xml:"SAMPLE>DESCRIPTION"`
	SampleName  struct {
		ScientificName string `xml:"SCIENTIFIC_NAME"`
		TaxonID        int    `xml:"TAXON_ID"`
	} `xml:"SAMPLE>SAMPLE_NAME"`
	Attributes []struct {
		Tag   string `xml:"TAG"`
		Value string `xml:"VALUE"`
	} `xml:"SAMPLE>SAMPLE_ATTRIBUTES>SAMPLE_ATTRIBUTE"`
}
