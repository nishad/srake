<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import type { Experiment } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Separator } from '$lib/components/ui/separator';
  import {
    ArrowLeft,
    ExternalLink,
    Hash,
    FlaskConical,
    Dna,
    Database
  } from 'lucide-svelte';

  let accession = $derived($page.params.id);
  let experiment = $state<Experiment | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    if (!accession) return;
    try {
      experiment = await ApiService.getExperiment(accession);
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load experiment';
    } finally {
      loading = false;
    }
  });
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
  {:else if experiment}
    <Button variant="ghost" size="sm" onclick={() => history.back()}>
      <ArrowLeft class="h-4 w-4 mr-1" /> Back
    </Button>

    <div>
      <h1 class="text-2xl font-bold tracking-tight">{experiment.title || experiment.experiment_accession}</h1>
      <div class="flex gap-1.5 mt-3 flex-wrap">
        <Badge variant="outline" class="font-mono text-xs">
          <Hash class="h-3 w-3 mr-1" />{experiment.experiment_accession}
        </Badge>
        <Badge variant="secondary" class="text-xs">Experiment</Badge>
        {#if experiment.platform}
          <Badge variant="secondary" class="text-xs">{experiment.platform}</Badge>
        {/if}
        {#if experiment.library_strategy}
          <Badge variant="secondary" class="text-xs">{experiment.library_strategy}</Badge>
        {/if}
        {#if experiment.library_layout}
          <Badge variant="outline" class="text-xs">{experiment.library_layout}</Badge>
        {/if}
      </div>
      <div class="flex gap-2 mt-3">
        {#if experiment.study_accession}
          <Button variant="outline" size="sm" href="/browse/study/{experiment.study_accession}">
            <Database class="h-3 w-3 mr-1.5" /> Study: {experiment.study_accession}
          </Button>
        {/if}
        {#if experiment.sample_accession}
          <Button variant="outline" size="sm" href="/browse/sample/{experiment.sample_accession}">
            <Dna class="h-3 w-3 mr-1.5" /> Sample: {experiment.sample_accession}
          </Button>
        {/if}
      </div>
    </div>

    {#if experiment.design_description}
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Design Description</Card.Title></Card.Header>
        <Card.Content>
          <p class="text-sm leading-relaxed">{experiment.design_description}</p>
        </Card.Content>
      </Card.Root>
    {/if}

    <div class="grid gap-4 md:grid-cols-2">
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Library Information</Card.Title></Card.Header>
        <Card.Content>
          <dl class="grid gap-2 text-sm">
            {#if experiment.library_name}
              <div class="flex justify-between"><dt class="text-muted-foreground">Name</dt><dd>{experiment.library_name}</dd></div>
            {/if}
            <div class="flex justify-between"><dt class="text-muted-foreground">Strategy</dt><dd>{experiment.library_strategy || '-'}</dd></div>
            <div class="flex justify-between"><dt class="text-muted-foreground">Source</dt><dd>{experiment.library_source || '-'}</dd></div>
            <div class="flex justify-between"><dt class="text-muted-foreground">Selection</dt><dd>{experiment.library_selection || '-'}</dd></div>
            <div class="flex justify-between"><dt class="text-muted-foreground">Layout</dt><dd>{experiment.library_layout || '-'}</dd></div>
            {#if experiment.nominal_length}
              <div class="flex justify-between"><dt class="text-muted-foreground">Nominal Length</dt><dd>{experiment.nominal_length}</dd></div>
            {/if}
          </dl>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header><Card.Title class="text-base">Platform Details</Card.Title></Card.Header>
        <Card.Content>
          <dl class="grid gap-2 text-sm">
            <div class="flex justify-between"><dt class="text-muted-foreground">Platform</dt><dd>{experiment.platform || '-'}</dd></div>
            <div class="flex justify-between"><dt class="text-muted-foreground">Instrument</dt><dd>{experiment.instrument_model || '-'}</dd></div>
            {#if experiment.spot_length}
              <div class="flex justify-between"><dt class="text-muted-foreground">Spot Length</dt><dd>{experiment.spot_length}</dd></div>
            {/if}
            {#if experiment.center_name}
              <div class="flex justify-between"><dt class="text-muted-foreground">Center</dt><dd>{experiment.center_name}</dd></div>
            {/if}
            {#if experiment.alias}
              <div class="flex justify-between"><dt class="text-muted-foreground">Alias</dt><dd>{experiment.alias}</dd></div>
            {/if}
          </dl>
        </Card.Content>
      </Card.Root>
    </div>

    {#if experiment.library_construction_protocol}
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Library Construction Protocol</Card.Title></Card.Header>
        <Card.Content>
          <p class="text-sm leading-relaxed whitespace-pre-wrap">{experiment.library_construction_protocol}</p>
        </Card.Content>
      </Card.Root>
    {/if}
  {/if}
</div>
