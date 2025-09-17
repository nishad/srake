package processor

import (
	"fmt"

	"github.com/nishad/srake/internal/database"
	"github.com/nishad/srake/internal/parser"
)

// PoolHandler manages sample pooling and multiplexing relationships
type PoolHandler struct {
	db Database
}

// NewPoolHandler creates a new pool handler
func NewPoolHandler(db Database) *PoolHandler {
	return &PoolHandler{
		db: db,
	}
}

// SamplePool represents a pool relationship
type SamplePool struct {
	PoolID       int
	ParentSample string
	MemberSample string
	MemberName   string
	Proportion   float64
	ReadLabel    string
	Barcode      string
	BarcodeRead  int
}

// ExtractPoolRelationships extracts pool relationships from an experiment
func (ph *PoolHandler) ExtractPoolRelationships(exp parser.Experiment) ([]SamplePool, error) {
	if exp.Design.SampleDescriptor.Pool == nil {
		return nil, nil
	}

	pool := exp.Design.SampleDescriptor.Pool
	parentSample := exp.Design.SampleDescriptor.Accession
	var relationships []SamplePool

	// Extract default member if present
	if pool.DefaultMember != nil {
		rel := SamplePool{
			ParentSample: parentSample,
			MemberSample: pool.DefaultMember.Accession,
			MemberName:   pool.DefaultMember.MemberName,
			Proportion:   float64(pool.DefaultMember.Proportion),
		}

		// Extract read label if present
		if len(pool.DefaultMember.ReadLabels) > 0 {
			rel.ReadLabel = pool.DefaultMember.ReadLabels[0].Value
		}

		relationships = append(relationships, rel)
	}

	// Extract other pool members
	for _, member := range pool.Members {
		rel := SamplePool{
			ParentSample: parentSample,
			MemberSample: member.Accession,
			MemberName:   member.MemberName,
			Proportion:   float64(member.Proportion),
		}

		// Extract read label if present
		if len(member.ReadLabels) > 0 {
			rel.ReadLabel = member.ReadLabels[0].Value
		}

		relationships = append(relationships, rel)
	}

	return relationships, nil
}

// StorePoolRelationships stores pool relationships in the database
func (ph *PoolHandler) StorePoolRelationships(relationships []SamplePool) error {
	for _, rel := range relationships {
		dbPool := &database.SamplePool{
			ParentSample: rel.ParentSample,
			MemberSample: rel.MemberSample,
			MemberName:   rel.MemberName,
			Proportion:   rel.Proportion,
			ReadLabel:    rel.ReadLabel,
		}

		if err := ph.db.InsertSamplePool(dbPool); err != nil {
			return fmt.Errorf("failed to insert pool relationship: %w", err)
		}
	}

	return nil
}

// GetPoolMembers retrieves all members of a pool
func (ph *PoolHandler) GetPoolMembers(parentSample string) ([]SamplePool, error) {
	pools, err := ph.db.GetSamplePools(parentSample)
	if err != nil {
		return nil, fmt.Errorf("failed to get pool members: %w", err)
	}

	var relationships []SamplePool
	for _, p := range pools {
		relationships = append(relationships, SamplePool{
			PoolID:       p.PoolID,
			ParentSample: p.ParentSample,
			MemberSample: p.MemberSample,
			MemberName:   p.MemberName,
			Proportion:   p.Proportion,
			ReadLabel:    p.ReadLabel,
		})
	}

	return relationships, nil
}

// ExtractBarcodeInfo extracts barcode information from experiment attributes
func (ph *PoolHandler) ExtractBarcodeInfo(attrs []parser.Attribute) map[string]string {
	barcodeInfo := make(map[string]string)

	for _, attr := range attrs {
		switch attr.Tag {
		case "barcode", "index", "adapter":
			barcodeInfo["barcode"] = attr.Value
		case "barcode_read", "index_read":
			barcodeInfo["barcode_read"] = attr.Value
		case "dual_barcode", "dual_index":
			barcodeInfo["dual_barcode"] = attr.Value
		case "umi", "unique_molecular_identifier":
			barcodeInfo["umi"] = attr.Value
		}
	}

	return barcodeInfo
}

// ValidatePoolProportions validates that pool member proportions sum to 1.0
func (ph *PoolHandler) ValidatePoolProportions(relationships []SamplePool) error {
	if len(relationships) == 0 {
		return nil
	}

	var totalProportion float64
	hasProportions := false

	for _, rel := range relationships {
		if rel.Proportion > 0 {
			hasProportions = true
			totalProportion += rel.Proportion
		}
	}

	// Only validate if proportions are specified
	if hasProportions && (totalProportion < 0.99 || totalProportion > 1.01) {
		return fmt.Errorf("pool member proportions sum to %f, expected 1.0", totalProportion)
	}

	return nil
}

// GetPoolStatistics returns pool statistics
func (ph *PoolHandler) GetPoolStatistics() (*PoolStats, error) {
	stats := &PoolStats{}

	// Count total pools
	count, err := ph.db.CountSamplePools()
	if err != nil {
		return nil, err
	}
	stats.TotalPools = count

	// Get average pool size
	avgSize, err := ph.db.GetAveragePoolSize()
	if err != nil {
		return nil, err
	}
	stats.AveragePoolSize = avgSize

	// Get max pool size
	maxSize, err := ph.db.GetMaxPoolSize()
	if err != nil {
		return nil, err
	}
	stats.MaxPoolSize = maxSize

	return stats, nil
}

// PoolStats contains pool statistics
type PoolStats struct {
	TotalPools        int
	AveragePoolSize   float64
	MaxPoolSize       int
	PoolsWithBarcodes int
}
