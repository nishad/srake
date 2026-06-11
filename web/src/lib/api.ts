import type {
  SearchParams,
  SearchResponse,
  Study,
  Experiment,
  Sample,
  Run,
  StatsResponse,
  HealthResponse,
  ExportRequest
} from './utils';

const API_BASE = '/api/v1';

async function fetchJSON<T>(url: string, init?: RequestInit): Promise<T> {
  const response = await fetch(url, init);
  if (!response.ok) {
    const body = await response.text();
    let message = `Request failed: ${response.status}`;
    try {
      const err = JSON.parse(body);
      if (err.message) message = err.message;
    } catch { /* use default message */ }
    throw new Error(message);
  }
  return response.json();
}

export class ApiService {
  // --- Search ---

  static async search(params: SearchParams): Promise<SearchResponse> {
    const q = new URLSearchParams();
    if (params.query) q.append('q', params.query);
    if (params.libraryStrategy) q.append('library_strategy', params.libraryStrategy);
    if (params.platform) q.append('platform', params.platform);
    if (params.organism) q.append('organism', params.organism);
    if (params.similarityThreshold !== undefined) q.append('similarity_threshold', params.similarityThreshold.toString());
    if (params.minScore !== undefined) q.append('min_score', params.minScore.toString());
    if (params.topPercentile !== undefined) q.append('top_percentile', params.topPercentile.toString());
    if (params.searchMode) q.append('search_mode', params.searchMode);
    if (params.limit) q.append('limit', params.limit.toString());
    if (params.offset) q.append('offset', params.offset.toString());
    if (params.showConfidence) q.append('show_confidence', 'true');

    return fetchJSON<SearchResponse>(`${API_BASE}/search?${q}`);
  }

  static async advancedSearch(params: SearchParams): Promise<SearchResponse> {
    return fetchJSON<SearchResponse>(`${API_BASE}/search/advanced`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(params)
    });
  }

  // --- Metadata: Studies ---

  static async getStudy(accession: string): Promise<Study> {
    return fetchJSON<Study>(`${API_BASE}/studies/${accession}`);
  }

  static async listStudies(limit = 20, offset = 0): Promise<{ studies: Study[]; limit: number; offset: number }> {
    return fetchJSON(`${API_BASE}/studies?limit=${limit}&offset=${offset}`);
  }

  static async getStudyMetadata(accession: string): Promise<any> {
    return fetchJSON(`${API_BASE}/studies/${accession}/metadata`);
  }

  static async getStudyExperiments(accession: string): Promise<{ study_accession: string; experiments: Experiment[]; total: number }> {
    return fetchJSON(`${API_BASE}/studies/${accession}/experiments`);
  }

  static async getStudySamples(accession: string): Promise<{ study_accession: string; samples: Sample[]; total: number }> {
    return fetchJSON(`${API_BASE}/studies/${accession}/samples`);
  }

  static async getStudyRuns(accession: string, limit = 100): Promise<{ study_accession: string; runs: Run[]; total: number; limit: number }> {
    return fetchJSON(`${API_BASE}/studies/${accession}/runs?limit=${limit}`);
  }

  // --- Metadata: Individual entities ---

  static async getExperiment(accession: string): Promise<Experiment> {
    return fetchJSON<Experiment>(`${API_BASE}/experiments/${accession}`);
  }

  static async getSample(accession: string): Promise<Sample> {
    return fetchJSON<Sample>(`${API_BASE}/samples/${accession}`);
  }

  static async getRun(accession: string): Promise<Run> {
    return fetchJSON<Run>(`${API_BASE}/runs/${accession}`);
  }

  // --- Statistics ---

  static async getStats(): Promise<StatsResponse> {
    return fetchJSON<StatsResponse>(`${API_BASE}/stats`);
  }

  static async getOrganismStats(): Promise<{ organisms: Array<{ name: string; count: number }>; total: number }> {
    return fetchJSON(`${API_BASE}/stats/organisms`);
  }

  static async getPlatformStats(): Promise<{ platforms: Array<{ name: string; count: number }>; total: number }> {
    return fetchJSON(`${API_BASE}/stats/platforms`);
  }

  static async getStrategyStats(): Promise<{ strategies: Array<{ name: string; count: number }>; total: number }> {
    return fetchJSON(`${API_BASE}/stats/strategies`);
  }

  // --- Health ---

  static async getHealth(): Promise<HealthResponse> {
    return fetchJSON<HealthResponse>(`${API_BASE}/health`);
  }

  // --- Export ---

  static async exportResults(req: ExportRequest): Promise<Blob> {
    const response = await fetch(`${API_BASE}/export`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req)
    });
    if (!response.ok) throw new Error('Export failed');
    return response.blob();
  }
}
