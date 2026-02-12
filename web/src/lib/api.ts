import type { SearchParams, SearchResult, StudyDetails, ExportOptions } from './utils';

const API_BASE = '/api/v1';

export class ApiService {
  static async search(params: SearchParams): Promise<{
    results: SearchResult[];
    total: number;
    offset: number;
    limit: number;
  }> {
    const queryParams = new URLSearchParams();

    if (params.query) queryParams.append('query', params.query);
    if (params.libraryStrategy) queryParams.append('library_strategy', params.libraryStrategy);
    if (params.platform) queryParams.append('platform', params.platform);
    if (params.organism) queryParams.append('organism', params.organism);
    if (params.similarityThreshold !== undefined) queryParams.append('similarity_threshold', params.similarityThreshold.toString());
    if (params.minScore !== undefined) queryParams.append('min_score', params.minScore.toString());
    if (params.topPercentile !== undefined) queryParams.append('top_percentile', params.topPercentile.toString());
    if (params.searchMode) queryParams.append('search_mode', params.searchMode);
    if (params.limit) queryParams.append('limit', params.limit.toString());
    if (params.offset) queryParams.append('offset', params.offset.toString());
    if (params.showConfidence) queryParams.append('show_confidence', 'true');

    const response = await fetch(`${API_BASE}/search?${queryParams}`);
    if (!response.ok) throw new Error('Search failed');
    return response.json();
  }

  static async getStudyDetails(studyId: string): Promise<StudyDetails> {
    const response = await fetch(`${API_BASE}/studies/${studyId}`);
    if (!response.ok) throw new Error('Failed to fetch study details');
    return response.json();
  }

  static async getRunDetails(runId: string): Promise<any> {
    const response = await fetch(`${API_BASE}/runs/${runId}`);
    if (!response.ok) throw new Error('Failed to fetch run details');
    return response.json();
  }

  static async getSampleDetails(sampleId: string): Promise<any> {
    const response = await fetch(`${API_BASE}/samples/${sampleId}`);
    if (!response.ok) throw new Error('Failed to fetch sample details');
    return response.json();
  }

  static async exportResults(searchParams: SearchParams, exportOptions: ExportOptions): Promise<Blob> {
    const response = await fetch(`${API_BASE}/export`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({
        ...searchParams,
        ...exportOptions
      })
    });

    if (!response.ok) throw new Error('Export failed');
    return response.blob();
  }

  static async getStats(): Promise<{
    total_documents: number;
    indexed_documents: number;
    index_size: number;
    last_indexed?: string;
    last_updated: string;
    top_organisms?: Array<{ name: string; count: number }>;
    top_platforms?: Array<{ name: string; count: number }>;
    top_strategies?: Array<{ name: string; count: number }>;
  }> {
    const response = await fetch(`${API_BASE}/stats`);
    if (!response.ok) throw new Error('Failed to fetch statistics');
    return response.json();
  }

  static async getHealth(): Promise<{
    status: string;
    database: string;
    search_index: string;
    metadata_service?: string;
    search_service?: string;
    timestamp?: string;
  }> {
    const response = await fetch(`${API_BASE}/health`);
    if (!response.ok) throw new Error('Health check failed');
    return response.json();
  }

  static async getAggregations(field: string): Promise<{
    field: string;
    values: Array<{
      value: string;
      count: number;
    }>;
  }> {
    const response = await fetch(`${API_BASE}/aggregations/${field}`);
    if (!response.ok) throw new Error('Failed to fetch aggregations');
    return response.json();
  }
}