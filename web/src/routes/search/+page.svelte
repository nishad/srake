<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { page } from '$app/stores';
  import { ApiService } from '$lib/api';
  import type { SearchResult, SearchParams } from '$lib/utils';
  import { formatDate } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Input } from '$lib/components/ui/input';
  import { Button } from '$lib/components/ui/button';
  import { Label } from '$lib/components/ui/label';
  import * as Select from '$lib/components/ui/select';
  import { Badge } from '$lib/components/ui/badge';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Separator } from '$lib/components/ui/separator';
  import {
    Search,
    Filter,
    ChevronRight,
    ChevronLeft,
    Sparkles,
    Zap,
    Database as DatabaseIcon,
    FileText,
    X
  } from 'lucide-svelte';

  let searchQuery = $state('');
  let results = $state<SearchResult[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let totalResults = $state(0);
  let timeTaken = $state(0);
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

  const searchModes = [
    { value: 'hybrid', label: 'Hybrid', description: 'Combined text + semantic', icon: Zap },
    { value: 'fts', label: 'Full-Text', description: 'FTS5 keyword search', icon: FileText },
    { value: 'vector', label: 'Semantic', description: 'Vector similarity search', icon: Sparkles },
    { value: 'database', label: 'Database', description: 'Direct SQL queries', icon: DatabaseIcon },
  ];

  // Parse URL parameters on mount
  onMount(() => {
    const q = $page.url.searchParams;
    if (q.get('q')) searchQuery = q.get('q')!;
    if (q.get('platform')) filters.platform = q.get('platform')!;
    if (q.get('library_strategy')) filters.libraryStrategy = q.get('library_strategy')!;
    if (q.get('organism')) filters.organism = q.get('organism')!;
    if (q.get('mode')) filters.searchMode = q.get('mode') as any;
    if (q.get('page')) currentPage = parseInt(q.get('page')!);

    if (searchQuery || filters.platform || filters.libraryStrategy || filters.organism) {
      handleSearch(false);
    }
  });

  function updateURL() {
    const params = new URLSearchParams();
    if (searchQuery) params.set('q', searchQuery);
    if (filters.platform) params.set('platform', filters.platform);
    if (filters.libraryStrategy) params.set('library_strategy', filters.libraryStrategy);
    if (filters.organism) params.set('organism', filters.organism);
    if (filters.searchMode !== 'hybrid') params.set('mode', filters.searchMode!);
    if (currentPage > 1) params.set('page', currentPage.toString());
    const qs = params.toString();
    goto(qs ? `/search?${qs}` : '/search', { replaceState: true, noScroll: true });
  }

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
      totalResults = response.total_results || 0;
      timeTaken = response.time_taken_ms || 0;
      updateURL();
    } catch (err) {
      error = err instanceof Error ? err.message : 'Search failed';
      results = [];
    } finally {
      loading = false;
    }
  }

  function clearFilters() {
    filters.libraryStrategy = '';
    filters.platform = '';
    filters.organism = '';
    filters.searchMode = 'hybrid';
  }

  function getConfidenceColor(confidence?: string): string {
    switch (confidence) {
      case 'high': return 'bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200';
      case 'medium': return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-200';
      case 'low': return 'bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-200';
      default: return '';
    }
  }

  function getDetailUrl(result: SearchResult): string {
    const type = result.type?.toLowerCase() || 'study';
    return `/browse/${type}/${result.id}`;
  }

  let totalPages = $derived(Math.ceil(totalResults / itemsPerPage));
  let hasActiveFilters = $derived(
    filters.libraryStrategy !== '' || filters.platform !== '' || filters.organism !== ''
  );
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-3xl font-bold tracking-tight">Search</h1>
    <p class="text-muted-foreground mt-1">
      Search across SRA studies, experiments, samples, and runs
    </p>
  </div>

  <!-- Search form -->
  <Card.Root>
    <Card.Content class="pt-6 space-y-4">
      <!-- Search input row -->
      <div class="flex gap-2">
        <div class="flex-1 relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search studies, experiments, samples... (e.g., 'RNA-Seq human cancer')"
            bind:value={searchQuery}
            onkeydown={(e) => e.key === 'Enter' && handleSearch()}
            class="pl-10"
          />
        </div>
        <Button onclick={() => handleSearch()} disabled={loading}>
          {#if loading}
            <span class="animate-spin mr-2">&#8987;</span>
          {:else}
            <Search class="mr-2 h-4 w-4" />
          {/if}
          Search
        </Button>
      </div>

      <!-- Search mode selector -->
      <div class="flex items-center gap-2 flex-wrap">
        <span class="text-sm text-muted-foreground">Mode:</span>
        {#each searchModes as mode}
          {@const Icon = mode.icon}
          <Button
            size="sm"
            variant={filters.searchMode === mode.value ? 'default' : 'outline'}
            onclick={() => { filters.searchMode = mode.value as any; }}
            class="gap-1.5"
          >
            <Icon class="h-3 w-3" />
            {mode.label}
          </Button>
        {/each}
      </div>

      <!-- Filter toggle row -->
      <div class="flex items-center justify-between">
        <Button
          variant="ghost"
          size="sm"
          onclick={() => showAdvanced = !showAdvanced}
          class="gap-1.5 text-muted-foreground"
        >
          <Filter class="h-3 w-3" />
          {showAdvanced ? 'Hide' : 'Show'} Filters
        </Button>

        {#if hasActiveFilters}
          <Button variant="ghost" size="sm" onclick={clearFilters} class="gap-1 text-muted-foreground">
            <X class="h-3 w-3" />
            Clear filters
          </Button>
        {/if}
      </div>

      {#if showAdvanced}
        <Separator />
        <div class="grid gap-4 md:grid-cols-3">
          <div class="space-y-1">
            <Label class="text-xs">Library Strategy</Label>
            <Input
              type="text"
              placeholder="e.g., RNA-Seq, WGS, ChIP-Seq"
              bind:value={filters.libraryStrategy}
            />
          </div>
          <div class="space-y-1">
            <Label class="text-xs">Platform</Label>
            <Input
              type="text"
              placeholder="e.g., ILLUMINA, OXFORD_NANOPORE"
              bind:value={filters.platform}
            />
          </div>
          <div class="space-y-1">
            <Label class="text-xs">Organism</Label>
            <Input
              type="text"
              placeholder="e.g., Homo sapiens"
              bind:value={filters.organism}
            />
          </div>
        </div>
        <div class="grid gap-4 md:grid-cols-3">
          <div class="space-y-2">
            <Label class="text-xs">Similarity Threshold: {advancedOptions.similarityThreshold}</Label>
            <input
              type="range"
              min="0"
              max="1"
              step="0.05"
              bind:value={advancedOptions.similarityThreshold}
              class="w-full accent-primary"
            />
          </div>
          <div class="space-y-2">
            <Label class="text-xs">Minimum Score: {advancedOptions.minScore}</Label>
            <input
              type="range"
              min="0"
              max="100"
              step="5"
              bind:value={advancedOptions.minScore}
              class="w-full accent-primary"
            />
          </div>
          <div class="flex items-center gap-2 self-end pb-2">
            <input
              id="show-confidence"
              type="checkbox"
              bind:checked={advancedOptions.showConfidence}
              class="rounded accent-primary"
            />
            <Label for="show-confidence" class="text-xs">Show confidence scores</Label>
          </div>
        </div>
      {/if}
    </Card.Content>
  </Card.Root>

  <!-- Results -->
  {#if loading}
    <div class="space-y-3">
      {#each Array(3) as _}
        <Card.Root>
          <Card.Header>
            <Skeleton class="h-5 w-3/4" />
            <Skeleton class="h-4 w-1/2 mt-2" />
          </Card.Header>
          <Card.Content>
            <Skeleton class="h-16 w-full" />
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
      <!-- Results header -->
      <div class="flex justify-between items-center">
        <p class="text-sm text-muted-foreground">
          {totalResults.toLocaleString()} results
          {#if timeTaken > 0}
            <span class="text-xs">({timeTaken}ms)</span>
          {/if}
        </p>
        {#if totalPages > 1}
          <span class="text-sm text-muted-foreground">
            Page {currentPage} of {totalPages}
          </span>
        {/if}
      </div>

      <!-- Result cards -->
      {#each results as result}
        <a href={getDetailUrl(result)} class="block group">
          <div class="rounded-lg border bg-card transition-colors group-hover:bg-accent/50 px-4 py-3">
            <div class="flex justify-between items-start gap-3">
              <div class="flex-1 min-w-0">
                <div class="flex items-center gap-2 min-w-0">
                  <span class="font-medium text-sm line-clamp-1">{result.title || result.id}</span>
                </div>
                <div class="flex gap-1.5 mt-1.5 flex-wrap items-center">
                  <span class="text-xs font-mono text-muted-foreground">{result.id}</span>
                  <Badge variant="outline" class="text-[10px] px-1.5 py-0">{result.type || 'study'}</Badge>
                  {#if result.organism}
                    <Badge variant="secondary" class="text-[10px] px-1.5 py-0">{result.organism}</Badge>
                  {/if}
                  {#if result.library_strategy}
                    <Badge variant="secondary" class="text-[10px] px-1.5 py-0">{result.library_strategy}</Badge>
                  {/if}
                  {#if result.platform}
                    <Badge variant="secondary" class="text-[10px] px-1.5 py-0">{result.platform}</Badge>
                  {/if}
                  {#if result.score}
                    <span class="text-[10px] text-muted-foreground tabular-nums">{result.score.toFixed(2)}</span>
                  {/if}
                  {#if result.similarity && result.similarity > 0}
                    <span class="text-[10px] text-muted-foreground gap-0.5 flex items-center">
                      <Sparkles class="h-2.5 w-2.5" />
                      {(result.similarity * 100).toFixed(0)}%
                    </span>
                  {/if}
                  {#if advancedOptions.showConfidence && result.confidence}
                    <Badge class="text-[10px] px-1.5 py-0 {getConfidenceColor(result.confidence)}">
                      {result.confidence}
                    </Badge>
                  {/if}
                </div>
                {#if result.description || result.abstract}
                  <p class="text-xs text-muted-foreground line-clamp-1 mt-1.5">
                    {result.description || result.abstract}
                  </p>
                {/if}
              </div>
              <ChevronRight class="h-4 w-4 text-muted-foreground shrink-0 mt-1 transition-transform group-hover:translate-x-0.5" />
            </div>
          </div>
        </a>
      {/each}

      <!-- Pagination -->
      {#if totalPages > 1}
        <div class="flex justify-center items-center gap-2 pt-4">
          <Button
            size="sm"
            variant="outline"
            disabled={currentPage <= 1}
            onclick={() => { currentPage--; handleSearch(false); }}
          >
            <ChevronLeft class="h-4 w-4" />
            Previous
          </Button>
          <span class="px-3 text-sm text-muted-foreground">
            {currentPage} / {totalPages}
          </span>
          <Button
            size="sm"
            variant="outline"
            disabled={currentPage >= totalPages}
            onclick={() => { currentPage++; handleSearch(false); }}
          >
            Next
            <ChevronRight class="h-4 w-4" />
          </Button>
        </div>
      {/if}
    </div>
  {:else if searchQuery || hasActiveFilters}
    <Card.Root>
      <Card.Content class="text-center py-12">
        <Search class="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <p class="text-lg font-medium">No results found</p>
        <p class="text-sm text-muted-foreground mt-1">
          Try different search terms, filters, or search mode
        </p>
      </Card.Content>
    </Card.Root>
  {:else}
    <Card.Root>
      <Card.Content class="text-center py-12">
        <Search class="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <p class="text-lg font-medium">Enter a search query</p>
        <p class="text-sm text-muted-foreground mt-1">
          Search by keywords, accession IDs, organism names, or any metadata
        </p>
        <div class="flex flex-wrap justify-center gap-2 mt-4">
          {#each ['RNA-Seq human', 'cancer genomics', 'CRISPR screen', 'metagenomics soil'] as example}
            <Button
              variant="outline"
              size="sm"
              onclick={() => { searchQuery = example; handleSearch(); }}
            >
              {example}
            </Button>
          {/each}
        </div>
      </Card.Content>
    </Card.Root>
  {/if}
</div>
