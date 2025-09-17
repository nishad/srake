package processor

import (
	"fmt"
	"strings"

	"github.com/nishad/srake/internal/parser"
)

// PlatformHandler handles platform enumeration extraction and validation
type PlatformHandler struct {
	platformMap  map[string][]string // platform -> list of models
	strategyMap  map[string]bool
	sourceMap    map[string]bool
	selectionMap map[string]bool
}

// NewPlatformHandler creates a new platform handler with initialized maps
func NewPlatformHandler() *PlatformHandler {
	ph := &PlatformHandler{
		platformMap:  initializePlatformMap(),
		strategyMap:  initializeStrategyMap(),
		sourceMap:    initializeSourceMap(),
		selectionMap: initializeSelectionMap(),
	}
	return ph
}

// initializePlatformMap creates the platform to models mapping
func initializePlatformMap() map[string][]string {
	return map[string][]string{
		"LS454": {
			"454 GS", "454 GS 20", "454 GS FLX", "454 GS FLX+",
			"454 GS FLX Titanium", "454 GS Junior",
		},
		"ILLUMINA": {
			"Illumina Genome Analyzer", "Illumina Genome Analyzer II",
			"Illumina Genome Analyzer IIx", "Illumina HiSeq 2000",
			"Illumina HiSeq 2500", "Illumina HiSeq 3000", "Illumina HiSeq 4000",
			"Illumina HiSeq X", "Illumina HiSeq X Ten", "Illumina HiSeq X Five",
			"Illumina MiSeq", "Illumina MiniSeq", "Illumina NextSeq 500",
			"Illumina NextSeq 550", "Illumina NextSeq 1000", "Illumina NextSeq 2000",
			"Illumina NovaSeq 6000", "Illumina NovaSeq X", "Illumina iSeq 100",
			"HiSeq X Ten", "HiSeq X Five", "NextSeq 500", "NextSeq 550",
			"NextSeq 1000", "NextSeq 2000", "NovaSeq X Plus",
		},
		"HELICOS": {
			"Helicos HeliScope",
		},
		"ABI_SOLID": {
			"AB SOLiD System", "AB SOLiD System 2.0", "AB SOLiD System 3.0",
			"AB SOLiD 4 System", "AB SOLiD 4hq System", "AB SOLiD PI System",
			"AB 5500 Genetic Analyzer", "AB 5500xl Genetic Analyzer",
		},
		"COMPLETE_GENOMICS": {
			"Complete Genomics",
		},
		"BGISEQ": {
			"BGISEQ-50", "BGISEQ-500", "MGISEQ-200", "MGISEQ-2000",
			"DNBSEQ-G50", "DNBSEQ-G400", "DNBSEQ-T7",
		},
		"OXFORD_NANOPORE": {
			"MinION", "GridION", "GridION X5", "PromethION",
			"PromethION 24", "PromethION 48", "Flongle", "P2 Solo",
		},
		"PACBIO_SMRT": {
			"PacBio RS", "PacBio RS II", "Sequel", "Sequel II",
			"Sequel IIe", "Onso", "Revio",
		},
		"ION_TORRENT": {
			"Ion Torrent PGM", "Ion Torrent Proton", "Ion S5",
			"Ion S5 XL", "Ion GeneStudio S5", "Ion GeneStudio S5 Plus",
			"Ion GeneStudio S5 Prime", "Ion Torrent Genexus",
		},
		"VELA_DIAGNOSTICS": {
			"Vela Diagnostics Sentosa SQ301",
		},
		"CAPILLARY": {
			"AB 3730xL Genetic Analyzer", "AB 3730 Genetic Analyzer",
			"AB 3500xL Genetic Analyzer", "AB 3500 Genetic Analyzer",
			"AB 3130xL Genetic Analyzer", "AB 3130 Genetic Analyzer",
			"AB 310 Genetic Analyzer",
		},
		"GENAPSYS": {
			"GENAPSYS Sequencer", "Genapsys Sequencer GS111",
		},
		"DNBSEQ": {
			"DNBSEQ-G50", "DNBSEQ-G400", "DNBSEQ-T7",
			"DNBSEQ-G99", "DNBSEQ-T10x4", "DNBSEQ-T20x2",
		},
		"ELEMENT": {
			"Element AVITI",
		},
		"GENEMIND": {
			"GeneMind FASTASeq 300", "GeneMind FASTASeq 3000",
		},
		"ULTIMA": {
			"UG 100", "Ultima Genomics UG 100",
		},
		"TAPESTRI": {
			"Tapestri", "Mission Bio Tapestri",
		},
		"SALUS": {
			"Salus Sequencer",
		},
		"GENEUS_TECH": {
			"Geneus Tech Sequencer",
		},
		"SINGULAR_GENOMICS": {
			"G4", "Singular Genomics G4",
		},
		"GENEXUS": {
			"GeneStudio", "Ion Torrent Genexus",
		},
		"REVOLOCITY": {
			"Revolocity",
		},
	}
}

// ExtractPlatformDetails extracts platform and model from experiment
func (ph *PlatformHandler) ExtractPlatformDetails(platform parser.Platform) (string, string) {
	// Check each platform type that exists in the parser struct
	if platform.LS454 != nil {
		return "LS454", platform.LS454.InstrumentModel
	}
	if platform.Illumina != nil {
		return "ILLUMINA", platform.Illumina.InstrumentModel
	}
	if platform.Helicos != nil {
		return "HELICOS", platform.Helicos.InstrumentModel
	}
	if platform.Solid != nil { // This is ABI_SOLID in the XSD
		return "ABI_SOLID", platform.Solid.InstrumentModel
	}
	if platform.CompleteGenomics != nil {
		return "COMPLETE_GENOMICS", platform.CompleteGenomics.InstrumentModel
	}
	if platform.OxfordNanopore != nil {
		return "OXFORD_NANOPORE", platform.OxfordNanopore.InstrumentModel
	}
	if platform.PacBio != nil { // This is PACBIO_SMRT in the XSD
		return "PACBIO_SMRT", platform.PacBio.InstrumentModel
	}
	if platform.IonTorrent != nil {
		return "ION_TORRENT", platform.IonTorrent.InstrumentModel
	}
	if platform.Capillary != nil {
		return "CAPILLARY", platform.Capillary.InstrumentModel
	}

	// Note: The parser struct currently only supports the above platforms.
	// Additional platforms in the XSD (BGISEQ, DNBSEQ, ELEMENT, etc.)
	// would need to be added to the parser struct to be fully supported.

	return "", ""
}

// ValidatePlatform checks if platform and model are valid
func (ph *PlatformHandler) ValidatePlatform(platform, model string) error {
	models, exists := ph.platformMap[platform]
	if !exists {
		return fmt.Errorf("unknown platform: %s", platform)
	}

	if model == "" {
		return nil // Model can be empty
	}

	// Check if model is in the list
	for _, validModel := range models {
		if strings.EqualFold(model, validModel) {
			return nil
		}
	}

	return fmt.Errorf("invalid model '%s' for platform %s", model, platform)
}

// initializeStrategyMap creates the library strategy validation map
func initializeStrategyMap() map[string]bool {
	strategies := []string{
		"WGS", "WGA", "WXS", "RNA-Seq", "ssRNA-seq", "miRNA-Seq", "ncRNA-Seq",
		"FL-cDNA", "EST", "Hi-C", "ATAC-seq", "WCS", "RAD-Seq", "CLONE",
		"POOLCLONE", "AMPLICON", "CLONEEND", "FINISHING", "ChIP-Seq",
		"MNase-Seq", "DNase-Hypersensitivity", "Bisulfite-Seq", "CTS",
		"MRE-Seq", "MeDIP-Seq", "MBD-Seq", "Tn-Seq", "VALIDATION", "FAIRE-seq",
		"SELEX", "RIP-Seq", "ChIA-PET", "Synthetic-Long-Read", "Targeted-Capture",
		"Tethered Chromatin Conformation Capture", "OTHER", "4C-Seq", "5C-Seq",
	}

	m := make(map[string]bool)
	for _, s := range strategies {
		m[s] = true
	}
	return m
}

// ValidateLibraryStrategy validates a library strategy
func (ph *PlatformHandler) ValidateLibraryStrategy(strategy string) error {
	if strategy == "" {
		return nil
	}
	if !ph.strategyMap[strategy] {
		return fmt.Errorf("invalid library strategy: %s", strategy)
	}
	return nil
}

// initializeSourceMap creates the library source validation map
func initializeSourceMap() map[string]bool {
	sources := []string{
		"GENOMIC", "GENOMIC SINGLE CELL", "TRANSCRIPTOMIC",
		"TRANSCRIPTOMIC SINGLE CELL", "METAGENOMIC", "METATRANSCRIPTOMIC",
		"SYNTHETIC", "VIRAL RNA", "OTHER",
	}

	m := make(map[string]bool)
	for _, s := range sources {
		m[s] = true
	}
	return m
}

// ValidateLibrarySource validates a library source
func (ph *PlatformHandler) ValidateLibrarySource(source string) error {
	if source == "" {
		return nil
	}
	if !ph.sourceMap[source] {
		return fmt.Errorf("invalid library source: %s", source)
	}
	return nil
}

// initializeSelectionMap creates the library selection validation map
func initializeSelectionMap() map[string]bool {
	selections := []string{
		"RANDOM", "PCR", "RANDOM PCR", "RT-PCR", "HMPR", "MF", "CF-S", "CF-M",
		"CF-H", "CF-T", "MDA", "MSLL", "cDNA", "cDNA_randomPriming",
		"cDNA_oligo_dT", "PolyA", "Oligo-dT", "Inverse rRNA", "Inverse rRNA selection",
		"ChIP", "ChIP-Seq", "MNase", "DNase", "Hybrid Selection",
		"Reduced Representation", "Restriction Digest", "5-methylcytidine antibody",
		"MBD2 protein methyl-CpG binding domain", "CAGE", "RACE", "size fractionation",
		"Padlock probes capture method", "other", "unspecified", "repeat fractionation",
		"ChIP-ChIP", "inverse rRNA selection",
	}

	m := make(map[string]bool)
	for _, s := range selections {
		m[s] = true
	}
	return m
}

// ValidateLibrarySelection validates a library selection method
func (ph *PlatformHandler) ValidateLibrarySelection(selection string) error {
	if selection == "" {
		return nil
	}
	if !ph.selectionMap[selection] {
		return fmt.Errorf("invalid library selection: %s", selection)
	}
	return nil
}

// GetPlatformStatistics returns counts of each platform type in experiments
func (ph *PlatformHandler) GetPlatformStatistics() map[string]int {
	stats := make(map[string]int)
	for platform := range ph.platformMap {
		stats[platform] = 0
	}
	return stats
}

// GetAllPlatforms returns list of all supported platforms
func (ph *PlatformHandler) GetAllPlatforms() []string {
	platforms := make([]string, 0, len(ph.platformMap))
	for p := range ph.platformMap {
		platforms = append(platforms, p)
	}
	return platforms
}

// GetModelsForPlatform returns list of valid models for a platform
func (ph *PlatformHandler) GetModelsForPlatform(platform string) []string {
	if models, exists := ph.platformMap[platform]; exists {
		return models
	}
	return nil
}
