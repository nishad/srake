<script lang="ts">
  import { page } from '$app/state';
  import { ApiService } from '$lib/api';
  import type { Study, Experiment, Sample, Run, SearchResult } from '$lib/utils';
  import { formatDate, formatDateLong, formatNumber, formatBytes, tryParseJSON } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Separator } from '$lib/components/ui/separator';
  import * as Tabs from '$lib/components/ui/tabs';
  import * as Table from '$lib/components/ui/table';
  import {
    ArrowLeft,
    ExternalLink,
    Calendar,
    Hash,
    FlaskConical,
    Dna,
    Activity,
    FileText,
    Sparkles,
    Database
  } from 'lucide-svelte';

  let study = $state<Study | null>(null);
  let experiments = $state<Experiment[]>([]);
  let samples = $state<Sample[]>([]);
  let runs = $state<Run[]>([]);
  let similarStudies = $state<SearchResult[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let activeTab = $state('overview');

  let studyId = $derived(page.params.id);

  // Re-load when the route param changes (client-side navigation)
  $effect(() => {
    const id = studyId;
    if (id) loadStudy(id);
  });

  async function loadStudy(id: string) {
    loading = true;
    error = null;
    activeTab = 'overview';
    try {
      study = await ApiService.getStudy(id);

      // Load related data in parallel
      const [expRes, sampleRes, runRes] = await Promise.all([
        ApiService.getStudyExperiments(id).catch(() => ({ experiments: [], total: 0, study_accession: id })),
        ApiService.getStudySamples(id).catch(() => ({ samples: [], total: 0, study_accession: id })),
        ApiService.getStudyRuns(id).catch(() => ({ runs: [], total: 0, limit: 100, study_accession: id })),
      ]);
      experiments = expRes.experiments || [];
      samples = sampleRes.samples || [];
      runs = runRes.runs || [];

      // Find similar studies via vector search
      const searchTerm = study.study_title || study.organism || id;
      try {
        const similar = await ApiService.search({
          query: searchTerm,
          searchMode: 'vector',
          limit: 6,
          showConfidence: true
        });
        similarStudies = (similar.results || []).filter(r => r.id !== id).slice(0, 5);
      } catch {
        similarStudies = [];
      }
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load study';
    } finally {
      loading = false;
    }
  }
</script>

<div class="space-y-6">
  {#if loading}
    <Skeleton class="h-8 w-48" />
    <Card.Root>
      <Card.Header><Skeleton class="h-6 w-3/4" /></Card.Header>
      <Card.Content><Skeleton class="h-32 w-full" /></Card.Content>
    </Card.Root>
  {:else if error}
    <Button variant="ghost" onclick={() => history.back()}>
      <ArrowLeft class="h-4 w-4 mr-2" /> Back
    </Button>
    <Card.Root class="border-destructive">
      <Card.Header><Card.Title class="text-destructive">Error</Card.Title></Card.Header>
      <Card.Content><p class="text-sm">{error}</p></Card.Content>
    </Card.Root>
  {:else if study}
    <!-- Back button -->
    <Button variant="ghost" size="sm" onclick={() => history.back()}>
      <ArrowLeft class="h-4 w-4 mr-1" /> Back
    </Button>

    <!-- Title section -->
    <div>
      <h1 class="text-2xl font-bold tracking-tight">{study.study_title || study.study_accession}</h1>
      <div class="flex gap-1.5 mt-3 flex-wrap">
        <Badge variant="outline" class="font-mono text-xs">
          <Hash class="h-3 w-3 mr-1" />{study.study_accession}
        </Badge>
        {#if study.study_type}
          <Badge variant="secondary" class="text-xs">{study.study_type}</Badge>
        {/if}
        {#if study.organism}
          <Badge variant="secondary" class="text-xs">
            <Dna class="h-3 w-3 mr-1" />{study.organism}
          </Badge>
        {/if}
        {#if study.center_name}
          <Badge variant="outline" class="text-xs">{study.center_name}</Badge>
        {/if}
      </div>
      <div class="flex gap-2 mt-3">
        <Button variant="outline" size="sm" href="https://www.ncbi.nlm.nih.gov/sra/{study.study_accession}" target="_blank">
          <ExternalLink class="h-3 w-3 mr-1.5" /> NCBI SRA
        </Button>
        {#if study.primary_id}
          <Button variant="outline" size="sm" href="https://www.ncbi.nlm.nih.gov/bioproject/{study.primary_id}" target="_blank">
            <ExternalLink class="h-3 w-3 mr-1.5" /> BioProject
          </Button>
        {/if}
      </div>
    </div>

    <!-- Tabs -->
    <Tabs.Root bind:value={activeTab}>
      <Tabs.List>
        <Tabs.Trigger value="overview">Overview</Tabs.Trigger>
        <Tabs.Trigger value="experiments">
          Experiments
          {#if experiments.length > 0}
            <Badge variant="secondary" class="ml-1.5 text-xs">{experiments.length}</Badge>
          {/if}
        </Tabs.Trigger>
        <Tabs.Trigger value="samples">
          Samples
          {#if samples.length > 0}
            <Badge variant="secondary" class="ml-1.5 text-xs">{samples.length}</Badge>
          {/if}
        </Tabs.Trigger>
        <Tabs.Trigger value="runs">
          Runs
          {#if runs.length > 0}
            <Badge variant="secondary" class="ml-1.5 text-xs">{runs.length}</Badge>
          {/if}
        </Tabs.Trigger>
        <Tabs.Trigger value="similar">
          Similar
          {#if similarStudies.length > 0}
            <Badge variant="secondary" class="ml-1.5 text-xs">{similarStudies.length}</Badge>
          {/if}
        </Tabs.Trigger>
      </Tabs.List>

      <!-- Overview -->
      <Tabs.Content value="overview" class="space-y-4 mt-4">
        {#if study.study_abstract}
          <Card.Root>
            <Card.Header><Card.Title class="text-base">Abstract</Card.Title></Card.Header>
            <Card.Content>
              <p class="text-sm leading-relaxed">{study.study_abstract}</p>
            </Card.Content>
          </Card.Root>
        {/if}

        {#if study.study_description && study.study_description !== study.study_abstract}
          <Card.Root>
            <Card.Header><Card.Title class="text-base">Description</Card.Title></Card.Header>
            <Card.Content>
              <p class="text-sm leading-relaxed">{study.study_description}</p>
            </Card.Content>
          </Card.Root>
        {/if}

        <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <Card.Root>
            <Card.Header class="pb-2"><Card.Title class="text-sm font-medium">Experiments</Card.Title></Card.Header>
            <Card.Content>
              <span class="text-2xl font-bold">{experiments.length}</span>
            </Card.Content>
          </Card.Root>
          <Card.Root>
            <Card.Header class="pb-2"><Card.Title class="text-sm font-medium">Samples</Card.Title></Card.Header>
            <Card.Content>
              <span class="text-2xl font-bold">{samples.length}</span>
            </Card.Content>
          </Card.Root>
          <Card.Root>
            <Card.Header class="pb-2"><Card.Title class="text-sm font-medium">Runs</Card.Title></Card.Header>
            <Card.Content>
              <span class="text-2xl font-bold">{runs.length}</span>
            </Card.Content>
          </Card.Root>
          <Card.Root>
            <Card.Header class="pb-2"><Card.Title class="text-sm font-medium">Submitted</Card.Title></Card.Header>
            <Card.Content>
              <span class="text-sm">{formatDateLong(study.submission_date)}</span>
            </Card.Content>
          </Card.Root>
        </div>

        <!-- Metadata details -->
        <Card.Root>
          <Card.Header><Card.Title class="text-base">Details</Card.Title></Card.Header>
          <Card.Content>
            <dl class="grid gap-2 text-sm md:grid-cols-2">
              {#if study.center_name}
                <div class="flex justify-between gap-2"><dt class="font-medium text-muted-foreground">Center</dt><dd>{study.center_name}</dd></div>
              {/if}
              {#if study.broker_name}
                <div class="flex justify-between gap-2"><dt class="font-medium text-muted-foreground">Broker</dt><dd>{study.broker_name}</dd></div>
              {/if}
              {#if study.first_public}
                <div class="flex justify-between gap-2"><dt class="font-medium text-muted-foreground">First Public</dt><dd>{formatDate(study.first_public)}</dd></div>
              {/if}
              {#if study.last_update}
                <div class="flex justify-between gap-2"><dt class="font-medium text-muted-foreground">Last Updated</dt><dd>{formatDate(study.last_update)}</dd></div>
              {/if}
              {#if study.alias}
                <div class="flex justify-between gap-2"><dt class="font-medium text-muted-foreground">Alias</dt><dd>{study.alias}</dd></div>
              {/if}
              {#if study.center_project_name}
                <div class="flex justify-between gap-2"><dt class="font-medium text-muted-foreground">Project Name</dt><dd>{study.center_project_name}</dd></div>
              {/if}
            </dl>
          </Card.Content>
        </Card.Root>
      </Tabs.Content>

      <!-- Experiments -->
      <Tabs.Content value="experiments" class="mt-4">
        {#if experiments.length > 0}
          <Card.Root>
            <Card.Content class="pt-6">
              <Table.Root>
                <Table.Header>
                  <Table.Row>
                    <Table.Head>Accession</Table.Head>
                    <Table.Head>Title</Table.Head>
                    <Table.Head>Platform</Table.Head>
                    <Table.Head>Strategy</Table.Head>
                    <Table.Head>Layout</Table.Head>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {#each experiments as exp}
                    <Table.Row>
                      <Table.Cell>
                        <a href="/browse/experiment/{exp.experiment_accession}" class="text-primary hover:underline font-mono text-xs">
                          {exp.experiment_accession}
                        </a>
                      </Table.Cell>
                      <Table.Cell class="max-w-[300px] truncate">{exp.title || '-'}</Table.Cell>
                      <Table.Cell><Badge variant="outline" class="text-xs">{exp.platform || '-'}</Badge></Table.Cell>
                      <Table.Cell><Badge variant="secondary" class="text-xs">{exp.library_strategy || '-'}</Badge></Table.Cell>
                      <Table.Cell class="text-xs">{exp.library_layout || '-'}</Table.Cell>
                    </Table.Row>
                  {/each}
                </Table.Body>
              </Table.Root>
            </Card.Content>
          </Card.Root>
        {:else}
          <Card.Root>
            <Card.Content class="text-center py-8 text-muted-foreground">No experiments found.</Card.Content>
          </Card.Root>
        {/if}
      </Tabs.Content>

      <!-- Samples -->
      <Tabs.Content value="samples" class="mt-4">
        {#if samples.length > 0}
          <Card.Root>
            <Card.Content class="pt-6">
              <Table.Root>
                <Table.Header>
                  <Table.Row>
                    <Table.Head>Accession</Table.Head>
                    <Table.Head>Title</Table.Head>
                    <Table.Head>Organism</Table.Head>
                    <Table.Head>Tissue</Table.Head>
                    <Table.Head>Disease</Table.Head>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {#each samples as sample}
                    <Table.Row>
                      <Table.Cell>
                        <a href="/browse/sample/{sample.sample_accession}" class="text-primary hover:underline font-mono text-xs">
                          {sample.sample_accession}
                        </a>
                      </Table.Cell>
                      <Table.Cell class="max-w-[300px] truncate">{sample.title || '-'}</Table.Cell>
                      <Table.Cell><Badge variant="secondary" class="text-xs">{sample.scientific_name || sample.organism || '-'}</Badge></Table.Cell>
                      <Table.Cell class="text-xs">{sample.tissue || '-'}</Table.Cell>
                      <Table.Cell class="text-xs">{sample.disease || '-'}</Table.Cell>
                    </Table.Row>
                  {/each}
                </Table.Body>
              </Table.Root>
            </Card.Content>
          </Card.Root>
        {:else}
          <Card.Root>
            <Card.Content class="text-center py-8 text-muted-foreground">No samples found.</Card.Content>
          </Card.Root>
        {/if}
      </Tabs.Content>

      <!-- Runs -->
      <Tabs.Content value="runs" class="mt-4">
        {#if runs.length > 0}
          <Card.Root>
            <Card.Content class="pt-6">
              <Table.Root>
                <Table.Header>
                  <Table.Row>
                    <Table.Head>Accession</Table.Head>
                    <Table.Head>Title</Table.Head>
                    <Table.Head class="text-right">Spots</Table.Head>
                    <Table.Head class="text-right">Bases</Table.Head>
                    <Table.Head class="text-right">Size</Table.Head>
                  </Table.Row>
                </Table.Header>
                <Table.Body>
                  {#each runs as run}
                    <Table.Row>
                      <Table.Cell>
                        <a href="/browse/run/{run.run_accession}" class="text-primary hover:underline font-mono text-xs">
                          {run.run_accession}
                        </a>
                      </Table.Cell>
                      <Table.Cell class="max-w-[300px] truncate">{run.title || '-'}</Table.Cell>
                      <Table.Cell class="text-right text-xs">{run.total_spots ? formatNumber(run.total_spots) : '-'}</Table.Cell>
                      <Table.Cell class="text-right text-xs">{run.total_bases ? formatNumber(run.total_bases) : '-'}</Table.Cell>
                      <Table.Cell class="text-right text-xs">{run.total_size ? formatBytes(run.total_size) : '-'}</Table.Cell>
                    </Table.Row>
                  {/each}
                </Table.Body>
              </Table.Root>
            </Card.Content>
          </Card.Root>
        {:else}
          <Card.Root>
            <Card.Content class="text-center py-8 text-muted-foreground">No runs found.</Card.Content>
          </Card.Root>
        {/if}
      </Tabs.Content>

      <!-- Similar Studies (Vector Search) -->
      <Tabs.Content value="similar" class="mt-4">
        {#if similarStudies.length > 0}
          <div class="space-y-3">
            <p class="text-sm text-muted-foreground flex items-center gap-1.5">
              <Sparkles class="h-4 w-4" />
              Studies found via semantic similarity search
            </p>
            {#each similarStudies as similar}
              <a href="/browse/study/{similar.id}" class="block group">
                <Card.Root class="transition-colors group-hover:bg-accent/50">
                  <Card.Header class="pb-2">
                    <Card.Title class="text-base line-clamp-1">{similar.title || similar.id}</Card.Title>
                    <div class="flex gap-1.5 flex-wrap mt-1">
                      <Badge variant="outline" class="text-xs font-mono">{similar.id}</Badge>
                      {#if similar.organism}
                        <Badge variant="secondary" class="text-xs">{similar.organism}</Badge>
                      {/if}
                      {#if similar.similarity && similar.similarity > 0}
                        <Badge variant="outline" class="text-xs gap-1">
                          <Sparkles class="h-2.5 w-2.5" />
                          {(similar.similarity * 100).toFixed(0)}% similar
                        </Badge>
                      {/if}
                      {#if similar.confidence}
                        <Badge variant="outline" class="text-xs">{similar.confidence}</Badge>
                      {/if}
                    </div>
                  </Card.Header>
                  {#if similar.description || similar.abstract}
                    <Card.Content class="pt-0">
                      <p class="text-sm text-muted-foreground line-clamp-2">{similar.description || similar.abstract}</p>
                    </Card.Content>
                  {/if}
                </Card.Root>
              </a>
            {/each}
          </div>
        {:else}
          <Card.Root>
            <Card.Content class="text-center py-8 text-muted-foreground">
              <Sparkles class="h-8 w-8 mx-auto mb-3 opacity-50" />
              No similar studies found via vector search.
            </Card.Content>
          </Card.Root>
        {/if}
      </Tabs.Content>
    </Tabs.Root>
  {/if}
</div>
