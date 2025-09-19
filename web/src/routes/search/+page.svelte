<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import type { SearchParams, SearchResult } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Input } from '$lib/components/ui/input';
  import { Button } from '$lib/components/ui/button';
  import { Label } from '$lib/components/ui/label';
  import * as Select from '$lib/components/ui/select';
  import { Badge } from '$lib/components/ui/badge';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import * as Tabs from '$lib/components/ui/tabs';
  import { Search, Filter, ChevronRight, ExternalLink } from 'lucide-svelte';

  let searchQuery = $state('');
  let results = $state<SearchResult[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let totalResults = $state(0);
  let currentPage = $state(1);
  let itemsPerPage = 20;

  let filters = $state({
    libraryStrategy: '',
    platform: '',
    organism: '',
    searchMode: 'hybrid' as SearchParams['searchMode']
  });

  let showAdvanced = $state(false);
  let advancedOptions = $state({
    similarityThreshold: 0.7,
    minScore: 0,
    showConfidence: true
  });

  // Parse URL parameters on mount
  onMount(() => {
    const urlParams = new URLSearchParams(window.location.search);

    const query = urlParams.get('query');
    if (query) searchQuery = query;

    const platform = urlParams.get('platform');
    if (platform) filters.platform = platform;

    const libraryStrategy = urlParams.get('libraryStrategy');
    if (libraryStrategy) filters.libraryStrategy = libraryStrategy;

    const organism = urlParams.get('organism');
    if (organism) filters.organism = organism;

    // Auto-search if we have parameters from browse
    if (query || platform || libraryStrategy || organism) {
      handleSearch();
    }
  });

  async function handleSearch(resetPage = true) {
    if (resetPage) currentPage = 1;
    loading = true;
    error = null;

    try {
      const params: SearchParams = {
        query: searchQuery,
        limit: itemsPerPage,
        offset: (currentPage - 1) * itemsPerPage,
        searchMode: filters.searchMode,
        showConfidence: advancedOptions.showConfidence
      };

      if (filters.libraryStrategy) params.libraryStrategy = filters.libraryStrategy;
      if (filters.platform) params.platform = filters.platform;
      if (filters.organism) params.organism = filters.organism;
      if (showAdvanced) {
        params.similarityThreshold = advancedOptions.similarityThreshold;
        params.minScore = advancedOptions.minScore;
      }

      const response = await ApiService.search(params);
      results = response.results || [];
      totalResults = response.total_results || response.total || 0;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Search failed';
      results = [];
    } finally {
      loading = false;
    }
  }

  function getConfidenceBadgeVariant(confidence?: string) {
    switch (confidence) {
      case 'high': return 'default';
      case 'medium': return 'secondary';
      case 'low': return 'outline';
      default: return 'outline';
    }
  }

  function formatDate(dateStr?: string) {
    if (!dateStr) return 'N/A';
    return new Date(dateStr).toLocaleDateString();
  }

  $inspect('Search state:', { searchQuery, filters, results: results.length });
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-3xl font-bold">Search SRA Database</h1>
    <p class="text-muted-foreground mt-2">
      Search across millions of sequencing studies, samples, and experiments
    </p>
  </div>

  <Card.Root>
    <Card.Header>
      <Card.Title>Search Parameters</Card.Title>
    </Card.Header>
    <Card.Content class="space-y-4">
      <div class="flex gap-2">
        <div class="flex-1">
          <Input
            type="text"
            placeholder="Enter search terms (e.g., 'RNA-Seq human cancer')"
            bind:value={searchQuery}
            onkeydown={(e) => e.key === 'Enter' && handleSearch()}
          />
        </div>
        <Button onclick={() => handleSearch()} disabled={loading}>
          <Search class="mr-2 h-4 w-4" />
          Search
        </Button>
      </div>

      <div class="grid gap-4 md:grid-cols-4">
        <div>
          <Label for="library-strategy">Library Strategy</Label>
          <Select.Root bind:value={filters.libraryStrategy}>
            <Select.Trigger id="library-strategy">
              <Select.Value placeholder="Any" />
            </Select.Trigger>
            <Select.Content>
              <Select.Item value="">Any</Select.Item>
              <Select.Item value="RNA-Seq">RNA-Seq</Select.Item>
              <Select.Item value="WGS">WGS</Select.Item>
              <Select.Item value="WXS">WXS</Select.Item>
              <Select.Item value="ChIP-Seq">ChIP-Seq</Select.Item>
              <Select.Item value="ATAC-Seq">ATAC-Seq</Select.Item>
            </Select.Content>
          </Select.Root>
        </div>

        <div>
          <Label for="platform">Platform</Label>
          <Select.Root bind:value={filters.platform}>
            <Select.Trigger id="platform">
              <Select.Value placeholder="Any" />
            </Select.Trigger>
            <Select.Content>
              <Select.Item value="">Any</Select.Item>
              <Select.Item value="ILLUMINA">Illumina</Select.Item>
              <Select.Item value="OXFORD_NANOPORE">Oxford Nanopore</Select.Item>
              <Select.Item value="PACBIO_SMRT">PacBio</Select.Item>
              <Select.Item value="ION_TORRENT">Ion Torrent</Select.Item>
            </Select.Content>
          </Select.Root>
        </div>

        <div>
          <Label for="organism">Organism</Label>
          <Input
            id="organism"
            type="text"
            placeholder="e.g., Homo sapiens"
            bind:value={filters.organism}
          />
        </div>

        <div>
          <Label for="search-mode">Search Mode</Label>
          <Select.Root bind:value={filters.searchMode}>
            <Select.Trigger id="search-mode">
              <Select.Value />
            </Select.Trigger>
            <Select.Content>
              <Select.Item value="hybrid">Hybrid</Select.Item>
              <Select.Item value="database">Database</Select.Item>
              <Select.Item value="fts">Full-Text</Select.Item>
              <Select.Item value="vector">Semantic</Select.Item>
            </Select.Content>
          </Select.Root>
        </div>
      </div>

      <Button
        variant="outline"
        size="sm"
        onclick={() => showAdvanced = !showAdvanced}
      >
        <Filter class="mr-2 h-4 w-4" />
        {showAdvanced ? 'Hide' : 'Show'} Advanced Options
      </Button>

      {#if showAdvanced}
        <div class="grid gap-4 md:grid-cols-3 pt-4 border-t">
          <div>
            <Label for="similarity">Similarity Threshold ({advancedOptions.similarityThreshold})</Label>
            <input
              id="similarity"
              type="range"
              min="0"
              max="1"
              step="0.05"
              bind:value={advancedOptions.similarityThreshold}
              class="w-full"
            />
          </div>
          <div>
            <Label for="min-score">Minimum Score ({advancedOptions.minScore})</Label>
            <input
              id="min-score"
              type="range"
              min="0"
              max="100"
              step="5"
              bind:value={advancedOptions.minScore}
              class="w-full"
            />
          </div>
          <div class="flex items-center space-x-2">
            <input
              id="show-confidence"
              type="checkbox"
              bind:checked={advancedOptions.showConfidence}
              class="rounded"
            />
            <Label for="show-confidence">Show Confidence Scores</Label>
          </div>
        </div>
      {/if}
    </Card.Content>
  </Card.Root>

  {#if loading}
    <div class="space-y-4">
      {#each Array(3) as _}
        <Card.Root>
          <Card.Header>
            <Skeleton class="h-6 w-3/4" />
            <Skeleton class="h-4 w-1/2 mt-2" />
          </Card.Header>
          <Card.Content>
            <Skeleton class="h-20 w-full" />
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if error}
    <Card.Root class="border-destructive">
      <Card.Header>
        <Card.Title class="text-destructive">Search Error</Card.Title>
      </Card.Header>
      <Card.Content>
        <p class="text-sm">{error}</p>
      </Card.Content>
    </Card.Root>
  {:else if results.length > 0}
    <div class="space-y-4">
      <div class="flex justify-between items-center">
        <p class="text-sm text-muted-foreground">
          Found {totalResults} results (showing {results.length})
        </p>
        {#if totalResults > itemsPerPage}
          <div class="flex gap-2">
            <Button
              size="sm"
              variant="outline"
              disabled={currentPage === 1}
              onclick={() => { currentPage--; handleSearch(false); }}
            >
              Previous
            </Button>
            <span class="px-3 py-1 text-sm">
              Page {currentPage} of {Math.ceil(totalResults / itemsPerPage)}
            </span>
            <Button
              size="sm"
              variant="outline"
              disabled={currentPage >= Math.ceil(totalResults / itemsPerPage)}
              onclick={() => { currentPage++; handleSearch(false); }}
            >
              Next
            </Button>
          </div>
        {/if}
      </div>

      {#each results as result}
        <Card.Root>
          <Card.Header>
            <div class="flex justify-between items-start">
              <div class="flex-1">
                <Card.Title class="text-lg">
                  {result.title || result.id}
                </Card.Title>
                <div class="flex gap-2 mt-2 flex-wrap">
                  <Badge variant="outline">{result.type}</Badge>
                  {#if result.organism}
                    <Badge variant="secondary">{result.organism}</Badge>
                  {/if}
                  {#if result.library_strategy}
                    <Badge variant="secondary">{result.library_strategy}</Badge>
                  {/if}
                  {#if result.platform}
                    <Badge variant="secondary">{result.platform}</Badge>
                  {/if}
                  {#if advancedOptions.showConfidence && result.confidence}
                    <Badge variant={getConfidenceBadgeVariant(result.confidence)}>
                      {result.confidence} confidence
                    </Badge>
                  {/if}
                </div>
              </div>
              <Button size="sm" variant="ghost" asChild>
                <a href={`/browse/${result.type}/${result.id}`}>
                  <ChevronRight class="h-4 w-4" />
                </a>
              </Button>
            </div>
          </Card.Header>
          {#if result.abstract}
            <Card.Content>
              <p class="text-sm text-muted-foreground line-clamp-3">
                {result.abstract}
              </p>
              <div class="flex gap-4 mt-4 text-sm text-muted-foreground">
                {#if result.submission_date}
                  <span>Submitted: {formatDate(result.submission_date)}</span>
                {/if}
                {#if result.sample_count}
                  <span>{result.sample_count} samples</span>
                {/if}
                {#if result.run_count}
                  <span>{result.run_count} runs</span>
                {/if}
                {#if result.score}
                  <span>Score: {result.score.toFixed(2)}</span>
                {/if}
              </div>
            </Card.Content>
          {/if}
        </Card.Root>
      {/each}
    </div>
  {:else if searchQuery}
    <Card.Root>
      <Card.Content class="text-center py-8">
        <p class="text-muted-foreground">No results found for your search.</p>
        <p class="text-sm text-muted-foreground mt-2">
          Try adjusting your search terms or filters.
        </p>
      </Card.Content>
    </Card.Root>
  {/if}
</div>