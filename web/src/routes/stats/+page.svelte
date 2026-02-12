<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import * as Card from '$lib/components/ui/card';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Badge } from '$lib/components/ui/badge';
  import { BarChart3, Database, FileSearch, Dna, FlaskConical, Users } from 'lucide-svelte';

  let stats = $state<any>(null);
  let health = $state<any>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

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

  function formatNumber(num: number): string {
    if (num >= 1_000_000_000) {
      return (num / 1_000_000_000).toFixed(1) + 'B';
    }
    if (num >= 1_000_000) {
      return (num / 1_000_000).toFixed(1) + 'M';
    }
    if (num >= 1_000) {
      return (num / 1_000).toFixed(1) + 'K';
    }
    return num.toString();
  }

  function formatBytes(bytes: number): string {
    if (bytes >= 1_073_741_824) {
      return (bytes / 1_073_741_824).toFixed(2) + ' GB';
    }
    if (bytes >= 1_048_576) {
      return (bytes / 1_048_576).toFixed(2) + ' MB';
    }
    if (bytes >= 1024) {
      return (bytes / 1024).toFixed(2) + ' KB';
    }
    return bytes + ' B';
  }
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold">Statistics</h1>
    <p class="text-muted-foreground mt-2">
      Detailed analytics and insights about the database
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
    <!-- Overview Stats -->
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Total Documents</Card.Title>
          <Database class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.total_documents)}</div>
          <p class="text-xs text-muted-foreground">Records in database</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Indexed Documents</Card.Title>
          <FileSearch class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatNumber(stats.indexed_documents)}</div>
          <p class="text-xs text-muted-foreground">
            {stats.total_documents > 0 ? ((stats.indexed_documents / stats.total_documents) * 100).toFixed(1) : 0}% coverage
          </p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Index Size</Card.Title>
          <BarChart3 class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatBytes(stats.index_size)}</div>
          <p class="text-xs text-muted-foreground">Search index storage</p>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">System Status</Card.Title>
          <Users class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          <div class="flex gap-2">
            {#if health}
              <Badge variant={health.database === 'healthy' ? 'default' : 'destructive'}>
                DB: {health.database}
              </Badge>
              <Badge variant={health.search_index === 'healthy' ? 'default' : 'destructive'}>
                Index: {health.search_index}
              </Badge>
            {/if}
          </div>
        </Card.Content>
      </Card.Root>
    </div>

    <!-- Top Lists -->
    <div class="grid gap-6 md:grid-cols-3">
      {#if stats.top_organisms && stats.top_organisms.length > 0}
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2">
              <Dna class="h-5 w-5" />
              Top Organisms
            </Card.Title>
          </Card.Header>
          <Card.Content>
            <div class="space-y-3">
              {#each stats.top_organisms.slice(0, 10) as org}
                <div class="flex items-center justify-between">
                  <span class="text-sm truncate max-w-[200px]" title={org.name}>
                    {org.name}
                  </span>
                  <Badge variant="secondary">{formatNumber(org.count)}</Badge>
                </div>
              {/each}
            </div>
          </Card.Content>
        </Card.Root>
      {/if}

      {#if stats.top_platforms && stats.top_platforms.length > 0}
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2">
              <Database class="h-5 w-5" />
              Top Platforms
            </Card.Title>
          </Card.Header>
          <Card.Content>
            <div class="space-y-3">
              {#each stats.top_platforms.slice(0, 10) as platform}
                <div class="flex items-center justify-between">
                  <span class="text-sm">{platform.name}</span>
                  <Badge variant="secondary">{formatNumber(platform.count)}</Badge>
                </div>
              {/each}
            </div>
          </Card.Content>
        </Card.Root>
      {/if}

      {#if stats.top_strategies && stats.top_strategies.length > 0}
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2">
              <FlaskConical class="h-5 w-5" />
              Top Library Strategies
            </Card.Title>
          </Card.Header>
          <Card.Content>
            <div class="space-y-3">
              {#each stats.top_strategies.slice(0, 10) as strategy}
                <div class="flex items-center justify-between">
                  <span class="text-sm">{strategy.name}</span>
                  <Badge variant="secondary">{formatNumber(strategy.count)}</Badge>
                </div>
              {/each}
            </div>
          </Card.Content>
        </Card.Root>
      {/if}
    </div>

    {#if stats.last_updated}
      <div class="text-center text-sm text-muted-foreground">
        Last updated: {new Date(stats.last_updated).toLocaleString()}
      </div>
    {/if}
  {/if}
</div>
