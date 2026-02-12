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
      // Use the actual backend response
      stats = {
        total_documents: rawStats.total_documents || 0,
        indexed_documents: rawStats.indexed_documents || 0,
        last_updated: rawStats.last_updated || new Date().toISOString(),
        top_platforms: rawStats.top_platforms || [],
        top_strategies: rawStats.top_strategies || [],
        top_organisms: rawStats.top_organisms || []
      };
    } catch (e) {
      error = e instanceof Error ? e.message : 'Failed to load statistics';
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
          <Card.Title class="text-sm font-medium">Total Records</Card.Title>
          <Database class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.total_documents)}</div>
          <p class="text-xs text-muted-foreground">Total documents in database</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Indexed</Card.Title>
          <FileSearch class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.indexed_documents)}</div>
          <p class="text-xs text-muted-foreground">Indexed for search</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Top Platform</Card.Title>
          <Dna class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          {#if stats.top_platforms && stats.top_platforms.length > 0}
            <div class="text-2xl font-bold">{stats.top_platforms[0].name}</div>
            <p class="text-xs text-muted-foreground">{formatNumber(stats.top_platforms[0].count)} experiments</p>
          {:else}
            <div class="text-2xl font-bold">-</div>
          {/if}
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Top Strategy</Card.Title>
          <FlaskConical class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          {#if stats.top_strategies && stats.top_strategies.length > 0}
            <div class="text-2xl font-bold">{stats.top_strategies[0].name}</div>
            <p class="text-xs text-muted-foreground">{formatNumber(stats.top_strategies[0].count)} experiments</p>
          {:else}
            <div class="text-2xl font-bold">-</div>
          {/if}
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

    {#if stats.last_updated}
      <div class="text-center text-sm text-muted-foreground">
        Database last updated: {new Date(stats.last_updated).toLocaleDateString()}
      </div>
    {/if}
  {/if}
</div>