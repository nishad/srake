<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import type { SearchResult } from '$lib/utils';
  import {
    ChevronLeft,
    ChevronRight,
    Calendar,
    Database,
    FlaskConical,
    ExternalLink
  } from 'lucide-svelte';

  let studies = $state<SearchResult[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let currentPage = $state(1);
  let itemsPerPage = 10;
  let totalResults = $state(0);
  let totalPages = $state(0);

  onMount(() => {
    loadStudies();
  });

  async function loadStudies() {
    loading = true;
    error = null;

    try {
      // Use search API to get recent studies
      const response = await ApiService.search({
        query: '',  // Empty query to get all
        limit: itemsPerPage,
        offset: (currentPage - 1) * itemsPerPage,
        searchMode: 'database'
      });

      studies = response.results || [];
      totalResults = response.total_results || response.total || 0;
      totalPages = Math.ceil(totalResults / itemsPerPage);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load studies';
      studies = [];
    } finally {
      loading = false;
    }
  }

  function formatDate(dateStr?: string) {
    if (!dateStr) return 'N/A';
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric'
    });
  }

  function formatNumber(num: number): string {
    return new Intl.NumberFormat('en-US').format(num);
  }

  function goToPage(page: number) {
    if (page < 1 || page > totalPages) return;
    currentPage = page;
    loadStudies();
  }

  function getPageNumbers(): (number | string)[] {
    const pages: (number | string)[] = [];
    const maxVisible = 10;

    if (totalPages <= maxVisible) {
      for (let i = 1; i <= totalPages; i++) {
        pages.push(i);
      }
    } else {
      // Always show first page
      pages.push(1);

      // Calculate range around current page
      let start = Math.max(2, currentPage - 3);
      let end = Math.min(totalPages - 1, currentPage + 3);

      // Add ellipsis if needed at the start
      if (start > 2) {
        pages.push('...');
      }

      // Add pages around current
      for (let i = start; i <= end; i++) {
        pages.push(i);
      }

      // Add ellipsis if needed at the end
      if (end < totalPages - 1) {
        pages.push('...');
      }

      // Always show last page
      if (totalPages > 1) {
        pages.push(totalPages);
      }
    }

    return pages;
  }

  function viewStudyDetails(study: SearchResult) {
    // Navigate to study detail page
    window.location.href = `/browse/study/${study.id}`;
  }
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-3xl font-bold">Browse Studies</h1>
    <p class="text-muted-foreground mt-2">
      Explore recent SRA studies and datasets
    </p>
  </div>

  {#if loading}
    <div class="space-y-4">
      {#each Array(5) as _}
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
        <Card.Title class="text-destructive">Error Loading Studies</Card.Title>
      </Card.Header>
      <Card.Content>
        <p class="text-sm">{error}</p>
      </Card.Content>
    </Card.Root>
  {:else if studies.length > 0}
    <div class="space-y-4">
      <!-- Results summary -->
      <div class="flex justify-between items-center">
        <p class="text-sm text-muted-foreground">
          Showing {((currentPage - 1) * itemsPerPage) + 1}-{Math.min(currentPage * itemsPerPage, totalResults)} of {formatNumber(totalResults)} studies
        </p>
        <div class="text-sm text-muted-foreground">
          Page {currentPage} of {totalPages}
        </div>
      </div>

      <!-- Study cards -->
      {#each studies as study}
        <Card.Root class="hover:shadow-lg transition-shadow">
          <Card.Header>
            <div class="flex justify-between items-start">
              <div class="flex-1">
                <Card.Title class="text-lg flex items-center gap-2">
                  {study.title || study.id}
                  <Badge variant="outline">{study.type || 'Study'}</Badge>
                </Card.Title>
                <div class="flex gap-2 mt-2 flex-wrap">
                  {#if study.organism}
                    <Badge variant="secondary">
                      <FlaskConical class="h-3 w-3 mr-1" />
                      {study.organism}
                    </Badge>
                  {/if}
                  {#if study.library_strategy}
                    <Badge variant="secondary">
                      <Database class="h-3 w-3 mr-1" />
                      {study.library_strategy}
                    </Badge>
                  {/if}
                  {#if study.platform}
                    <Badge variant="secondary">{study.platform}</Badge>
                  {/if}
                </div>
              </div>
              <Button size="sm" variant="ghost" onclick={() => viewStudyDetails(study)}>
                View Details
                <ChevronRight class="h-4 w-4 ml-1" />
              </Button>
            </div>
          </Card.Header>
          {#if study.abstract || study.description}
            <Card.Content>
              <p class="text-sm text-muted-foreground line-clamp-3">
                {study.abstract || study.description}
              </p>
              <div class="flex gap-4 mt-4 text-xs text-muted-foreground">
                <span class="flex items-center gap-1">
                  <Calendar class="h-3 w-3" />
                  Submitted: {formatDate(study.submission_date)}
                </span>
                {#if study.sample_count}
                  <span>{formatNumber(study.sample_count)} samples</span>
                {/if}
                {#if study.run_count}
                  <span>{formatNumber(study.run_count)} runs</span>
                {/if}
                {#if study.experiment_count}
                  <span>{formatNumber(study.experiment_count)} experiments</span>
                {/if}
              </div>
              {#if study.external_links}
                <div class="mt-3 flex gap-2">
                  <a
                    href={`https://www.ncbi.nlm.nih.gov/sra/${study.id}`}
                    target="_blank"
                    rel="noopener noreferrer"
                    class="inline-flex items-center text-xs text-primary hover:underline"
                  >
                    <ExternalLink class="h-3 w-3 mr-1" />
                    View on NCBI
                  </a>
                </div>
              {/if}
            </Card.Content>
          {/if}
        </Card.Root>
      {/each}

      <!-- Pagination controls -->
      <div class="flex justify-center items-center gap-1 mt-8">
        <Button
          size="sm"
          variant="outline"
          disabled={currentPage === 1}
          onclick={() => goToPage(currentPage - 1)}
        >
          <ChevronLeft class="h-4 w-4" />
          Previous
        </Button>

        <div class="flex gap-1 mx-2">
          {#each getPageNumbers() as page}
            {#if typeof page === 'number'}
              <Button
                size="sm"
                variant={page === currentPage ? 'default' : 'outline'}
                onclick={() => goToPage(page)}
                class="min-w-[2.5rem]"
              >
                {page}
              </Button>
            {:else}
              <span class="px-2 py-1 text-sm text-muted-foreground">...</span>
            {/if}
          {/each}
        </div>

        <Button
          size="sm"
          variant="outline"
          disabled={currentPage === totalPages}
          onclick={() => goToPage(currentPage + 1)}
        >
          Next
          <ChevronRight class="h-4 w-4" />
        </Button>
      </div>
    </div>
  {:else}
    <Card.Root>
      <Card.Content class="text-center py-8">
        <p class="text-muted-foreground">No studies found.</p>
      </Card.Content>
    </Card.Root>
  {/if}
</div>