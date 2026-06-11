<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import type { StatsResponse, HealthResponse } from '$lib/utils';
  import { formatCompactNumber, formatNumber, formatBytes } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Skeleton } from '$lib/components/ui/skeleton';
  import { Badge } from '$lib/components/ui/badge';
  import { Button } from '$lib/components/ui/button';
  import {
    BarChart3,
    Database,
    FileSearch,
    Dna,
    FlaskConical,
    Activity,
    CheckCircle2,
    XCircle
  } from 'lucide-svelte';

  let stats = $state<StatsResponse | null>(null);
  let health = $state<HealthResponse | null>(null);
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
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold tracking-tight">Statistics</h1>
    <p class="text-muted-foreground mt-1">Database analytics and system health</p>
  </div>

  {#if loading}
    <div class="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      {#each Array(4) as _}
        <Card.Root>
          <Card.Header class="pb-2"><Skeleton class="h-4 w-24" /></Card.Header>
          <Card.Content><Skeleton class="h-8 w-32" /></Card.Content>
        </Card.Root>
      {/each}
    </div>
  {:else if error}
    <Card.Root class="border-destructive">
      <Card.Header><Card.Title class="text-destructive">Error</Card.Title></Card.Header>
      <Card.Content>
        <p class="text-sm">{error}</p>
        <Button variant="outline" class="mt-3" onclick={() => location.reload()}>Retry</Button>
      </Card.Content>
    </Card.Root>
  {:else if stats}
    <!-- Overview cards -->
    <div class="grid gap-4 grid-cols-2 lg:grid-cols-4">
      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Studies</Card.Title>
          <Database class="h-4 w-4 text-blue-500" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_studies || stats.total_documents || 0)}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Experiments</Card.Title>
          <FlaskConical class="h-4 w-4 text-violet-500" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_experiments || 0)}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">Samples</Card.Title>
          <Dna class="h-4 w-4 text-emerald-500" />
        </Card.Header>
        <Card.Content>
          <div class="text-2xl font-bold">{formatCompactNumber(stats.total_samples || 0)}</div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header class="flex flex-row items-center justify-between space-y-0 pb-2">
          <Card.Title class="text-sm font-medium">System</Card.Title>
          <Activity class="h-4 w-4 text-muted-foreground" />
        </Card.Header>
        <Card.Content>
          {#if health}
            <div class="flex gap-1.5 flex-wrap">
              <Badge variant={health.status === 'healthy' ? 'default' : 'destructive'} class="text-xs gap-1">
                {#if health.status === 'healthy'}<CheckCircle2 class="h-3 w-3" />{:else}<XCircle class="h-3 w-3" />{/if}
                {health.status}
              </Badge>
            </div>
          {/if}
        </Card.Content>
      </Card.Root>
    </div>

    <!-- Top lists with visual bars -->
    <div class="grid gap-6 md:grid-cols-3">
      {#if stats.top_organisms && stats.top_organisms.length > 0}
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2 text-base">
              <Dna class="h-4 w-4 text-emerald-500" /> Top Organisms
            </Card.Title>
          </Card.Header>
          <Card.Content>
            <div class="space-y-2.5">
              {#each stats.top_organisms as org, i}
                {@const maxCount = stats.top_organisms[0].count}
                <div class="space-y-1">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="text-xs text-muted-foreground w-5 tabular-nums">{i + 1}.</span>
                      <span class="text-sm truncate" title={org.name}>{org.name}</span>
                    </div>
                    <span class="text-xs text-muted-foreground tabular-nums shrink-0 ml-2">{formatCompactNumber(org.count)}</span>
                  </div>
                  <div class="h-1 rounded-full bg-muted overflow-hidden ml-7">
                    <div class="h-full rounded-full bg-emerald-500/50" style="width: {(org.count / maxCount) * 100}%"></div>
                  </div>
                </div>
              {/each}
            </div>
          </Card.Content>
        </Card.Root>
      {/if}

      {#if stats.top_platforms && stats.top_platforms.length > 0}
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2 text-base">
              <Database class="h-4 w-4 text-blue-500" /> Top Platforms
            </Card.Title>
          </Card.Header>
          <Card.Content>
            <div class="space-y-2.5">
              {#each stats.top_platforms as platform, i}
                {@const maxCount = stats.top_platforms[0].count}
                <div class="space-y-1">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="text-xs text-muted-foreground w-5 tabular-nums">{i + 1}.</span>
                      <span class="text-sm truncate">{platform.name}</span>
                    </div>
                    <span class="text-xs text-muted-foreground tabular-nums shrink-0 ml-2">{formatCompactNumber(platform.count)}</span>
                  </div>
                  <div class="h-1 rounded-full bg-muted overflow-hidden ml-7">
                    <div class="h-full rounded-full bg-blue-500/50" style="width: {(platform.count / maxCount) * 100}%"></div>
                  </div>
                </div>
              {/each}
            </div>
          </Card.Content>
        </Card.Root>
      {/if}

      {#if stats.top_strategies && stats.top_strategies.length > 0}
        <Card.Root>
          <Card.Header>
            <Card.Title class="flex items-center gap-2 text-base">
              <FlaskConical class="h-4 w-4 text-violet-500" /> Top Strategies
            </Card.Title>
          </Card.Header>
          <Card.Content>
            <div class="space-y-2.5">
              {#each stats.top_strategies as strategy, i}
                {@const maxCount = stats.top_strategies[0].count}
                <div class="space-y-1">
                  <div class="flex items-center justify-between">
                    <div class="flex items-center gap-2 min-w-0">
                      <span class="text-xs text-muted-foreground w-5 tabular-nums">{i + 1}.</span>
                      <span class="text-sm truncate">{strategy.name}</span>
                    </div>
                    <span class="text-xs text-muted-foreground tabular-nums shrink-0 ml-2">{formatCompactNumber(strategy.count)}</span>
                  </div>
                  <div class="h-1 rounded-full bg-muted overflow-hidden ml-7">
                    <div class="h-full rounded-full bg-violet-500/50" style="width: {(strategy.count / maxCount) * 100}%"></div>
                  </div>
                </div>
              {/each}
            </div>
          </Card.Content>
        </Card.Root>
      {/if}
    </div>

    {#if stats.last_updated || stats.last_update}
      <p class="text-center text-sm text-muted-foreground">
        Last updated: {new Date(stats.last_updated || stats.last_update).toLocaleString()}
      </p>
    {/if}
  {/if}
</div>
