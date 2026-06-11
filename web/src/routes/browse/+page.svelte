<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import type { Study } from '$lib/utils';
  import { formatDate, formatNumber } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import {
    ChevronLeft,
    ChevronRight,
    ExternalLink,
    FlaskConical,
    Database,
    Calendar
  } from 'lucide-svelte';

  let studies = $state<Study[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let currentPage = $state(1);
  let itemsPerPage = 20;
  let hasMore = $state(true);

  onMount(() => {
    loadStudies();
  });

  async function loadStudies() {
    loading = true;
    error = null;
    try {
      const response = await ApiService.listStudies(itemsPerPage, (currentPage - 1) * itemsPerPage);
      studies = response.studies || [];
      hasMore = studies.length === itemsPerPage;
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load studies';
      studies = [];
    } finally {
      loading = false;
    }
  }

  function goToPage(page: number) {
    if (page < 1) return;
    currentPage = page;
    loadStudies();
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-3xl font-bold tracking-tight">Browse Studies</h1>
    <p class="text-muted-foreground mt-1">
      Explore SRA studies and their associated metadata
    </p>
  </div>

  {#if loading}
    <div class="space-y-3">
      {#each Array(5) as _}
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
        <Card.Title class="text-destructive">Error</Card.Title>
      </Card.Header>
      <Card.Content>
        <p class="text-sm">{error}</p>
        <Button variant="outline" class="mt-3" onclick={loadStudies}>Retry</Button>
      </Card.Content>
    </Card.Root>
  {:else if studies.length > 0}
    <div class="space-y-3">
      {#each studies as study}
        <a href="/browse/study/{study.study_accession}" class="block group">
          <div class="rounded-lg border bg-card transition-colors group-hover:bg-accent/50 px-4 py-3">
            <div class="flex justify-between items-start gap-3">
              <div class="flex-1 min-w-0">
                <span class="font-medium text-sm line-clamp-1">
                  {study.study_title || study.study_accession}
                </span>
                <div class="flex gap-1.5 mt-1.5 flex-wrap items-center">
                  <span class="text-xs font-mono text-muted-foreground">{study.study_accession}</span>
                  {#if study.study_type}
                    <Badge variant="secondary" class="text-[10px] px-1.5 py-0">{study.study_type}</Badge>
                  {/if}
                  {#if study.organism}
                    <Badge variant="secondary" class="text-[10px] px-1.5 py-0">{study.organism}</Badge>
                  {/if}
                </div>
                {#if study.study_abstract}
                  <p class="text-xs text-muted-foreground line-clamp-2 mt-1.5">
                    {study.study_abstract}
                  </p>
                {/if}
              </div>
              <ChevronRight class="h-4 w-4 text-muted-foreground shrink-0 mt-1 group-hover:translate-x-0.5 transition-transform" />
            </div>
          </div>
        </a>
      {/each}

      <!-- Pagination -->
      <div class="flex justify-center items-center gap-2 pt-4">
        <Button
          size="sm"
          variant="outline"
          disabled={currentPage <= 1}
          onclick={() => goToPage(currentPage - 1)}
        >
          <ChevronLeft class="h-4 w-4" />
          Previous
        </Button>
        <span class="px-3 text-sm text-muted-foreground tabular-nums">
          Page {currentPage}
        </span>
        <Button
          size="sm"
          variant="outline"
          disabled={!hasMore}
          onclick={() => goToPage(currentPage + 1)}
        >
          Next
          <ChevronRight class="h-4 w-4" />
        </Button>
      </div>
    </div>
  {:else}
    <Card.Root>
      <Card.Content class="text-center py-12">
        <Database class="h-12 w-12 text-muted-foreground mx-auto mb-4" />
        <p class="text-lg font-medium">No studies found</p>
        <p class="text-sm text-muted-foreground mt-1">
          The database may be empty. Try ingesting data first.
        </p>
      </Card.Content>
    </Card.Root>
  {/if}
</div>
