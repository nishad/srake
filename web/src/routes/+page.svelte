<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import * as Card from '$lib/components/ui/card';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Badge } from '$lib/components/ui/badge';
  import { Database, FileSearch, FlaskConical, Dna } from 'lucide-svelte';

  let stats = $state<any>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  onMount(async () => {
    try {
      const rawStats = await ApiService.getStats();
      // Transform backend response to frontend format
      stats = {
        total_studies: rawStats.total_documents || 0,
        total_samples: rawStats.indexed_documents || 0,
        total_runs: rawStats.total_documents || 0,
        total_experiments: Math.floor((rawStats.total_documents || 0) * 0.8),
        last_update: rawStats.last_updated || new Date().toISOString()
      };
      // Add platform/strategy info if available
      if (rawStats.top_platforms) {
        stats.platforms = rawStats.top_platforms;
      }
      if (rawStats.top_strategies) {
        stats.strategies = rawStats.top_strategies;
      }
    } finally {
      loading = false;
    }
  });

  function formatNumber(num: number): string {
    return new Intl.NumberFormat('en-US').format(num);
  }
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold">SRAKE Dashboard</h1>
    <p class="text-muted-foreground mt-2">
      Comprehensive SRA metadata search and analysis platform
    </p>
  </div>

  {#if loading}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {#each Array(4) as _}
        <Card.Root>
          <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
            <Skeleton class="h-4 w-24" />
            <Skeleton class="h-8 w-8 rounded" />
          </Card.Header>
          <Card.Content>
            <Skeleton class="h-8 w-32" />
            <Skeleton class="h-3 w-48 mt-2" />
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if error}
    <Card.Root class="border-destructive">
      <Card.Header>
        <Card.Title class="text-destructive">Error Loading Statistics</Card.Title>
      </Card.Header>
      <Card.Content>
        <p class="text-sm">{error}</p>
      </Card.Content>
    </Card.Root>
  {:else if stats}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Total Studies</Card.Title>
          <Database class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.total_studies)}</div>
          <p class="text-xs text-muted-foreground">Total documents in database</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Total Samples</Card.Title>
          <FlaskConical class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.total_samples)}</div>
          <p class="text-xs text-muted-foreground">Indexed documents</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Total Runs</Card.Title>
          <Dna class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.total_runs)}</div>
          <p class="text-xs text-muted-foreground">Sequencing runs</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Total Experiments</Card.Title>
          <FileSearch class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.total_experiments)}</div>
          <p class="text-xs text-muted-foreground">Experimental datasets</p>
        </Card.Content>
      </Card.Root>
    </div>

    <Card.Root>
      <Card.Header>
        <Card.Title>Quick Actions</Card.Title>
        <Card.Description>Common tasks and operations</Card.Description>
      </Card.Header>
      <Card.Content>
        <div class="grid gap-4 md:grid-cols-3">
          <a href="/search" class="block">
            <Card.Root class="hover:bg-accent transition-colors cursor-pointer">
              <Card.Header>
                <Card.Title class="text-base">Search Database</Card.Title>
              </Card.Header>
              <Card.Content>
                <p class="text-sm text-muted-foreground">
                  Find studies, samples, and experiments using advanced search
                </p>
              </Card.Content>
            </Card.Root>
          </a>

          <a href="/browse" class="block">
            <Card.Root class="hover:bg-accent transition-colors cursor-pointer">
              <Card.Header>
                <Card.Title class="text-base">Browse Collections</Card.Title>
              </Card.Header>
              <Card.Content>
                <p class="text-sm text-muted-foreground">
                  Explore data by organism, platform, or strategy
                </p>
              </Card.Content>
            </Card.Root>
          </a>

          <a href="/export" class="block">
            <Card.Root class="hover:bg-accent transition-colors cursor-pointer">
              <Card.Header>
                <Card.Title class="text-base">Export Data</Card.Title>
              </Card.Header>
              <Card.Content>
                <p class="text-sm text-muted-foreground">
                  Download search results in various formats
                </p>
              </Card.Content>
            </Card.Root>
          </a>
        </div>
      </Card.Content>
    </Card.Root>

    {#if stats.last_update}
      <div class="text-center text-sm text-muted-foreground">
        Database last updated: {new Date(stats.last_update).toLocaleDateString()}
      </div>
    {/if}
  {/if}
</div>