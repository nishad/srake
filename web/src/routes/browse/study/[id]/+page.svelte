<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import * as Tabs from '$lib/components/ui/tabs';
  import type { SearchResult } from '$lib/utils';
  import {
    ArrowLeft,
    ExternalLink,
    Calendar,
    Database,
    FlaskConical,
    FileText,
    Download,
    Hash,
    Globe,
    Dna,
    Activity
  } from 'lucide-svelte';

  let studyId = $state('');
  let study = $state<SearchResult | null>(null);
  let relatedStudies = $state<SearchResult[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Get study ID from URL
  $effect(() => {
    studyId = $page.params.id;
  });

  onMount(() => {
    if (studyId) {
      loadStudyDetails();
    }
  });

  async function loadStudyDetails() {
    loading = true;
    error = null;

    try {
      // Search for the specific study by ID
      const response = await ApiService.search({
        query: studyId,
        limit: 1,
        searchMode: 'database'
      });

      if (response.results && response.results.length > 0) {
        study = response.results[0];

        // Load related studies based on organism or strategy
        if (study.organism || study.library_strategy) {
          const relatedResponse = await ApiService.search({
            query: study.organism || study.library_strategy || '',
            limit: 5,
            searchMode: 'database'
          });

          // Filter out the current study
          relatedStudies = (relatedResponse.results || [])
            .filter(s => s.id !== studyId)
            .slice(0, 4);
        }
      } else {
        error = 'Study not found';
      }
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load study details';
    } finally {
      loading = false;
    }
  }

  function formatDate(dateStr?: string) {
    if (!dateStr) return 'N/A';
    return new Date(dateStr).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric'
    });
  }

  function formatNumber(num?: number): string {
    if (!num) return '0';
    return new Intl.NumberFormat('en-US').format(num);
  }

  function goBack() {
    window.history.back();
  }
</script>

<div class="space-y-6">
  {#if loading}
    <div class="space-y-4">
      <Skeleton class="h-8 w-48" />
      <Card.Root>
        <Card.Header>
          <Skeleton class="h-6 w-3/4" />
          <Skeleton class="h-4 w-1/2 mt-2" />
        </Card.Header>
        <Card.Content>
          <Skeleton class="h-32 w-full" />
        </Card.Content>
      </Card.Root>
    </div>
  {:else if error}
    <div class="space-y-4">
      <Button variant="ghost" onclick={goBack}>
        <ArrowLeft class="h-4 w-4 mr-2" />
        Back to Browse
      </Button>
      <Card.Root class="border-destructive">
        <Card.Header>
          <Card.Title class="text-destructive">Error</Card.Title>
        </Card.Header>
        <Card.Content>
          <p class="text-sm">{error}</p>
        </Card.Content>
      </Card.Root>
    </div>
  {:else if study}
    <div>
      <Button variant="ghost" onclick={goBack} class="mb-4">
        <ArrowLeft class="h-4 w-4 mr-2" />
        Back to Browse
      </Button>

      <div class="flex justify-between items-start mb-6">
        <div>
          <h1 class="text-3xl font-bold">{study.title || study.id}</h1>
          <div class="flex gap-2 mt-3 flex-wrap">
            <Badge variant="outline">
              <Hash class="h-3 w-3 mr-1" />
              {study.id}
            </Badge>
            {#if study.type}
              <Badge variant="secondary">{study.type}</Badge>
            {/if}
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
        <div class="flex gap-2">
          <Button variant="outline" asChild>
            <a
              href={`https://www.ncbi.nlm.nih.gov/sra/${study.id}`}
              target="_blank"
              rel="noopener noreferrer"
            >
              <ExternalLink class="h-4 w-4 mr-2" />
              View on NCBI
            </a>
          </Button>
        </div>
      </div>

      <Tabs.Root value="overview">
        <Tabs.List>
          <Tabs.Trigger value="overview">Overview</Tabs.Trigger>
          <Tabs.Trigger value="metadata">Metadata</Tabs.Trigger>
          <Tabs.Trigger value="related">Related Studies</Tabs.Trigger>
        </Tabs.List>

        <Tabs.Content value="overview" class="space-y-4 mt-6">
          <Card.Root>
            <Card.Header>
              <Card.Title>Study Description</Card.Title>
            </Card.Header>
            <Card.Content>
              <p class="text-sm leading-relaxed">
                {study.abstract || study.description || 'No description available for this study.'}
              </p>
            </Card.Content>
          </Card.Root>

          <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Card.Root>
              <Card.Header class="pb-3">
                <Card.Title class="text-sm font-medium">Submission Date</Card.Title>
              </Card.Header>
              <Card.Content>
                <div class="flex items-center gap-2">
                  <Calendar class="h-4 w-4 text-muted-foreground" />
                  <span class="text-sm">{formatDate(study.submission_date)}</span>
                </div>
              </Card.Content>
            </Card.Root>

            <Card.Root>
              <Card.Header class="pb-3">
                <Card.Title class="text-sm font-medium">Samples</Card.Title>
              </Card.Header>
              <Card.Content>
                <div class="flex items-center gap-2">
                  <FlaskConical class="h-4 w-4 text-muted-foreground" />
                  <span class="text-2xl font-bold">{formatNumber(study.sample_count)}</span>
                </div>
              </Card.Content>
            </Card.Root>

            <Card.Root>
              <Card.Header class="pb-3">
                <Card.Title class="text-sm font-medium">Runs</Card.Title>
              </Card.Header>
              <Card.Content>
                <div class="flex items-center gap-2">
                  <Activity class="h-4 w-4 text-muted-foreground" />
                  <span class="text-2xl font-bold">{formatNumber(study.run_count)}</span>
                </div>
              </Card.Content>
            </Card.Root>

            <Card.Root>
              <Card.Header class="pb-3">
                <Card.Title class="text-sm font-medium">Experiments</Card.Title>
              </Card.Header>
              <Card.Content>
                <div class="flex items-center gap-2">
                  <Dna class="h-4 w-4 text-muted-foreground" />
                  <span class="text-2xl font-bold">{formatNumber(study.experiment_count)}</span>
                </div>
              </Card.Content>
            </Card.Root>
          </div>

          {#if study.keywords || study.tags}
            <Card.Root>
              <Card.Header>
                <Card.Title>Keywords & Tags</Card.Title>
              </Card.Header>
              <Card.Content>
                <div class="flex flex-wrap gap-2">
                  {#each (study.keywords || study.tags || '').split(',') as keyword}
                    {#if keyword.trim()}
                      <Badge variant="outline">{keyword.trim()}</Badge>
                    {/if}
                  {/each}
                </div>
              </Card.Content>
            </Card.Root>
          {/if}
        </Tabs.Content>

        <Tabs.Content value="metadata" class="space-y-4 mt-6">
          <Card.Root>
            <Card.Header>
              <Card.Title>Technical Details</Card.Title>
            </Card.Header>
            <Card.Content>
              <dl class="space-y-3 text-sm">
                <div class="flex justify-between">
                  <dt class="font-medium">Study ID</dt>
                  <dd class="text-muted-foreground">{study.id}</dd>
                </div>
                {#if study.accession}
                  <div class="flex justify-between">
                    <dt class="font-medium">Accession</dt>
                    <dd class="text-muted-foreground">{study.accession}</dd>
                  </div>
                {/if}
                {#if study.organism}
                  <div class="flex justify-between">
                    <dt class="font-medium">Organism</dt>
                    <dd class="text-muted-foreground">{study.organism}</dd>
                  </div>
                {/if}
                {#if study.library_strategy}
                  <div class="flex justify-between">
                    <dt class="font-medium">Library Strategy</dt>
                    <dd class="text-muted-foreground">{study.library_strategy}</dd>
                  </div>
                {/if}
                {#if study.library_source}
                  <div class="flex justify-between">
                    <dt class="font-medium">Library Source</dt>
                    <dd class="text-muted-foreground">{study.library_source}</dd>
                  </div>
                {/if}
                {#if study.library_selection}
                  <div class="flex justify-between">
                    <dt class="font-medium">Library Selection</dt>
                    <dd class="text-muted-foreground">{study.library_selection}</dd>
                  </div>
                {/if}
                {#if study.platform}
                  <div class="flex justify-between">
                    <dt class="font-medium">Platform</dt>
                    <dd class="text-muted-foreground">{study.platform}</dd>
                  </div>
                {/if}
                {#if study.instrument_model}
                  <div class="flex justify-between">
                    <dt class="font-medium">Instrument Model</dt>
                    <dd class="text-muted-foreground">{study.instrument_model}</dd>
                  </div>
                {/if}
                {#if study.submission_date}
                  <div class="flex justify-between">
                    <dt class="font-medium">Submission Date</dt>
                    <dd class="text-muted-foreground">{formatDate(study.submission_date)}</dd>
                  </div>
                {/if}
                {#if study.update_date}
                  <div class="flex justify-between">
                    <dt class="font-medium">Last Updated</dt>
                    <dd class="text-muted-foreground">{formatDate(study.update_date)}</dd>
                  </div>
                {/if}
              </dl>
            </Card.Content>
          </Card.Root>

          <Card.Root>
            <Card.Header>
              <Card.Title>Data Statistics</Card.Title>
            </Card.Header>
            <Card.Content>
              <dl class="space-y-3 text-sm">
                <div class="flex justify-between">
                  <dt class="font-medium">Total Samples</dt>
                  <dd class="text-muted-foreground">{formatNumber(study.sample_count)}</dd>
                </div>
                <div class="flex justify-between">
                  <dt class="font-medium">Total Runs</dt>
                  <dd class="text-muted-foreground">{formatNumber(study.run_count)}</dd>
                </div>
                <div class="flex justify-between">
                  <dt class="font-medium">Total Experiments</dt>
                  <dd class="text-muted-foreground">{formatNumber(study.experiment_count)}</dd>
                </div>
                {#if study.total_bases}
                  <div class="flex justify-between">
                    <dt class="font-medium">Total Bases</dt>
                    <dd class="text-muted-foreground">{formatNumber(study.total_bases)}</dd>
                  </div>
                {/if}
                {#if study.total_size}
                  <div class="flex justify-between">
                    <dt class="font-medium">Total Size</dt>
                    <dd class="text-muted-foreground">{study.total_size}</dd>
                  </div>
                {/if}
              </dl>
            </Card.Content>
          </Card.Root>
        </Tabs.Content>

        <Tabs.Content value="related" class="space-y-4 mt-6">
          {#if relatedStudies.length > 0}
            <div class="grid gap-4 md:grid-cols-2">
              {#each relatedStudies as relatedStudy}
                <Card.Root class="hover:shadow-md transition-shadow cursor-pointer"
                          onclick={() => window.location.href = `/browse/study/${relatedStudy.id}`}>
                  <Card.Header>
                    <Card.Title class="text-base line-clamp-1">
                      {relatedStudy.title || relatedStudy.id}
                    </Card.Title>
                    <div class="flex gap-2 mt-2">
                      {#if relatedStudy.organism}
                        <Badge variant="outline" class="text-xs">{relatedStudy.organism}</Badge>
                      {/if}
                      {#if relatedStudy.library_strategy}
                        <Badge variant="outline" class="text-xs">{relatedStudy.library_strategy}</Badge>
                      {/if}
                    </div>
                  </Card.Header>
                  {#if relatedStudy.abstract}
                    <Card.Content>
                      <p class="text-sm text-muted-foreground line-clamp-2">
                        {relatedStudy.abstract}
                      </p>
                    </Card.Content>
                  {/if}
                </Card.Root>
              {/each}
            </div>
          {:else}
            <Card.Root>
              <Card.Content class="text-center py-8">
                <p class="text-muted-foreground">No related studies found.</p>
              </Card.Content>
            </Card.Root>
          {/if}
        </Tabs.Content>
      </Tabs.Root>
    </div>
  {/if}
</div>