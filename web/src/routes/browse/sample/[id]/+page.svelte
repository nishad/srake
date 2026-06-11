<script lang="ts">
  import { page } from '$app/stores';
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import type { Sample, SearchResult } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import {
    ArrowLeft,
    ExternalLink,
    Hash,
    Dna,
    MapPin,
    Sparkles
  } from 'lucide-svelte';

  let accession = $derived($page.params.id);
  let sample = $state<Sample | null>(null);
  let similarSamples = $state<SearchResult[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    if (!accession) return;
    try {
      sample = await ApiService.getSample(accession);

      // Find similar samples via vector search
      const searchTerm = sample.scientific_name || sample.organism || sample.title || accession;
      try {
        const similar = await ApiService.search({
          query: searchTerm,
          searchMode: 'vector',
          limit: 6,
          showConfidence: true
        });
        similarSamples = (similar.results || []).filter(r => r.id !== accession).slice(0, 4);
      } catch {
        similarSamples = [];
      }
    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load sample';
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
  {:else if sample}
    <Button variant="ghost" size="sm" onclick={() => history.back()}>
      <ArrowLeft class="h-4 w-4 mr-1" /> Back
    </Button>

    <div>
      <h1 class="text-2xl font-bold tracking-tight">{sample.title || sample.sample_accession}</h1>
      <div class="flex gap-1.5 mt-3 flex-wrap">
        <Badge variant="outline" class="font-mono text-xs">
          <Hash class="h-3 w-3 mr-1" />{sample.sample_accession}
        </Badge>
        <Badge variant="secondary" class="text-xs">Sample</Badge>
        {#if sample.scientific_name || sample.organism}
          <Badge variant="secondary" class="text-xs">
            <Dna class="h-3 w-3 mr-1" />{sample.scientific_name || sample.organism}
          </Badge>
        {/if}
        {#if sample.biosample_accession}
          <Badge variant="outline" class="text-xs">BioSample: {sample.biosample_accession}</Badge>
        {/if}
      </div>
      <div class="flex gap-2 mt-3">
        <Button variant="outline" size="sm" href="https://www.ncbi.nlm.nih.gov/sra/{sample.sample_accession}" target="_blank">
          <ExternalLink class="h-3 w-3 mr-1.5" /> NCBI
        </Button>
        {#if sample.biosample_accession}
          <Button variant="outline" size="sm" href="https://www.ncbi.nlm.nih.gov/biosample/{sample.biosample_accession}" target="_blank">
            <ExternalLink class="h-3 w-3 mr-1.5" /> BioSample
          </Button>
        {/if}
      </div>
    </div>

    {#if sample.description}
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Description</Card.Title></Card.Header>
        <Card.Content>
          <p class="text-sm leading-relaxed">{sample.description}</p>
        </Card.Content>
      </Card.Root>
    {/if}

    <div class="grid gap-4 md:grid-cols-2">
      <Card.Root>
        <Card.Header><Card.Title class="text-base">Taxonomy</Card.Title></Card.Header>
        <Card.Content>
          <dl class="grid gap-2 text-sm">
            {#if sample.scientific_name}
              <div class="flex justify-between"><dt class="text-muted-foreground">Scientific Name</dt><dd class="italic">{sample.scientific_name}</dd></div>
            {/if}
            {#if sample.common_name}
              <div class="flex justify-between"><dt class="text-muted-foreground">Common Name</dt><dd>{sample.common_name}</dd></div>
            {/if}
            {#if sample.taxon_id}
              <div class="flex justify-between"><dt class="text-muted-foreground">Taxon ID</dt><dd>{sample.taxon_id}</dd></div>
            {/if}
            {#if sample.strain}
              <div class="flex justify-between"><dt class="text-muted-foreground">Strain</dt><dd>{sample.strain}</dd></div>
            {/if}
          </dl>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header><Card.Title class="text-base">Biological Details</Card.Title></Card.Header>
        <Card.Content>
          <dl class="grid gap-2 text-sm">
            {#if sample.tissue}
              <div class="flex justify-between"><dt class="text-muted-foreground">Tissue</dt><dd>{sample.tissue}</dd></div>
            {/if}
            {#if sample.cell_type}
              <div class="flex justify-between"><dt class="text-muted-foreground">Cell Type</dt><dd>{sample.cell_type}</dd></div>
            {/if}
            {#if sample.cell_line}
              <div class="flex justify-between"><dt class="text-muted-foreground">Cell Line</dt><dd>{sample.cell_line}</dd></div>
            {/if}
            {#if sample.sex}
              <div class="flex justify-between"><dt class="text-muted-foreground">Sex</dt><dd>{sample.sex}</dd></div>
            {/if}
            {#if sample.age}
              <div class="flex justify-between"><dt class="text-muted-foreground">Age</dt><dd>{sample.age}</dd></div>
            {/if}
            {#if sample.disease}
              <div class="flex justify-between"><dt class="text-muted-foreground">Disease</dt><dd>{sample.disease}</dd></div>
            {/if}
            {#if sample.treatment}
              <div class="flex justify-between"><dt class="text-muted-foreground">Treatment</dt><dd>{sample.treatment}</dd></div>
            {/if}
          </dl>
        </Card.Content>
      </Card.Root>
    </div>

    {#if sample.geo_loc_name || sample.collection_date || sample.env_biome}
      <Card.Root>
        <Card.Header>
          <Card.Title class="text-base flex items-center gap-2">
            <MapPin class="h-4 w-4" /> Geographic & Environmental
          </Card.Title>
        </Card.Header>
        <Card.Content>
          <dl class="grid gap-2 text-sm md:grid-cols-2">
            {#if sample.geo_loc_name}
              <div class="flex justify-between"><dt class="text-muted-foreground">Location</dt><dd>{sample.geo_loc_name}</dd></div>
            {/if}
            {#if sample.lat_lon}
              <div class="flex justify-between"><dt class="text-muted-foreground">Coordinates</dt><dd>{sample.lat_lon}</dd></div>
            {/if}
            {#if sample.collection_date}
              <div class="flex justify-between"><dt class="text-muted-foreground">Collection Date</dt><dd>{sample.collection_date}</dd></div>
            {/if}
            {#if sample.env_biome}
              <div class="flex justify-between"><dt class="text-muted-foreground">Biome</dt><dd>{sample.env_biome}</dd></div>
            {/if}
            {#if sample.env_feature}
              <div class="flex justify-between"><dt class="text-muted-foreground">Feature</dt><dd>{sample.env_feature}</dd></div>
            {/if}
            {#if sample.env_material}
              <div class="flex justify-between"><dt class="text-muted-foreground">Material</dt><dd>{sample.env_material}</dd></div>
            {/if}
          </dl>
        </Card.Content>
      </Card.Root>
    {/if}

    <!-- Similar Samples -->
    {#if similarSamples.length > 0}
      <Card.Root>
        <Card.Header>
          <Card.Title class="text-base flex items-center gap-2">
            <Sparkles class="h-4 w-4" /> Similar Samples
          </Card.Title>
        </Card.Header>
        <Card.Content>
          <div class="grid gap-3 md:grid-cols-2">
            {#each similarSamples as sim}
              <a href="/browse/{sim.type?.toLowerCase() || 'study'}/{sim.id}" class="block group">
                <div class="rounded-lg border p-3 transition-colors group-hover:bg-accent/50">
                  <p class="text-sm font-medium line-clamp-1">{sim.title || sim.id}</p>
                  <div class="flex gap-1.5 mt-1.5">
                    <Badge variant="outline" class="text-xs">{sim.id}</Badge>
                    {#if sim.similarity && sim.similarity > 0}
                      <Badge variant="outline" class="text-xs gap-1">
                        <Sparkles class="h-2.5 w-2.5" />{(sim.similarity * 100).toFixed(0)}%
                      </Badge>
                    {/if}
                  </div>
                </div>
              </a>
            {/each}
          </div>
        </Card.Content>
      </Card.Root>
    {/if}
  {/if}
</div>
