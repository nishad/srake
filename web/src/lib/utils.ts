import { type ClassValue, clsx } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export const API_BASE_URL = '/api/v1';

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

export interface SearchResult {
  id: string;
  type: string;
  title?: string;
  abstract?: string;
  score?: number;
  similarity?: number;
  confidence?: 'high' | 'medium' | 'low';
  organism?: string;
  platform?: string;
  library_strategy?: string;
  submission_date?: string;
  update_date?: string;
  sample_count?: number;
  run_count?: number;
  fields?: Record<string, any>;
  highlights?: Record<string, string[]>;
}

export interface StudyDetails {
  study_id: string;
  study_title: string;
  study_abstract: string;
  study_type: string;
  center_name: string;
  submission_date: string;
  update_date: string;
  experiments: any[];
  samples: any[];
  runs: any[];
}

export interface ExportOptions {
  format: 'json' | 'csv' | 'tsv' | 'xml';
  fields?: string[];
  includeHeaders?: boolean;
}