<script lang="ts">
  import { onMount } from 'svelte';
  import { goto } from '$app/navigation';
  import { ApiService } from '$lib/api';
  import type { StatsResponse, HealthResponse } from '$lib/utils';
  import { formatNumber, formatCompactNumber, formatBytes } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Separator } from '$lib/components/ui/separator';
  import {
    Database,
    FlaskConical,
    Dna,
    Activity,
    Search,
    Download,
    BarChart3,
    CheckCircle2,
    XCircle,
    ArrowRight,
    FileSearch,
    CircleDot
  } from 'lucide-svelte';

  let stats = $state<StatsResponse | null>(null);
  let health = $state<HealthResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let quickSearch = $state('');

  onMount(async () => {
    try {
      const [statsData, healthData] = await Promise.all([
        ApiService.getStats(),
        ApiService.getHealth()
      ]);
      stats = statsData;
      health = healthData;
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load statistics';
    } finally {
      loading = false;
    }
  });

  function handleQuickSearch() {
    if (quickSearch.trim()) {
      goto(`/search?q=${encodeURIComponent(quickSearch.trim())}`);
    }
  }
</script>

<div class="space-y-8">
  <!-- Hero section with search -->
  <div class="rounded-xl border bg-card p-6 md:p-8">
    <div class="max-w-2xl">
      <h1 class="text-2xl md:text-3xl font-bold tracking-tight">SRA Knowledge Engine</h1>
      <p class="text-muted-foreground mt-2 text-sm md:text-base">
        Search and analyze sequencing metadata across studies, experiments, samples, and runs.
      </p>
      <div class="flex gap-2 mt-5">
        <div class="flex-1 relative">
          <Search class="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <Input
            type="text"
            placeholder="Search studies, experiments, samples..."
            bind:value={quickSearch}
            onkeydown={(e) => e.key === 'Enter' && handleQuickSearch()}
            class="pl-10"
          />
        </div>
        <Button onclick={handleQuickSearch}>
          <Search class="h-4 w-4 mr-2" /> Search
        </Button>
      </div>
      <div class="flex flex-wrap gap-2 mt-3">
        {#each ['RNA-Seq human', 'metagenomics', 'CRISPR', 'cancer genomics'] as term}
          <button
            onclick={() => goto(`/search?q=${encodeURIComponent(term)}`)}
            class="text-xs text-muted-foreground hover:text-foreground border rounded-full px-2.5 py-0.5 transition-colors"
          >
            {term}
          </button>
        {/each}
      </div>
    </div>
  </div>

  {#if loading}
    <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
      {#each Array(4) as _}
        <Card.Root>
          <Card.Content class="pt-6">
            <Skeleton class="h-4 w-20 mb-2" />
            <Skeleton class="h-8 w-24" />
          </Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if error}
    <Card.Root class="border-destructive">
      <Card.Header>
        <Card.Title class="text-destructive">Error Loading Dashboard</Card.Title>
      </Card.Header>
      <Card.Content>
        <p class="text-sm">{error}</p>
        <Button variant="outline" class="mt-3" onclick={() => location.reload()}>Retry</Button>
      </Card.Content>
    </Card.Root>
  {:else if stats}
    <!-- Stats row -->
    <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
      <Card.Root>
        <Card.Content class="pt-6">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm font-medium text-muted-foreground">Studies</span>
            <Database class="h-4 w-4 text-blue-500" />
          </div>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_studies || stats.total_documents || 0)}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Content class="pt-6">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm font-medium text-muted-foreground">Experiments</span>
            <FlaskConical class="h-4 w-4 text-violet-500" />
          </div>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_experiments || 0)}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Content class="pt-6">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm font-medium text-muted-foreground">Samples</span>
            <Dna class="h-4 w-4 text-emerald-500" />
          </div>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_samples || 0)}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Content class="pt-6">
          <div class="flex items-center justify-between mb-1">
            <span class="text-sm font-medium text-muted-foreground">Runs</span>
            <Activity class="h-4 w-4 text-amber-500" />
          </div>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_runs || 0)}</div>
        </Card.Content>
      </Card.Root>
    </div>

    <!-- Health status - compact inline -->
    {#if health}
      <div class="flex items-center gap-3 text-sm px-1">
        <span class="text-muted-foreground">System:</span>
        <div class="flex items-center gap-1.5">
          {#if health.status === 'healthy'}
            <CircleDot class="h-3 w-3 text-green-500" />
          {:else}
            <XCircle class="h-3 w-3 text-red-500" />
          {/if}
          <span class="text-muted-foreground">{health.status}</span>
        </div>
        <Separator orientation="vertical" class="h-4" />
        <span class="text-muted-foreground">DB: {health.database}</span>
        <Separator orientation="vertical" class="h-4" />
        <span class="text-muted-foreground">Index: {health.search_index}</span>
        {#if stats.index_size}
          <Separator orientation="vertical" class="h-4" />
          <span class="text-muted-foreground">{formatBytes(stats.index_size)}</span>
        {/if}
      </div>
    {/if}

    <!-- Quick actions -->
    <div class="grid gap-4 md:grid-cols-3">
      <a href="/search" class="block group">
        <Card.Root class="transition-all group-hover:border-primary/50 group-hover:shadow-sm h-full">
          <Card.Content class="pt-6">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-blue-500/10 p-2.5">
                <Search class="h-5 w-5 text-blue-500" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="font-medium text-sm">Search Database</p>
                <p class="text-xs text-muted-foreground mt-0.5">Full-text, semantic, and hybrid search</p>
              </div>
              <ArrowRight class="h-4 w-4 text-muted-foreground group-hover:translate-x-0.5 transition-transform" />
            </div>
          </Card.Content>
        </Card.Root>
      </a>

      <a href="/browse" class="block group">
        <Card.Root class="transition-all group-hover:border-primary/50 group-hover:shadow-sm h-full">
          <Card.Content class="pt-6">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-violet-500/10 p-2.5">
                <FileSearch class="h-5 w-5 text-violet-500" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="font-medium text-sm">Browse Studies</p>
                <p class="text-xs text-muted-foreground mt-0.5">Explore experiments, samples, and runs</p>
              </div>
              <ArrowRight class="h-4 w-4 text-muted-foreground group-hover:translate-x-0.5 transition-transform" />
            </div>
          </Card.Content>
        </Card.Root>
      </a>

      <a href="/export" class="block group">
        <Card.Root class="transition-all group-hover:border-primary/50 group-hover:shadow-sm h-full">
          <Card.Content class="pt-6">
            <div class="flex items-center gap-3">
              <div class="rounded-lg bg-emerald-500/10 p-2.5">
                <Download class="h-5 w-5 text-emerald-500" />
              </div>
              <div class="flex-1 min-w-0">
                <p class="font-medium text-sm">Export Data</p>
                <p class="text-xs text-muted-foreground mt-0.5">JSON, CSV, TSV, XML, JSONL formats</p>
              </div>
              <ArrowRight class="h-4 w-4 text-muted-foreground group-hover:translate-x-0.5 transition-transform" />
            </div>
          </Card.Content>
        </Card.Root>
      </a>
    </div>

    <!-- Top lists -->
    {#if (stats.top_organisms && stats.top_organisms.length > 0) || (stats.top_platforms && stats.top_platforms.length > 0)}
      <div class="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
        {#if stats.top_organisms && stats.top_organisms.length > 0}
          <Card.Root>
            <Card.Header class="pb-3">
              <Card.Title class="text-sm font-medium flex items-center gap-2">
                <Dna class="h-4 w-4 text-emerald-500" />
                Top Organisms
              </Card.Title>
            </Card.Header>
            <Card.Content class="space-y-2">
              {#each stats.top_organisms.slice(0, 5) as org}
                {@const maxCount = stats.top_organisms[0].count}
                <div class="space-y-1">
                  <div class="flex items-center justify-between text-sm">
                    <span class="truncate max-w-[200px]" title={org.name}>{org.name}</span>
                    <span class="text-xs text-muted-foreground tabular-nums">{formatCompactNumber(org.count)}</span>
                  </div>
                  <div class="h-1.5 rounded-full bg-muted overflow-hidden">
                    <div
                      class="h-full rounded-full bg-emerald-500/60"
                      style="width: {(org.count / maxCount) * 100}%"
                    ></div>
                  </div>
                </div>
              {/each}
              <a href="/stats" class="flex items-center gap-1 mt-2 text-xs text-primary hover:underline">
                View all <ArrowRight class="h-3 w-3" />
              </a>
            </Card.Content>
          </Card.Root>
        {/if}

        {#if stats.top_platforms && stats.top_platforms.length > 0}
          <Card.Root>
            <Card.Header class="pb-3">
              <Card.Title class="text-sm font-medium flex items-center gap-2">
                <Database class="h-4 w-4 text-blue-500" />
                Top Platforms
              </Card.Title>
            </Card.Header>
            <Card.Content class="space-y-2">
              {#each stats.top_platforms.slice(0, 5) as platform}
                {@const maxCount = stats.top_platforms[0].count}
                <div class="space-y-1">
                  <div class="flex items-center justify-between text-sm">
                    <span>{platform.name}</span>
                    <span class="text-xs text-muted-foreground tabular-nums">{formatCompactNumber(platform.count)}</span>
                  </div>
                  <div class="h-1.5 rounded-full bg-muted overflow-hidden">
                    <div
                      class="h-full rounded-full bg-blue-500/60"
                      style="width: {(platform.count / maxCount) * 100}%"
                    ></div>
                  </div>
                </div>
              {/each}
              <a href="/stats" class="flex items-center gap-1 mt-2 text-xs text-primary hover:underline">
                View all <ArrowRight class="h-3 w-3" />
              </a>
            </Card.Content>
          </Card.Root>
        {/if}

        {#if stats.top_strategies && stats.top_strategies.length > 0}
          <Card.Root>
            <Card.Header class="pb-3">
              <Card.Title class="text-sm font-medium flex items-center gap-2">
                <BarChart3 class="h-4 w-4 text-violet-500" />
                Top Strategies
              </Card.Title>
            </Card.Header>
            <Card.Content class="space-y-2">
              {#each stats.top_strategies.slice(0, 5) as strategy}
                {@const maxCount = stats.top_strategies[0].count}
                <div class="space-y-1">
                  <div class="flex items-center justify-between text-sm">
                    <span>{strategy.name}</span>
                    <span class="text-xs text-muted-foreground tabular-nums">{formatCompactNumber(strategy.count)}</span>
                  </div>
                  <div class="h-1.5 rounded-full bg-muted overflow-hidden">
                    <div
                      class="h-full rounded-full bg-violet-500/60"
                      style="width: {(strategy.count / maxCount) * 100}%"
                    ></div>
                  </div>
                </div>
              {/each}
              <a href="/stats" class="flex items-center gap-1 mt-2 text-xs text-primary hover:underline">
                View all <ArrowRight class="h-3 w-3" />
              </a>
            </Card.Content>
          </Card.Root>
        {/if}
      </div>
    {/if}
  {/if}
</div>
