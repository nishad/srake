package processor

import (
	"time"

	"github.com/nishad/srake/internal/validator"
)

// Note: Database interface is already defined in http_stream.go
// We use that interface throughout the processor package

// ComprehensiveExtractor handles comprehensive extraction of SRA XML data
type ComprehensiveExtractor struct {
	db                Database
	stats             ExtractionStats
	options           ExtractionOptions
	validator         *validator.Validator
	platformHandler   *PlatformHandler
	poolHandler       *PoolHandler
	identifierHandler *IdentifierHandler
}

// ExtractionStats tracks extraction statistics
type ExtractionStats struct {
	StudiesProcessed     int
	ExperimentsProcessed int
	SamplesProcessed     int
	RunsProcessed        int
	AnalysesProcessed    int
	SubmissionsProcessed int
	StudiesExtracted     int
	ExperimentsExtracted int
	SamplesExtracted     int
	RunsExtracted        int
	AnalysesExtracted    int
	SubmissionsExtracted int
	ValidationErrors     int
	ValidationWarnings   int
	Errors               []string
	StartTime            time.Time
}

// ExtractionOptions configures extraction behavior
type ExtractionOptions struct {
	ExtractAttributes     bool // Extract all custom attributes
	ExtractLinks          bool // Extract all external links
	NormalizeOrganism     bool // Normalize organism names
	ExtractFromAttributes bool // Extract known fields from attributes
	BatchSize             int  // Batch size for database operations
	ValidateXML           bool // Validate XML against schemas
	StrictValidation      bool // Fail on validation errors (vs warnings only)
	LogValidationIssues   bool // Log validation issues
}

// DefaultExtractionOptions returns default extraction options
func DefaultExtractionOptions() ExtractionOptions {
	return ExtractionOptions{
		ExtractAttributes:     true,
		ExtractLinks:          true,
		NormalizeOrganism:     true,
		ExtractFromAttributes: true,
		BatchSize:             1000,
		ValidateXML:           true,
		StrictValidation:      false,
		LogValidationIssues:   true,
	}
}

// Platform enumeration constants
var SupportedPlatforms = []string{
	"LS454", "ILLUMINA", "HELICOS", "ABI_SOLID", "COMPLETE_GENOMICS",
	"BGISEQ", "OXFORD_NANOPORE", "PACBIO_SMRT", "ION_TORRENT",
	"VELA_DIAGNOSTICS", "CAPILLARY", "GENAPSYS", "DNBSEQ", "ELEMENT",
	"GENEMIND", "ULTIMA", "TAPESTRI", "SALUS", "GENEUS_TECH",
	"SINGULAR_GENOMICS", "GENEXUS", "REVOLOCITY",
}

// Library strategy enumeration constants
var SupportedLibraryStrategies = []string{
	"WGS", "WGA", "WXS", "RNA-Seq", "ssRNA-seq", "miRNA-Seq", "ncRNA-Seq",
	"FL-cDNA", "EST", "Hi-C", "ATAC-seq", "WCS", "RAD-Seq", "CLONE",
	"POOLCLONE", "AMPLICON", "CLONEEND", "FINISHING", "ChIP-Seq",
	"MNase-Seq", "DNase-Hypersensitivity", "Bisulfite-Seq", "CTS",
	"MRE-Seq", "MeDIP-Seq", "MBD-Seq", "Tn-Seq", "VALIDATION", "FAIRE-seq",
	"SELEX", "RIP-Seq", "ChIA-PET", "Synthetic-Long-Read", "Targeted-Capture",
	"Tethered Chromatin Conformation Capture", "OTHER",
}

// Library source enumeration constants
var SupportedLibrarySources = []string{
	"GENOMIC", "GENOMIC SINGLE CELL", "TRANSCRIPTOMIC",
	"TRANSCRIPTOMIC SINGLE CELL", "METAGENOMIC", "METATRANSCRIPTOMIC",
	"SYNTHETIC", "VIRAL RNA", "OTHER",
}

// Library selection enumeration constants
var SupportedLibrarySelections = []string{
	"RANDOM", "PCR", "RANDOM PCR", "RT-PCR", "HMPR", "MF", "CF-S", "CF-M",
	"CF-H", "CF-T", "MDA", "MSLL", "cDNA", "cDNA_randomPriming",
	"cDNA_oligo_dT", "PolyA", "Oligo-dT", "Inverse rRNA", "Inverse rRNA selection",
	"ChIP", "ChIP-Seq", "MNase", "DNase", "Hybrid Selection",
	"Reduced Representation", "Restriction Digest", "5-methylcytidine antibody",
	"MBD2 protein methyl-CpG binding domain", "CAGE", "RACE", "size fractionation",
	"Padlock probes capture method", "other", "unspecified",
}

// Analysis type enumeration constants
var SupportedAnalysisTypes = []string{
	"REFERENCE_ALIGNMENT", "SEQUENCE_VARIATION", "SEQUENCE_ASSEMBLY",
	"SEQUENCE_ANNOTATION", "REFERENCE_SEQUENCE", "SAMPLE_PHENOTYPE",
	"TRANSCRIPTOME_ASSEMBLY", "TAXONOMIC_REFERENCE_SET", "DE_NOVO_ASSEMBLY",
	"GENOME_MAP", "AMR_ANTIBIOGRAM", "PATHOGEN_ANALYSIS",
	"PROCESSED_READS", "SEQUENCE_FLATFILE",
}

// File type enumeration constants
var SupportedFileTypes = []string{
	"sra", "srf", "sff", "fastq", "fasta", "tab", "bam", "bai", "cram", "crai",
	"vcf", "bcf", "vcf_aggregate", "bcf_aggregate", "gff", "gtf", "bed", "bigwig",
	"wiggle", "454_native", "Illumina_native", "Illumina_native_qseq",
	"Illumina_native_scarf", "Illumina_native_fastq", "SOLiD_native",
	"SOLiD_native_csfasta", "SOLiD_native_qual", "PacBio_HDF5",
	"CompleteGenomics_native", "OxfordNanopore_native", "agp", "unlocalised_list",
	"info", "manifest", "readme", "phenotype_file", "BioNano_native",
	"Bionano_native", "chromosome_list", "sample_list", "other",
}
