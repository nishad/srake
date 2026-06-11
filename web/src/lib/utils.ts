import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const API_BASE_URL = '/api/v1';

// --- Search Types (matching Go service/types.go) ---

export interface SearchParams {
  query?: string;
  libraryStrategy?: string;
  platform?: string;
  organism?: string;
  similarityThreshold?: number;
  minScore?: number;
  topPercentile?: number;
  searchMode?: 'database' | 'fts' | 'hybrid' | 'vector';
  limit?: number;
  offset?: number;
  showConfidence?: boolean;
}

export interface SearchResponse {
  results: SearchResult[];
  total_results: number;
  query: string;
  time_taken_ms: number;
  search_mode?: string;
  facets?: Record<string, any>;
}

export interface SearchResult {
  id: string;
  type: string;
  title?: string;
  description?: string;
  abstract?: string;
  organism?: string;
  platform?: string;
  library_strategy?: string;
  score?: number;
  similarity?: number;
  confidence?: 'high' | 'medium' | 'low';
  fields?: Record<string, any>;
  highlights?: Record<string, string[]>;
  // Extended fields from browse
  submission_date?: string;
  update_date?: string;
  sample_count?: number;
  run_count?: number;
  experiment_count?: number;
  external_links?: string;
  keywords?: string;
  tags?: string;
  // Additional metadata fields
  accession?: string;
  library_source?: string;
  library_selection?: string;
  instrument_model?: string;
  total_bases?: number;
  total_size?: number;
}

// --- Metadata Types (matching Go database/models.go) ---

export interface Study {
  study_accession: string;
  alias: string;
  center_name: string;
  broker_name: string;
  study_title: string;
  study_type: string;
  study_abstract: string;
  study_description: string;
  center_project_name: string;
  submission_date?: string;
  first_public?: string;
  last_update?: string;
  primary_id: string;
  secondary_ids: string;
  external_ids: string;
  submitter_ids: string;
  study_links: string;
  study_attributes: string;
  related_studies: string;
  organism: string;
  metadata: string;
}

export interface Experiment {
  experiment_accession: string;
  alias: string;
  center_name: string;
  broker_name: string;
  study_accession: string;
  sample_accession: string;
  title: string;
  design_description: string;
  library_name: string;
  library_strategy: string;
  library_source: string;
  library_selection: string;
  library_layout: string;
  library_construction_protocol: string;
  nominal_length: number;
  nominal_sdev: number;
  platform: string;
  instrument_model: string;
  spot_length: number;
  experiment_links: string;
  experiment_attributes: string;
  metadata: string;
}

export interface Sample {
  sample_accession: string;
  alias: string;
  center_name: string;
  broker_name: string;
  title: string;
  description: string;
  taxon_id: number;
  scientific_name: string;
  common_name: string;
  organism: string;
  tissue: string;
  cell_type: string;
  cell_line: string;
  strain: string;
  sex: string;
  age: string;
  disease: string;
  treatment: string;
  geo_loc_name: string;
  lat_lon: string;
  collection_date: string;
  env_biome: string;
  env_feature: string;
  env_material: string;
  sample_links: string;
  sample_attributes: string;
  biosample_accession: string;
  bioproject_accession: string;
  metadata: string;
}

export interface Run {
  run_accession: string;
  alias: string;
  center_name: string;
  broker_name: string;
  run_center: string;
  experiment_accession: string;
  title: string;
  run_date?: string;
  total_spots: number;
  total_bases: number;
  total_size: number;
  load_done: boolean;
  published: string;
  data_files: string;
  run_links: string;
  run_attributes: string;
  quality_score_mean: number;
  quality_score_std: number;
  read_count_r1: number;
  read_count_r2: number;
  metadata: string;
}

// --- Stats Types ---

export interface StatsResponse {
  total_studies: number;
  total_experiments: number;
  total_samples: number;
  total_runs: number;
  last_update: string;
  index_size: number;
  database_size: number;
  // Also from SearchStats
  total_documents?: number;
  indexed_documents?: number;
  last_indexed?: string;
  last_updated?: string;
  top_organisms?: CountItem[];
  top_platforms?: CountItem[];
  top_strategies?: CountItem[];
}

export interface CountItem {
  name: string;
  count: number;
}

export interface HealthResponse {
  status: string;
  database: string;
  search_index: string;
  metadata_service?: string;
  search_service?: string;
  timestamp?: string;
}

// --- Export Types ---

export interface ExportOptions {
  format: 'json' | 'csv' | 'tsv' | 'xml' | 'jsonl';
  fields?: string[];
  limit?: number;
}

export interface ExportRequest {
  query: string;
  filters?: Record<string, string>;
  format: string;
  limit?: number;
  fields?: string[];
}

// --- Utility Functions ---

export function formatNumber(num: number): string {
  return new Intl.NumberFormat('en-US').format(num);
}

export function formatCompactNumber(num: number): string {
  if (num >= 1_000_000_000) return (num / 1_000_000_000).toFixed(1) + 'B';
  if (num >= 1_000_000) return (num / 1_000_000).toFixed(1) + 'M';
  if (num >= 1_000) return (num / 1_000).toFixed(1) + 'K';
  return num.toString();
}

export function formatBytes(bytes: number): string {
  if (bytes >= 1_073_741_824) return (bytes / 1_073_741_824).toFixed(2) + ' GB';
  if (bytes >= 1_048_576) return (bytes / 1_048_576).toFixed(2) + ' MB';
  if (bytes >= 1024) return (bytes / 1024).toFixed(2) + ' KB';
  return bytes + ' B';
}

export function formatDate(dateStr?: string | null): string {
  if (!dateStr) return 'N/A';
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'short',
    day: 'numeric'
  });
}

export function formatDateLong(dateStr?: string | null): string {
  if (!dateStr) return 'N/A';
  return new Date(dateStr).toLocaleDateString('en-US', {
    year: 'numeric',
    month: 'long',
    day: 'numeric'
  });
}

export function tryParseJSON(str: string): any {
  if (!str) return null;
  try {
    return JSON.parse(str);
  } catch {
    return null;
  }
}
