package parser

import (
	"encoding/xml"
	"time"
)

// =============== STUDY STRUCTURES ===============

// StudySet represents a collection of studies
type StudySet struct {
	XMLName xml.Name `xml:"STUDY_SET"`
	Studies []Study  `xml:"STUDY"`
}

// Study represents a complete SRA study record matching XSD schema
type Study struct {
	XMLName xml.Name `xml:"STUDY"`

	// Attributes from NameGroup
	Alias      string `xml:"alias,attr,omitempty"`
	CenterName string `xml:"center_name,attr,omitempty"`
	BrokerName string `xml:"broker_name,attr,omitempty"`
	Accession  string `xml:"accession,attr,omitempty"`

	// Elements
	Identifiers     *Identifiers     `xml:"IDENTIFIERS"`
	Descriptor      StudyDescriptor  `xml:"DESCRIPTOR"`
	StudyLinks      *StudyLinks      `xml:"STUDY_LINKS"`
	StudyAttributes *StudyAttributes `xml:"STUDY_ATTRIBUTES"`
}

// StudyDescriptor contains study metadata
type StudyDescriptor struct {
	StudyTitle        string          `xml:"STUDY_TITLE"`
	StudyType         *StudyType      `xml:"STUDY_TYPE"`
	StudyAbstract     string          `xml:"STUDY_ABSTRACT"`
	CenterProjectName string          `xml:"CENTER_PROJECT_NAME"`
	RelatedStudies    *RelatedStudies `xml:"RELATED_STUDIES"`
	StudyDescription  string          `xml:"STUDY_DESCRIPTION"`
}

// StudyType defines the type of study
type StudyType struct {
	ExistingStudyType string `xml:"existing_study_type,attr"`
	NewStudyType      string `xml:"new_study_type,attr,omitempty"`
}

// RelatedStudies contains links to related studies
type RelatedStudies struct {
	RelatedStudy []RelatedStudy `xml:"RELATED_STUDY"`
}

// RelatedStudy represents a single related study
type RelatedStudy struct {
	RelatedLink XRef `xml:"RELATED_LINK"`
	IsPrimary   bool `xml:"IS_PRIMARY"`
}

// StudyLinks contains external links for the study
type StudyLinks struct {
	Links []Link `xml:"STUDY_LINK"`
}

// StudyAttributes contains custom attributes
type StudyAttributes struct {
	Attributes []Attribute `xml:"STUDY_ATTRIBUTE"`
}

// =============== EXPERIMENT STRUCTURES ===============

// ExperimentSet represents a collection of experiments
type ExperimentSet struct {
	XMLName     xml.Name     `xml:"EXPERIMENT_SET"`
	Experiments []Experiment `xml:"EXPERIMENT"`
}

// Experiment represents a complete SRA experiment record
type Experiment struct {
	XMLName xml.Name `xml:"EXPERIMENT"`

	// Attributes from NameGroup
	Alias      string `xml:"alias,attr,omitempty"`
	CenterName string `xml:"center_name,attr,omitempty"`
	BrokerName string `xml:"broker_name,attr,omitempty"`
	Accession  string `xml:"accession,attr,omitempty"`

	// Elements
	Identifiers          *Identifiers          `xml:"IDENTIFIERS"`
	Title                string                `xml:"TITLE"`
	StudyRef             StudyRef              `xml:"STUDY_REF"`
	Design               Design                `xml:"DESIGN"`
	Platform             Platform              `xml:"PLATFORM"`
	ExperimentLinks      *ExperimentLinks      `xml:"EXPERIMENT_LINKS"`
	ExperimentAttributes *ExperimentAttributes `xml:"EXPERIMENT_ATTRIBUTES"`
}

// StudyRef references the parent study
type StudyRef struct {
	Identifiers *Identifiers `xml:"IDENTIFIERS"`
	RefName     string       `xml:"refname,attr,omitempty"`
	RefCenter   string       `xml:"refcenter,attr,omitempty"`
	Accession   string       `xml:"accession,attr,omitempty"`
}

// Design contains experiment design information
type Design struct {
	DesignDescription string            `xml:"DESIGN_DESCRIPTION"`
	SampleDescriptor  SampleDescriptor  `xml:"SAMPLE_DESCRIPTOR"`
	LibraryDescriptor LibraryDescriptor `xml:"LIBRARY_DESCRIPTOR"`
	SpotDescriptor    *SpotDescriptor   `xml:"SPOT_DESCRIPTOR"`
}

// SampleDescriptor references the sample
type SampleDescriptor struct {
	Identifiers *Identifiers `xml:"IDENTIFIERS"`
	Pool        *Pool        `xml:"POOL"`
	RefName     string       `xml:"refname,attr,omitempty"`
	RefCenter   string       `xml:"refcenter,attr,omitempty"`
	Accession   string       `xml:"accession,attr,omitempty"`
}

// Pool represents pooled samples
type Pool struct {
	DefaultMember *PoolMember  `xml:"DEFAULT_MEMBER"`
	Members       []PoolMember `xml:"MEMBER"`
}

// PoolMember represents a member of a sample pool
type PoolMember struct {
	RefName    string      `xml:"refname,attr,omitempty"`
	RefCenter  string      `xml:"refcenter,attr,omitempty"`
	Accession  string      `xml:"accession,attr,omitempty"`
	MemberName string      `xml:"member_name,attr,omitempty"`
	Proportion float32     `xml:"proportion,attr,omitempty"`
	ReadLabels []ReadLabel `xml:"READ_LABEL"`
}

// ReadLabel for pool members
type ReadLabel struct {
	ReadGroupTag string `xml:"read_group_tag,attr"`
	Value        string `xml:",chardata"`
}

// LibraryDescriptor contains library preparation details
type LibraryDescriptor struct {
	LibraryName                 string        `xml:"LIBRARY_NAME"`
	LibraryStrategy             string        `xml:"LIBRARY_STRATEGY"`
	LibrarySource               string        `xml:"LIBRARY_SOURCE"`
	LibrarySelection            string        `xml:"LIBRARY_SELECTION"`
	LibraryLayout               LibraryLayout `xml:"LIBRARY_LAYOUT"`
	TargetedLoci                *TargetedLoci `xml:"TARGETED_LOCI"`
	LibraryConstructionProtocol string        `xml:"LIBRARY_CONSTRUCTION_PROTOCOL"`
}

// LibraryLayout specifies single or paired reads
type LibraryLayout struct {
	Single *struct{}   `xml:"SINGLE"`
	Paired *PairedInfo `xml:"PAIRED"`
}

// PairedInfo contains paired-end library information
type PairedInfo struct {
	NominalLength int     `xml:"NOMINAL_LENGTH,attr,omitempty"`
	NominalSdev   float64 `xml:"NOMINAL_SDEV,attr,omitempty"`
}

// TargetedLoci for targeted sequencing
type TargetedLoci struct {
	Loci []Locus `xml:"LOCUS"`
}

// Locus represents a targeted locus
type Locus struct {
	LocusName   string `xml:"locus_name,attr"`
	Description string `xml:"description,attr,omitempty"`
	ProbeSet    *XRef  `xml:"PROBE_SET"`
}

// SpotDescriptor for decoding reads
type SpotDescriptor struct {
	SpotDecodeSpec *SpotDecodeSpec `xml:"SPOT_DECODE_SPEC"`
}

// SpotDecodeSpec contains spot decoding specification
type SpotDecodeSpec struct {
	SpotLength int        `xml:"SPOT_LENGTH"`
	ReadSpecs  []ReadSpec `xml:"READ_SPEC"`
}

// ReadSpec defines read specifications
type ReadSpec struct {
	ReadIndex  int    `xml:"READ_INDEX"`
	ReadClass  string `xml:"READ_CLASS"`
	ReadType   string `xml:"READ_TYPE"`
	BaseCoord  int    `xml:"BASE_COORD"`
	ReadLength int    `xml:"READ_LENGTH,omitempty"`
}

// Platform contains sequencing platform information
type Platform struct {
	Illumina         *PlatformDetails `xml:"ILLUMINA"`
	IonTorrent       *PlatformDetails `xml:"ION_TORRENT"`
	PacBio           *PlatformDetails `xml:"PACBIO_SMRT"`
	OxfordNanopore   *PlatformDetails `xml:"OXFORD_NANOPORE"`
	LS454            *PlatformDetails `xml:"LS454"`
	Solid            *PlatformDetails `xml:"ABI_SOLID"`
	Helicos          *PlatformDetails `xml:"HELICOS"`
	CompleteGenomics *PlatformDetails `xml:"COMPLETE_GENOMICS"`
	Capillary        *PlatformDetails `xml:"CAPILLARY"`
}

// PlatformDetails contains platform-specific information
type PlatformDetails struct {
	InstrumentModel string `xml:"INSTRUMENT_MODEL"`
	// Additional platform-specific fields can be added here
}

// ExperimentLinks contains external links
type ExperimentLinks struct {
	Links []Link `xml:"EXPERIMENT_LINK"`
}

// ExperimentAttributes contains custom attributes
type ExperimentAttributes struct {
	Attributes []Attribute `xml:"EXPERIMENT_ATTRIBUTE"`
}

// =============== SAMPLE STRUCTURES ===============

// SampleSet represents a collection of samples
type SampleSet struct {
	XMLName xml.Name `xml:"SAMPLE_SET"`
	Samples []Sample `xml:"SAMPLE"`
}

// Sample represents a complete SRA sample record
type Sample struct {
	XMLName xml.Name `xml:"SAMPLE"`

	// Attributes from NameGroup
	Alias      string `xml:"alias,attr,omitempty"`
	CenterName string `xml:"center_name,attr,omitempty"`
	BrokerName string `xml:"broker_name,attr,omitempty"`
	Accession  string `xml:"accession,attr,omitempty"`

	// Elements
	Identifiers      *Identifiers      `xml:"IDENTIFIERS"`
	Title            string            `xml:"TITLE"`
	SampleName       SampleName        `xml:"SAMPLE_NAME"`
	Description      string            `xml:"DESCRIPTION"`
	SampleLinks      *SampleLinks      `xml:"SAMPLE_LINKS"`
	SampleAttributes *SampleAttributes `xml:"SAMPLE_ATTRIBUTES"`
}

// SampleName contains taxonomic information
type SampleName struct {
	DisplayName    string `xml:"display_name,attr,omitempty"`
	TaxonID        int    `xml:"TAXON_ID"`
	ScientificName string `xml:"SCIENTIFIC_NAME"`
	CommonName     string `xml:"COMMON_NAME"`
}

// SampleLinks contains external links
type SampleLinks struct {
	Links []Link `xml:"SAMPLE_LINK"`
}

// SampleAttributes contains custom attributes
type SampleAttributes struct {
	Attributes []Attribute `xml:"SAMPLE_ATTRIBUTE"`
}

// =============== RUN STRUCTURES ===============

// RunSet represents a collection of runs
type RunSet struct {
	XMLName xml.Name `xml:"RUN_SET"`
	Runs    []Run    `xml:"RUN"`
}

// Run represents a complete SRA run record
type Run struct {
	XMLName xml.Name `xml:"RUN"`

	// Attributes from NameGroup
	Alias      string `xml:"alias,attr,omitempty"`
	CenterName string `xml:"center_name,attr,omitempty"`
	BrokerName string `xml:"broker_name,attr,omitempty"`
	Accession  string `xml:"accession,attr,omitempty"`
	RunCenter  string `xml:"run_center,attr,omitempty"`
	RunDate    string `xml:"run_date,attr,omitempty"`

	// Elements
	Identifiers   *Identifiers   `xml:"IDENTIFIERS"`
	Title         string         `xml:"TITLE"`
	ExperimentRef ExperimentRef  `xml:"EXPERIMENT_REF"`
	DataBlock     *DataBlock     `xml:"DATA_BLOCK"`
	RunLinks      *RunLinks      `xml:"RUN_LINKS"`
	RunAttributes *RunAttributes `xml:"RUN_ATTRIBUTES"`
	Statistics    *RunStatistics `xml:"Statistics"`
}

// ExperimentRef references the parent experiment
type ExperimentRef struct {
	Identifiers *Identifiers `xml:"IDENTIFIERS"`
	RefName     string       `xml:"refname,attr,omitempty"`
	RefCenter   string       `xml:"refcenter,attr,omitempty"`
	Accession   string       `xml:"accession,attr,omitempty"`
}

// DataBlock contains file information
type DataBlock struct {
	Files []RunFile `xml:"FILES>FILE"`
}

// RunFile represents a data file for the run
type RunFile struct {
	Filename            string `xml:"filename,attr"`
	FileType            string `xml:"filetype,attr"`
	ChecksumMethod      string `xml:"checksum_method,attr,omitempty"`
	Checksum            string `xml:"checksum,attr,omitempty"`
	UnencryptedChecksum string `xml:"unencrypted_checksum,attr,omitempty"`
}

// RunStatistics contains run statistics
type RunStatistics struct {
	TotalSpots int64  `xml:"total_spots,attr"`
	TotalBases int64  `xml:"total_bases,attr"`
	TotalSize  int64  `xml:"total_size,attr,omitempty"`
	LoadDone   bool   `xml:"load_done,attr,omitempty"`
	Published  string `xml:"published,attr,omitempty"`
}

// RunLinks contains external links
type RunLinks struct {
	Links []Link `xml:"RUN_LINK"`
}

// RunAttributes contains custom attributes
type RunAttributes struct {
	Attributes []Attribute `xml:"RUN_ATTRIBUTE"`
}

// =============== ANALYSIS STRUCTURES ===============

// AnalysisSet represents a collection of analyses
type AnalysisSet struct {
	XMLName  xml.Name   `xml:"ANALYSIS_SET"`
	Analyses []Analysis `xml:"ANALYSIS"`
}

// Analysis represents a complete SRA analysis record
type Analysis struct {
	XMLName xml.Name `xml:"ANALYSIS"`

	// Attributes from NameGroup
	Alias          string `xml:"alias,attr,omitempty"`
	CenterName     string `xml:"center_name,attr,omitempty"`
	BrokerName     string `xml:"broker_name,attr,omitempty"`
	Accession      string `xml:"accession,attr,omitempty"`
	AnalysisCenter string `xml:"analysis_center,attr,omitempty"`
	AnalysisDate   string `xml:"analysis_date,attr,omitempty"`

	// Elements
	Identifiers        *Identifiers        `xml:"IDENTIFIERS"`
	Title              string              `xml:"TITLE"`
	StudyRef           StudyRef            `xml:"STUDY_REF"`
	Description        string              `xml:"DESCRIPTION"`
	AnalysisType       AnalysisType        `xml:"ANALYSIS_TYPE"`
	Targets            *AnalysisTargets    `xml:"TARGETS"`
	DataBlocks         []AnalysisDataBlock `xml:"DATA_BLOCK"`
	AnalysisLinks      *AnalysisLinks      `xml:"ANALYSIS_LINKS"`
	AnalysisAttributes *AnalysisAttributes `xml:"ANALYSIS_ATTRIBUTES"`
}

// AnalysisType represents the type of analysis
type AnalysisType struct {
	DeNovoAssembly       *DeNovoAssembly       `xml:"DE_NOVO_ASSEMBLY"`
	ReferenceAlignment   *ReferenceAlignment   `xml:"REFERENCE_ALIGNMENT"`
	SequenceAnnotation   *SequenceAnnotation   `xml:"SEQUENCE_ANNOTATION"`
	AbundanceMeasurement *AbundanceMeasurement `xml:"ABUNDANCE_MEASUREMENT"`
}

// DeNovoAssembly represents de novo assembly analysis
type DeNovoAssembly struct {
	Processing ProcessingType `xml:"PROCESSING"`
}

// ReferenceAlignment represents reference alignment analysis
type ReferenceAlignment struct {
	Assembly   Assembly            `xml:"ASSEMBLY"`
	RunLabels  *RunLabels          `xml:"RUN_LABELS"`
	SeqLabels  *SeqLabels          `xml:"SEQ_LABELS"`
	Processing AlignmentProcessing `xml:"PROCESSING"`
}

// SequenceAnnotation represents sequence annotation analysis
type SequenceAnnotation struct {
	Processing ProcessingType `xml:"PROCESSING"`
}

// AbundanceMeasurement represents abundance measurement analysis
type AbundanceMeasurement struct {
	Processing ProcessingType `xml:"PROCESSING"`
}

// Assembly represents assembly reference
type Assembly struct {
	Standard *StandardAssembly `xml:"STANDARD"`
	Custom   *CustomAssembly   `xml:"CUSTOM"`
}

// StandardAssembly represents standard reference assembly
type StandardAssembly struct {
	ShortName string `xml:"short_name,attr,omitempty"`
	Names     []XRef `xml:"NAME"`
}

// CustomAssembly represents custom reference assembly
type CustomAssembly struct {
	Description     string `xml:"DESCRIPTION"`
	ReferenceSource []Link `xml:"REFERENCE_SOURCE"`
}

// RunLabels maps run labels to archive runs
type RunLabels struct {
	Runs []RunLabel `xml:"RUN"`
}

// RunLabel represents a run label mapping
type RunLabel struct {
	RefName        string `xml:"refname,attr,omitempty"`
	RefCenter      string `xml:"refcenter,attr,omitempty"`
	Accession      string `xml:"accession,attr,omitempty"`
	DataBlockName  string `xml:"data_block_name,attr,omitempty"`
	ReadGroupLabel string `xml:"read_group_label,attr,omitempty"`
}

// SeqLabels maps sequence labels to reference sequences
type SeqLabels struct {
	Sequences []SeqLabel `xml:"SEQUENCE"`
}

// SeqLabel represents a sequence label mapping
type SeqLabel struct {
	Accession     string `xml:"accession,attr"`
	GI            int    `xml:"gi,attr,omitempty"`
	DataBlockName string `xml:"data_block_name,attr,omitempty"`
	SeqLabel      string `xml:"seq_label,attr,omitempty"`
}

// ProcessingType represents processing information
type ProcessingType struct {
	Pipeline PipelineType `xml:"PIPELINE"`
}

// AlignmentProcessing represents alignment-specific processing
type AlignmentProcessing struct {
	Pipeline   PipelineType             `xml:"PIPELINE"`
	Directives *AlignmentDirectivesType `xml:"DIRECTIVES"`
}

// PipelineType represents pipeline information
type PipelineType struct {
	Programs []ProgramType `xml:"PIPE_SECTION>PROGRAM"`
}

// ProgramType represents a program in the pipeline
type ProgramType struct {
	Name    string `xml:"name,attr"`
	Version string `xml:"version,attr,omitempty"`
}

// AlignmentDirectivesType represents alignment directives
type AlignmentDirectivesType struct {
	// Add specific directives as needed
	Directives map[string]string
}

// AnalysisTargets represents analysis targets
type AnalysisTargets struct {
	Targets []AnalysisTarget `xml:"TARGET"`
}

// AnalysisTarget represents a single analysis target
type AnalysisTarget struct {
	SraObjectType string       `xml:"sra_object_type,attr,omitempty"`
	RefName       string       `xml:"refname,attr,omitempty"`
	RefCenter     string       `xml:"refcenter,attr,omitempty"`
	Accession     string       `xml:"accession,attr,omitempty"`
	Identifiers   *Identifiers `xml:"IDENTIFIERS"`
}

// AnalysisDataBlock represents a data block in analysis
type AnalysisDataBlock struct {
	Name   string         `xml:"name,attr,omitempty"`
	Serial int            `xml:"serial,attr,omitempty"`
	Member string         `xml:"member,attr,omitempty"`
	Files  []AnalysisFile `xml:"FILES>FILE"`
}

// AnalysisFile represents a file in analysis data block
type AnalysisFile struct {
	Filename       string `xml:"filename,attr"`
	FileType       string `xml:"filetype,attr"`
	ChecksumMethod string `xml:"checksum_method,attr"`
	Checksum       string `xml:"checksum,attr"`
}

// AnalysisLinks contains external links
type AnalysisLinks struct {
	Links []Link `xml:"ANALYSIS_LINK"`
}

// AnalysisAttributes contains custom attributes
type AnalysisAttributes struct {
	Attributes []Attribute `xml:"ANALYSIS_ATTRIBUTE"`
}

// =============== SUBMISSION STRUCTURES ===============

// SubmissionSet represents a collection of submissions
type SubmissionSet struct {
	XMLName     xml.Name     `xml:"SUBMISSION_SET"`
	Submissions []Submission `xml:"SUBMISSION"`
}

// Submission represents a complete SRA submission record
type Submission struct {
	XMLName xml.Name `xml:"SUBMISSION"`

	// Attributes from NameGroup
	Alias             string `xml:"alias,attr,omitempty"`
	CenterName        string `xml:"center_name,attr,omitempty"`
	BrokerName        string `xml:"broker_name,attr,omitempty"`
	Accession         string `xml:"accession,attr,omitempty"`
	SubmissionDate    string `xml:"submission_date,attr,omitempty"`
	SubmissionComment string `xml:"submission_comment,attr,omitempty"`
	LabName           string `xml:"lab_name,attr,omitempty"`

	// Elements
	Identifiers          *Identifiers          `xml:"IDENTIFIERS"`
	Title                string                `xml:"TITLE"`
	Contacts             *SubmissionContacts   `xml:"CONTACTS"`
	Actions              *SubmissionActions    `xml:"ACTIONS"`
	SubmissionLinks      *SubmissionLinks      `xml:"SUBMISSION_LINKS"`
	SubmissionAttributes *SubmissionAttributes `xml:"SUBMISSION_ATTRIBUTES"`
}

// SubmissionContacts contains submission contacts
type SubmissionContacts struct {
	Contacts []Contact `xml:"CONTACT"`
}

// Contact represents a submission contact
type Contact struct {
	Name           string `xml:"name,attr,omitempty"`
	InformOnStatus string `xml:"inform_on_status,attr,omitempty"`
	InformOnError  string `xml:"inform_on_error,attr,omitempty"`
}

// SubmissionActions contains submission actions
type SubmissionActions struct {
	Actions []Action `xml:"ACTION"`
}

// Action represents a submission action
type Action struct {
	Add      *AddAction      `xml:"ADD"`
	Modify   *ModifyAction   `xml:"MODIFY"`
	Suppress *SuppressAction `xml:"SUPPRESS"`
	Hold     *HoldAction     `xml:"HOLD"`
	Release  *ReleaseAction  `xml:"RELEASE"`
	Protect  *ProtectAction  `xml:"PROTECT"`
	Validate *ValidateAction `xml:"VALIDATE"`
}

// AddAction represents an ADD action
type AddAction struct {
	Source string `xml:"source,attr"`
	Schema string `xml:"schema,attr,omitempty"`
}

// ModifyAction represents a MODIFY action
type ModifyAction struct {
	Source string `xml:"source,attr"`
	Schema string `xml:"schema,attr,omitempty"`
}

// SuppressAction represents a SUPPRESS action
type SuppressAction struct {
	Target string `xml:"target,attr"`
}

// HoldAction represents a HOLD action
type HoldAction struct {
	Target        string `xml:"target,attr,omitempty"`
	HoldUntilDate string `xml:"HoldUntilDate,attr,omitempty"`
}

// ReleaseAction represents a RELEASE action
type ReleaseAction struct {
	Target string `xml:"target,attr,omitempty"`
}

// ProtectAction represents a PROTECT action
type ProtectAction struct {
	// Empty by design
}

// ValidateAction represents a VALIDATE action
type ValidateAction struct {
	// Empty by design
}

// SubmissionLinks contains external links
type SubmissionLinks struct {
	Links []Link `xml:"SUBMISSION_LINK"`
}

// SubmissionAttributes contains custom attributes
type SubmissionAttributes struct {
	Attributes []Attribute `xml:"SUBMISSION_ATTRIBUTE"`
}

// =============== COMMON STRUCTURES ===============

// Identifiers contains record identifiers
type Identifiers struct {
	PrimaryID    *Identifier   `xml:"PRIMARY_ID"`
	SecondaryIDs []Identifier  `xml:"SECONDARY_ID"`
	ExternalIDs  []QualifiedID `xml:"EXTERNAL_ID"`
	SubmitterIDs []QualifiedID `xml:"SUBMITTER_ID"`
	UUIDs        []Identifier  `xml:"UUID"`
}

// Identifier represents a simple identifier
type Identifier struct {
	Label string `xml:"label,attr,omitempty"`
	Value string `xml:",chardata"`
}

// QualifiedID represents an identifier with namespace
type QualifiedID struct {
	Namespace string `xml:"namespace,attr"`
	Label     string `xml:"label,attr,omitempty"`
	Value     string `xml:",chardata"`
}

// Attribute represents a tag-value pair with optional units
type Attribute struct {
	Tag   string `xml:"TAG"`
	Value string `xml:"VALUE"`
	Units string `xml:"UNITS,omitempty"`
}

// Link represents an external link
type Link struct {
	URLLink  *URLLink `xml:"URL_LINK"`
	XRefLink *XRef    `xml:"XREF_LINK"`
}

// URLLink represents a URL link
type URLLink struct {
	Label string `xml:"LABEL"`
	URL   string `xml:"URL"`
}

// XRef represents a cross-reference
type XRef struct {
	DB    string `xml:"DB"`
	ID    string `xml:"ID"`
	Label string `xml:"LABEL,omitempty"`
}

// =============== HELPER FUNCTIONS ===============

// ParseTime converts SRA date strings to time.Time
func ParseTime(dateStr string) time.Time {
	// Try different date formats used in SRA
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, dateStr); err == nil {
			return t
		}
	}

	return time.Time{}
}

// GetPlatformName extracts the platform name from the Platform struct
func (p *Platform) GetPlatformName() string {
	if p.Illumina != nil {
		return "ILLUMINA"
	} else if p.IonTorrent != nil {
		return "ION_TORRENT"
	} else if p.PacBio != nil {
		return "PACBIO_SMRT"
	} else if p.OxfordNanopore != nil {
		return "OXFORD_NANOPORE"
	} else if p.LS454 != nil {
		return "LS454"
	} else if p.Solid != nil {
		return "ABI_SOLID"
	} else if p.Helicos != nil {
		return "HELICOS"
	} else if p.CompleteGenomics != nil {
		return "COMPLETE_GENOMICS"
	} else if p.Capillary != nil {
		return "CAPILLARY"
	}
	return ""
}

// GetInstrumentModel extracts the instrument model from the Platform struct
func (p *Platform) GetInstrumentModel() string {
	if p.Illumina != nil {
		return p.Illumina.InstrumentModel
	} else if p.IonTorrent != nil {
		return p.IonTorrent.InstrumentModel
	} else if p.PacBio != nil {
		return p.PacBio.InstrumentModel
	} else if p.OxfordNanopore != nil {
		return p.OxfordNanopore.InstrumentModel
	} else if p.LS454 != nil {
		return p.LS454.InstrumentModel
	} else if p.Solid != nil {
		return p.Solid.InstrumentModel
	} else if p.Helicos != nil {
		return p.Helicos.InstrumentModel
	} else if p.CompleteGenomics != nil {
		return p.CompleteGenomics.InstrumentModel
	} else if p.Capillary != nil {
		return p.Capillary.InstrumentModel
	}
	return ""
}

// IsPaired checks if the library layout is paired-end
func (l *LibraryLayout) IsPaired() bool {
	return l.Paired != nil
}

// GetAttributeValue finds an attribute by tag name
func GetAttributeValue(attributes []Attribute, tag string) string {
	for _, attr := range attributes {
		if attr.Tag == tag {
			return attr.Value
		}
	}
	return ""
}

// GetAnalysisTypeName returns the name of the analysis type
func (a *AnalysisType) GetAnalysisTypeName() string {
	if a.DeNovoAssembly != nil {
		return "DE_NOVO_ASSEMBLY"
	} else if a.ReferenceAlignment != nil {
		return "REFERENCE_ALIGNMENT"
	} else if a.SequenceAnnotation != nil {
		return "SEQUENCE_ANNOTATION"
	} else if a.AbundanceMeasurement != nil {
		return "ABUNDANCE_MEASUREMENT"
	}
	return ""
}

// GetActionType returns the type of action
func (a *Action) GetActionType() string {
	if a.Add != nil {
		return "ADD"
	} else if a.Modify != nil {
		return "MODIFY"
	} else if a.Suppress != nil {
		return "SUPPRESS"
	} else if a.Hold != nil {
		return "HOLD"
	} else if a.Release != nil {
		return "RELEASE"
	} else if a.Protect != nil {
		return "PROTECT"
	} else if a.Validate != nil {
		return "VALIDATE"
	}
	return ""
}

// HasMultipleDataBlocks checks if analysis has multiple data blocks
func (a *Analysis) HasMultipleDataBlocks() bool {
	return len(a.DataBlocks) > 1
}

// GetTargetAccessions returns all target accessions
func (a *Analysis) GetTargetAccessions() []string {
	var accessions []string
	if a.Targets != nil {
		for _, target := range a.Targets.Targets {
			if target.Accession != "" {
				accessions = append(accessions, target.Accession)
			}
		}
	}
	return accessions
}
