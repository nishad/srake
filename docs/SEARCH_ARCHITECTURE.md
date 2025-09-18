# SRA Search Architecture Recommendations

## Executive Summary

This document outlines the recommended search architecture for the SRAKE system, including full-text search (FTS) indexing strategy and vector embedding implementation for semantic search capabilities.

## Current State Analysis

### Index Statistics
- **Total Documents**: 829,746 across 4 entity types
- **Index Size**: 335MB (Bleve index)
- **Performance**: 70-125ms for most queries
- **Coverage**: 100% of all entities

### Document Distribution
```
Studies:      4,856 (0.6%)    - Rich text content
Experiments:  268,039 (32.3%) - Technical metadata
Samples:      259,155 (31.2%) - Biological descriptions
Runs:         297,696 (35.9%) - Minimal text
```

## Recommended Architecture: Hybrid Approach

### 1. Dual Index Strategy

#### Content Index (Primary)
- **Scope**: Studies + Samples with meaningful text
- **Size**: ~110MB (68% reduction from current)
- **Purpose**: Rich text search, semantic queries
- **Fields**:
  - Studies: title, abstract, organism, study_type
  - Samples: description, organism, tissue, cell_type

#### Reference Index (Secondary)
- **Scope**: All entities, minimal fields
- **Size**: ~50MB
- **Purpose**: Accession lookups, technical parameters
- **Fields**:
  - All: accession IDs (keyword only)
  - Experiments: library_strategy, platform
  - Runs: None beyond accession

### 2. Vector Embedding Strategy

#### Tier 1: Essential (Implement First)
**Target**: Studies only
- **Model**: SapBERT or BioBERT (biomedical-specific)
- **Dimensions**: 384 (all-MiniLM-L6-v2 as fallback)
- **Fields**: `title + abstract + organism`
- **Storage**: ~8MB for 5K studies
- **Min Text Length**: 100 characters

#### Tier 2: Enhanced (Phase 2)
**Target**: Samples with descriptions
- **Selection Criteria**: description length > 50 chars
- **Fields**: `description + organism + tissue + cell_type`
- **Storage**: +100-200MB (subset of 260K samples)
- **Estimated Coverage**: 30-40% of samples

#### Tier 3: Optional (Future)
**Target**: Experiments with meaningful titles
- **Selection Criteria**: title length > 20 chars
- **Fields**: `title + library_strategy`
- **Storage**: +50MB
- **Estimated Coverage**: 10-15% of experiments

### 3. Configuration Schema

```yaml
search:
  # Index configuration
  index_strategy: hybrid  # hybrid | unified | minimal

  content_index:
    enabled: true
    path: ${CACHE_DIR}/index/content.bleve
    scope:
      studies:
        enabled: true
        fields: [title, abstract, organism, study_type, submission_date]
        field_weights:
          title: 2.0
          abstract: 1.5
          organism: 1.0
      samples:
        enabled: true
        fields: [description, organism, tissue, cell_type, scientific_name]
        min_text_length: 50
        field_weights:
          description: 1.5
          organism: 1.0
          tissue: 1.2

  reference_index:
    enabled: true
    path: ${CACHE_DIR}/index/reference.bleve
    scope:
      studies: {enabled: true, fields: [study_accession]}
      experiments: {enabled: true, fields: [experiment_accession, library_strategy, platform]}
      samples: {enabled: true, fields: [sample_accession]}
      runs: {enabled: true, fields: [run_accession]}

  # Vector configuration
  vectors:
    enabled: true
    backend: faiss  # faiss | hnswlib | sqlite-vss

    model:
      name: sapbert-base  # sapbert-base | biobert-base | all-MiniLM-L6-v2
      dimensions: 384
      max_sequence_length: 512
      batch_size: 32

    document_configs:
      study:
        enabled: true
        fields: [study_title, study_abstract, organism]
        field_weights: {title: 2.0, abstract: 1.5, organism: 1.0}
        min_text_length: 100
        max_documents: null  # Index all

      sample:
        enabled: true
        fields: [description, organism, tissue, cell_type]
        field_weights: {description: 1.5, organism: 1.0, tissue: 1.2}
        min_text_length: 50
        max_documents: 100000  # Limit for memory

      experiment:
        enabled: false  # Start disabled
        fields: [title]
        min_text_length: 20
        max_documents: 50000

      run:
        enabled: false  # Never embed runs

  # Search behavior
  search_modes:
    default: auto  # auto | fts | vector | hybrid | database

    auto_routing:
      accession_pattern: "^[SED]R[RSXP][0-9]+"  # Route to reference index
      min_query_length_for_vectors: 5  # Use vectors for longer queries

    hybrid_weights:
      text_score: 0.7
      vector_score: 0.3

  # Performance tuning
  performance:
    batch_size: 10000  # Index batch size
    flush_interval: 10  # Flush every N batches
    cache_enabled: true
    cache_ttl: 900  # 15 minutes
    max_results_default: 100
    timeout_ms: 5000
```

## Implementation Phases

### Phase 1: Optimize FTS (Week 1)
1. Implement dual index strategy
2. Add configurable field selection
3. Optimize index building with selective fields
4. Add index routing logic
5. Test performance improvements

### Phase 2: Basic Vectors (Week 2)
1. Integrate SapBERT/BioBERT model
2. Implement embedding generation for studies
3. Add vector storage backend (FAISS recommended)
4. Implement hybrid search scoring
5. Test semantic search quality

### Phase 3: Enhanced Coverage (Week 3)
1. Add sample embeddings with smart selection
2. Implement incremental vector updates
3. Add vector similarity threshold filtering
4. Optimize memory usage
5. Performance tuning

### Phase 4: Production Ready (Week 4)
1. Add monitoring and metrics
2. Implement vector index persistence
3. Add vector index rebuild capability
4. Documentation and examples
5. Performance benchmarks

## Performance Targets

### Index Size
- **Current**: 335MB (all fields, all documents)
- **Target**: 160MB total
  - Content Index: 110MB
  - Reference Index: 50MB
- **Reduction**: 52%

### Query Performance
- **Simple keywords**: < 50ms (from 70-90ms)
- **Complex queries**: < 80ms (from 110-125ms)
- **Semantic search**: < 150ms (new capability)
- **Hybrid search**: < 200ms

### Memory Usage
- **FTS Index**: 200MB resident
- **Vector Index**: 500MB for studies + selected samples
- **Total**: < 1GB for full system

## Text Preparation Functions

### For Studies
```python
def prepare_study_text(study, max_length=512):
    """Prepare study text for embedding"""
    components = []

    # Title is most important (repeat for emphasis)
    if study.title:
        components.extend([study.title, study.title])

    # Abstract provides context
    if study.abstract:
        components.append(study.abstract[:300])

    # Organism for biological context
    if study.organism:
        components.append(f"Organism: {study.organism}")

    text = " ".join(components)
    return text[:max_length]
```

### For Samples
```python
def prepare_sample_text(sample, max_length=512):
    """Prepare sample text for embedding"""
    components = []

    # Description is primary
    if sample.description:
        components.append(sample.description[:200])

    # Biological context
    context = []
    if sample.organism:
        context.append(f"organism:{sample.organism}")
    if sample.tissue:
        context.append(f"tissue:{sample.tissue}")
    if sample.cell_type:
        context.append(f"cell:{sample.cell_type}")

    if context:
        components.append(" ".join(context))

    text = " ".join(components)
    return text[:max_length] if text else None
```

## Query Examples and Expected Behavior

### Text Search (FTS)
```bash
# Simple keyword - routes to content index
srake search "breast cancer"
# Expected: Finds studies/samples with these terms

# Accession - routes to reference index
srake search "SRP123456"
# Expected: Exact match lookup

# Technical term - routes to content index
srake search "RNA-Seq"
# Expected: Finds all RNA-Seq experiments
```

### Semantic Search (Vectors)
```bash
# Disease concept search
srake search "neurodegenerative disorders" --mode vector
# Expected: Finds Alzheimer's, Parkinson's, ALS studies

# Methodology search
srake search "single cell sequencing" --mode vector
# Expected: Finds scRNA-seq, Drop-seq, 10x Genomics

# Cross-species search
srake search "mouse model diabetes" --mode vector
# Expected: Finds db/db mice, ob/ob mice, STZ-induced
```

### Hybrid Search
```bash
# Combined text + semantic
srake search "BRCA1 tumor" --mode hybrid
# Expected: Exact BRCA1 matches + related cancer studies

# Filtered semantic search
srake search "inflammation" --organism "homo sapiens" --mode hybrid
# Expected: Human inflammation studies with semantic expansion
```

## Monitoring Metrics

### Key Performance Indicators
1. **Index Build Time**: Target < 5 minutes for 1M documents
2. **Query Latency P95**: Target < 200ms
3. **Index Size Ratio**: Target < 0.5MB per 1000 docs
4. **Vector Generation Rate**: Target > 100 docs/second
5. **Cache Hit Rate**: Target > 60%

### Health Checks
```yaml
health_checks:
  - name: index_document_count
    threshold: 0.95  # 95% of database records indexed
  - name: index_size_ratio
    threshold: 0.5  # MB per 1000 documents
  - name: query_latency_p95
    threshold: 200  # milliseconds
  - name: vector_coverage
    threshold: 0.90  # 90% of eligible documents have vectors
```

## Migration Strategy

### From Current Full Index
1. **Parallel Operation**: Run new indexes alongside current
2. **A/B Testing**: Route 10% traffic to new indexes
3. **Validation**: Compare result quality
4. **Gradual Rollout**: Increase traffic percentage
5. **Cutover**: Switch completely after validation

### Rollback Plan
1. Keep original index for 30 days
2. Configuration flag to revert
3. No data migration required (read from DB)

## Cost-Benefit Analysis

### Benefits
1. **52% reduction in index size** (335MB → 160MB)
2. **30-40% faster query performance**
3. **New semantic search capability**
4. **Lower memory footprint**
5. **Faster index rebuilds**

### Costs
1. **Development time**: ~4 weeks
2. **Additional complexity**: Dual indexes
3. **Vector model storage**: ~150MB for model files
4. **CPU usage**: Higher during embedding generation

### ROI
- **Break-even**: At 10K daily searches
- **Performance gain**: 40ms saved per query
- **User satisfaction**: Semantic search adds significant value
- **Scalability**: Supports 10x data growth

## Conclusion

The recommended hybrid approach with dual indexes and tiered vector embeddings provides:
- **Optimal performance** with 52% smaller indexes
- **Rich search capabilities** including semantic search
- **Scalable architecture** for future growth
- **Configurable implementation** for different use cases

This architecture balances performance, functionality, and resource usage while maintaining flexibility for future enhancements.

## Appendix: Search Test Results

### Current System Performance (829K documents)
- Simple keyword search: 70-90ms
- Multi-word queries: 110-125ms
- Large result sets (1000): ~80ms
- Wildcard search ("*"): 13 seconds (performance issue)
- Accession lookup: 5-10ms

### Search Behavior Observations
1. Text search finds content in all text fields
2. Accession searches return exact matches
3. Multi-word queries use OR logic by default
4. Special characters handled correctly
5. Database mode lacks text search capability

### Recommended Improvements
1. Implement AND logic option for multi-word queries
2. Add query syntax parser for advanced searches
3. Optimize wildcard queries with result limiting
4. Add search result ranking improvements
5. Implement search suggestions/autocomplete