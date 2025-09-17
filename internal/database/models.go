package database

import (
	"time"
)

// Study represents a comprehensive SRA study record
type Study struct {
	// Primary key
	StudyAccession string `json:"study_accession"`

	// NameGroup attributes
	Alias      string `json:"alias"`
	CenterName string `json:"center_name"`
	BrokerName string `json:"broker_name"`

	// Core fields
	StudyTitle        string `json:"study_title"`
	StudyType         string `json:"study_type"`
	StudyAbstract     string `json:"study_abstract"`
	StudyDescription  string `json:"study_description"`
	CenterProjectName string `json:"center_project_name"`

	// Dates
	SubmissionDate *time.Time `json:"submission_date"`
	FirstPublic    *time.Time `json:"first_public"`
	LastUpdate     *time.Time `json:"last_update"`

	// Identifiers (JSON)
	PrimaryID    string `json:"primary_id"`
	SecondaryIDs string `json:"secondary_ids"` // JSON array
	ExternalIDs  string `json:"external_ids"`  // JSON array
	SubmitterIDs string `json:"submitter_ids"` // JSON array

	// Links and attributes (JSON)
	StudyLinks      string `json:"study_links"`      // JSON array
	StudyAttributes string `json:"study_attributes"` // JSON array
	RelatedStudies  string `json:"related_studies"`  // JSON array

	// Extracted organism
	Organism string `json:"organism"`

	// Full metadata
	Metadata string `json:"metadata"` // JSON
}

// Experiment represents a comprehensive SRA experiment record
type Experiment struct {
	// Primary key
	ExperimentAccession string `json:"experiment_accession"`

	// NameGroup attributes
	Alias      string `json:"alias"`
	CenterName string `json:"center_name"`
	BrokerName string `json:"broker_name"`

	// References
	StudyAccession  string `json:"study_accession"`
	SampleAccession string `json:"sample_accession"`

	// Core fields
	Title             string `json:"title"`
	DesignDescription string `json:"design_description"`

	// Library information
	LibraryName                 string `json:"library_name"`
	LibraryStrategy             string `json:"library_strategy"`
	LibrarySource               string `json:"library_source"`
	LibrarySelection            string `json:"library_selection"`
	LibraryLayout               string `json:"library_layout"` // 'SINGLE' or 'PAIRED'
	LibraryConstructionProtocol string `json:"library_construction_protocol"`

	// Paired-end specific
	NominalLength int     `json:"nominal_length"`
	NominalSdev   float64 `json:"nominal_sdev"`

	// Platform information
	Platform        string `json:"platform"`
	InstrumentModel string `json:"instrument_model"`

	// Targeted sequencing
	TargetedLoci string `json:"targeted_loci"` // JSON array

	// Pooling information
	PoolMemberCount int    `json:"pool_member_count"`
	PoolInfo        string `json:"pool_info"` // JSON object

	// Links and attributes
	ExperimentLinks      string `json:"experiment_links"`      // JSON array
	ExperimentAttributes string `json:"experiment_attributes"` // JSON array

	// Spot descriptor
	SpotLength     int    `json:"spot_length"`
	SpotDecodeSpec string `json:"spot_decode_spec"` // JSON object

	// Full metadata
	Metadata string `json:"metadata"` // JSON
}

// Sample represents a comprehensive SRA sample record
type Sample struct {
	// Primary key
	SampleAccession string `json:"sample_accession"`

	// NameGroup attributes
	Alias      string `json:"alias"`
	CenterName string `json:"center_name"`
	BrokerName string `json:"broker_name"`

	// Core fields
	Title       string `json:"title"`
	Description string `json:"description"`

	// Taxonomy
	TaxonID        int    `json:"taxon_id"`
	ScientificName string `json:"scientific_name"`
	CommonName     string `json:"common_name"`
	Organism       string `json:"organism"`

	// Sample source information
	Tissue    string `json:"tissue"`
	CellType  string `json:"cell_type"`
	CellLine  string `json:"cell_line"`
	Strain    string `json:"strain"`
	Sex       string `json:"sex"`
	Age       string `json:"age"`
	Disease   string `json:"disease"`
	Treatment string `json:"treatment"`

	// Geographic/environmental
	GeoLocName     string `json:"geo_loc_name"`
	LatLon         string `json:"lat_lon"`
	CollectionDate string `json:"collection_date"`
	EnvBiome       string `json:"env_biome"`
	EnvFeature     string `json:"env_feature"`
	EnvMaterial    string `json:"env_material"`

	// Links and attributes
	SampleLinks      string `json:"sample_links"`      // JSON array
	SampleAttributes string `json:"sample_attributes"` // JSON array

	// BioSample/BioProject references
	BiosampleAccession  string `json:"biosample_accession"`
	BioprojectAccession string `json:"bioproject_accession"`

	// Full metadata
	Metadata string `json:"metadata"` // JSON
}

// Run represents a comprehensive SRA run record
type Run struct {
	// Primary key
	RunAccession string `json:"run_accession"`

	// NameGroup attributes
	Alias      string `json:"alias"`
	CenterName string `json:"center_name"`
	BrokerName string `json:"broker_name"`
	RunCenter  string `json:"run_center"`

	// References
	ExperimentAccession string `json:"experiment_accession"`

	// Core fields
	Title   string     `json:"title"`
	RunDate *time.Time `json:"run_date"`

	// Statistics
	TotalSpots int64  `json:"total_spots"`
	TotalBases int64  `json:"total_bases"`
	TotalSize  int64  `json:"total_size"`
	LoadDone   bool   `json:"load_done"`
	Published  string `json:"published"`

	// File information
	DataFiles string `json:"data_files"` // JSON array

	// Links and attributes
	RunLinks      string `json:"run_links"`      // JSON array
	RunAttributes string `json:"run_attributes"` // JSON array

	// Quality metrics
	QualityScoreMean float64 `json:"quality_score_mean"`
	QualityScoreStd  float64 `json:"quality_score_std"`
	ReadCountR1      int64   `json:"read_count_r1"`
	ReadCountR2      int64   `json:"read_count_r2"`

	// Full metadata
	Metadata string `json:"metadata"` // JSON
}

// Submission represents a submission record with enhanced fields
type Submission struct {
	SubmissionAccession  string     `json:"submission_accession"`
	Alias                string     `json:"alias"`
	CenterName           string     `json:"center_name"`
	BrokerName           string     `json:"broker_name"`
	LabName              string     `json:"lab_name"`
	Title                string     `json:"title"`
	SubmissionDate       *time.Time `json:"submission_date"`
	SubmissionComment    string     `json:"submission_comment"`
	Contacts             string     `json:"contacts"`              // JSON array of contacts
	Actions              string     `json:"actions"`               // JSON array of actions
	SubmissionLinks      string     `json:"submission_links"`      // JSON array
	SubmissionAttributes string     `json:"submission_attributes"` // JSON array
	Metadata             string     `json:"metadata"`              // JSON
}

// Analysis represents an analysis record with comprehensive fields
type Analysis struct {
	AnalysisAccession string     `json:"analysis_accession"`
	Alias             string     `json:"alias"`
	CenterName        string     `json:"center_name"`
	BrokerName        string     `json:"broker_name"`
	AnalysisCenter    string     `json:"analysis_center"`
	AnalysisDate      *time.Time `json:"analysis_date"`
	StudyAccession    string     `json:"study_accession"`
	Title             string     `json:"title"`
	Description       string     `json:"description"`
	AnalysisType      string     `json:"analysis_type"`

	// Analysis-specific fields
	Targets     string `json:"targets"`      // JSON array of target SRA objects
	DataBlocks  string `json:"data_blocks"`  // JSON array of data blocks
	AssemblyRef string `json:"assembly_ref"` // JSON object for assembly reference
	RunLabels   string `json:"run_labels"`   // JSON array for run label mappings
	SeqLabels   string `json:"seq_labels"`   // JSON array for sequence label mappings
	Processing  string `json:"processing"`   // JSON object for pipeline info

	// Links and attributes
	AnalysisLinks      string `json:"analysis_links"`      // JSON array
	AnalysisAttributes string `json:"analysis_attributes"` // JSON array
	Metadata           string `json:"metadata"`            // JSON
}

// SamplePool represents a pool/multiplex relationship
type SamplePool struct {
	PoolID       int     `json:"pool_id"`
	ParentSample string  `json:"parent_sample"`
	MemberSample string  `json:"member_sample"`
	MemberName   string  `json:"member_name"`
	Proportion   float64 `json:"proportion"`
	ReadLabel    string  `json:"read_label"`
}

// Identifier represents a structured identifier
type Identifier struct {
	RecordType      string `json:"record_type"`
	RecordAccession string `json:"record_accession"`
	IDType          string `json:"id_type"`
	IDNamespace     string `json:"id_namespace"`
	IDValue         string `json:"id_value"`
	IDLabel         string `json:"id_label"`
}

// Link represents a structured link
type Link struct {
	RecordType      string `json:"record_type"`
	RecordAccession string `json:"record_accession"`
	LinkType        string `json:"link_type"`
	DB              string `json:"db"`
	ID              string `json:"id"`
	Label           string `json:"label"`
	URL             string `json:"url"`
}
