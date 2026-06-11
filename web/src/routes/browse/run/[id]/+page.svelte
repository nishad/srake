<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import type { Run } from '$lib/utils';
  import { formatNumber, formatBytes, formatDate, tryParseJSON } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import * as Table from '$lib/components/ui/table';
  import {
    ArrowLeft,
    ExternalLink,
    Hash,
    Activity,
    HardDrive,
    FileDown,
    FlaskConical
  } from 'lucide-svelte';

  let accession = $derived($page.params.id);
  let run = $state<Run | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    if (!accession) return;
    try {
      run = await ApiService.getRun(accession);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load run';
    } finally {
      loading = false;
    }
  });

  let dataFiles = $derived(run?.data_files ? tryParseJSON(run.data_files) : null);
</script>

<div class="space-y-6">
  {#if loading}
    <Skeleton class="h-8 w-48" />
    <Card.Root><Card.Content class="py-6"><Skeleton class="h-32 w-full" /></Card.Content></Card.Root>
  {:else if error}
    <Button variant="ghost" size="sm" onclick={() => history.back()}>
      <ArrowLeft class="h-4 w-4 mr-1" /> Back
    </Button>
    <Card.Root class="border-destructive">
      <Card.Header><Card.Title class="text-destructive">Error</Card.Title></Card.Header>
      <Card.Content><p class="text-sm">{error}</p></Card.Content>
    </Card.Root>
  {:else if run}
    <Button variant="ghost" size="sm" onclick={() => history.back()}>
      <ArrowLeft class="h-4 w-4 mr-1" /> Back
    </Button>

    <div>
      <h1 class="text-2xl font-bold tracking-tight">{run.title || run.run_accession}</h1>
      <div class="flex gap-1.5 mt-3 flex-wrap">
        <Badge variant="outline" class="font-mono text-xs">
          <Hash class="h-3 w-3 mr-1" />{run.run_accession}
        </Badge>
        <Badge variant="secondary" class="text-xs">Run</Badge>
        {#if run.load_done}
          <Badge variant="default" class="text-xs">Loaded</Badge>
        {/if}
      </div>
      <div class="flex gap-2 mt-3">
        {#if run.experiment_accession}
          <Button variant="outline" size="sm" href="/browse/experiment/{run.experiment_accession}">
            <FlaskConical class="h-3 w-3 mr-1.5" /> Experiment: {run.experiment_accession}
          </Button>
        {/if}
        <Button variant="outline" size="sm" href="https://www.ncbi.nlm.nih.gov/sra/{run.run_accession}" target="_blank">
          <ExternalLink class="h-3 w-3 mr-1.5" /> NCBI SRA
        </Button>
        <Button variant="outline" size="sm" href="https://trace.ncbi.nlm.nih.gov/Traces/?view=run_browser&acc={run.run_accession}" target="_blank">
          <ExternalLink class="h-3 w-3 mr-1.5" /> Trace Viewer
        </Button>
      </div>
    </div>

    <!-- Stats cards -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card.Root>
        <Card.Header class="pb-2">
          <Card.Title class="text-sm font-medium flex items-center gap-1.5">
            <Activity class="h-4 w-4 text-muted-foreground" /> Total Spots
          </Card.Title>
        </Card.Header>
        <Card.Content>
          <span class="text-2xl font-bold">{run.total_spots ? formatNumber(run.total_spots) : '-'}</span>
        </Card.Content>
      </Card.Root>
      <Card.Root>
        <Card.Header class="pb-2">
          <Card.Title class="text-sm font-medium flex items-center gap-1.5">
            <HardDrive class="h-4 w-4 text-muted-foreground" /> Total Bases
          </Card.Title>
        </Card.Header>
        <Card.Content>
          <span class="text-2xl font-bold">{run.total_bases ? formatNumber(run.total_bases) : '-'}</span>
        </Card.Content>
      </Card.Root>
      <Card.Root>
        <Card.Header class="pb-2">
          <Card.Title class="text-sm font-medium flex items-center gap-1.5">
            <FileDown class="h-4 w-4 text-muted-foreground" /> File Size
          </Card.Title>
        </Card.Header>
        <Card.Content>
          <span class="text-2xl font-bold">{run.total_size ? formatBytes(run.total_size) : '-'}</span>
        </Card.Content>
      </Card.Root>
      <Card.Root>
        <Card.Header class="pb-2">
          <Card.Title class="text-sm font-medium">Run Date</Card.Title>
        </Card.Header>
        <Card.Content>
          <span class="text-sm">{formatDate(run.run_date)}</span>
        </Card.Content>
      </Card.Root>
    </div>

    <!-- Quality metrics -->
    {#if run.quality_score_mean || run.read_count_r1}
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Quality Metrics</Card.Title></Card.Header>
        <Card.Content>
          <dl class="grid gap-2 text-sm md:grid-cols-2">
            {#if run.quality_score_mean}
              <div class="flex justify-between"><dt class="text-muted-foreground">Mean Quality Score</dt><dd>{run.quality_score_mean.toFixed(2)}</dd></div>
            {/if}
            {#if run.quality_score_std}
              <div class="flex justify-between"><dt class="text-muted-foreground">Quality Score Std Dev</dt><dd>{run.quality_score_std.toFixed(2)}</dd></div>
            {/if}
            {#if run.read_count_r1}
              <div class="flex justify-between"><dt class="text-muted-foreground">Read Count R1</dt><dd>{formatNumber(run.read_count_r1)}</dd></div>
            {/if}
            {#if run.read_count_r2}
              <div class="flex justify-between"><dt class="text-muted-foreground">Read Count R2</dt><dd>{formatNumber(run.read_count_r2)}</dd></div>
            {/if}
          </dl>
        </Card.Content>
      </Card.Root>
    {/if}

    <!-- Run details -->
    <Card.Root>
      <Card.Header><Card.Title class="text-base">Details</Card.Title></Card.Header>
      <Card.Content>
        <dl class="grid gap-2 text-sm md:grid-cols-2">
          {#if run.center_name}
            <div class="flex justify-between"><dt class="text-muted-foreground">Center</dt><dd>{run.center_name}</dd></div>
          {/if}
          {#if run.run_center}
            <div class="flex justify-between"><dt class="text-muted-foreground">Run Center</dt><dd>{run.run_center}</dd></div>
          {/if}
          {#if run.alias}
            <div class="flex justify-between"><dt class="text-muted-foreground">Alias</dt><dd>{run.alias}</dd></div>
          {/if}
          {#if run.published}
            <div class="flex justify-between"><dt class="text-muted-foreground">Published</dt><dd>{run.published}</dd></div>
          {/if}
        </dl>
      </Card.Content>
    </Card.Root>

    <!-- Data files -->
    {#if dataFiles && Array.isArray(dataFiles) && dataFiles.length > 0}
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Data Files</Card.Title></Card.Header>
        <Card.Content class="pt-0">
          <Table.Root>
            <Table.Header>
              <Table.Row>
                <Table.Head>Filename</Table.Head>
                <Table.Head>Type</Table.Head>
                <Table.Head class="text-right">Size</Table.Head>
              </Table.Row>
            </Table.Header>
            <Table.Body>
              {#each dataFiles as file}
                <Table.Row>
                  <Table.Cell class="font-mono text-xs">{file.filename || file.name || '-'}</Table.Cell>
                  <Table.Cell class="text-xs">{file.filetype || file.type || '-'}</Table.Cell>
                  <Table.Cell class="text-right text-xs">{file.size ? formatBytes(file.size) : '-'}</Table.Cell>
                </Table.Row>
              {/each}
            </Table.Body>
          </Table.Root>
        </Card.Content>
      </Card.Root>
    {/if}
  {/if}
</div>
