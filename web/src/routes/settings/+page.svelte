<script lang="ts">
  import { onMount } from 'svelte';
  import { setMode, resetMode, mode, userPrefersMode } from 'mode-watcher';
  import { ApiService } from '$lib/api';
  import type { HealthResponse, StatsResponse } from '$lib/utils';
  import { formatNumber, formatBytes } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { Badge } from '$lib/components/ui/badge';
  import { Separator } from '$lib/components/ui/separator';
  import {
    Database,
    Search,
    Palette,
    CheckCircle2,
    XCircle,
    Sun,
    Moon,
    Monitor,
    RotateCcw
  } from 'lucide-svelte';

  let searchLimit = $state(25);
  let defaultSearchMode = $state('hybrid');
  let showConfidence = $state(true);
  let health = $state<HealthResponse | null>(null);
  let stats = $state<StatsResponse | null>(null);
  let loading = $state(true);

  const searchModes = [
    { value: 'database', label: 'Database' },
    { value: 'fts', label: 'Full-Text' },
    { value: 'hybrid', label: 'Hybrid' },
    { value: 'vector', label: 'Semantic' }
  ];

  onMount(async () => {
    // Load settings from localStorage
    const saved = localStorage.getItem('srake-settings');
    if (saved) {
      try {
        const s = JSON.parse(saved);
        searchLimit = s.searchLimit ?? 25;
        defaultSearchMode = s.defaultSearchMode ?? 'hybrid';
        showConfidence = s.showConfidence ?? true;
      } catch { /* ignore */ }
    }

    try {
      const [h, s] = await Promise.all([
        ApiService.getHealth(),
        ApiService.getStats()
      ]);
      health = h;
      stats = s;
    } catch { /* ignore */ }
    finally { loading = false; }
  });

  function saveSettings() {
    localStorage.setItem('srake-settings', JSON.stringify({
      searchLimit,
      defaultSearchMode,
      showConfidence,
    }));
  }

  function resetSettings() {
    searchLimit = 25;
    defaultSearchMode = 'hybrid';
    showConfidence = true;
    saveSettings();
  }
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold tracking-tight">Settings</h1>
    <p class="text-muted-foreground mt-1">Preferences and system information</p>
  </div>

  <div class="grid gap-6 lg:grid-cols-3">
    <div class="lg:col-span-2 space-y-6">
      <!-- Search settings -->
      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Search class="h-4 w-4" /> Search Defaults
          </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-4">
          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="limit">Results Per Page</Label>
              <Input id="limit" type="number" bind:value={searchLimit} min="10" max="100" onchange={saveSettings} />
            </div>
            <div class="space-y-2">
              <Label>Default Search Mode</Label>
              <div class="flex gap-2 flex-wrap">
                {#each searchModes as m}
                  <Button
                    size="sm"
                    variant={defaultSearchMode === m.value ? 'default' : 'outline'}
                    onclick={() => { defaultSearchMode = m.value; saveSettings(); }}
                  >
                    {m.label}
                  </Button>
                {/each}
              </div>
            </div>
          </div>
          <div class="flex items-center gap-3">
            <input
              type="checkbox"
              id="confidence"
              bind:checked={showConfidence}
              onchange={saveSettings}
              class="rounded accent-primary"
            />
            <Label for="confidence">Show confidence scores in search results</Label>
          </div>
        </Card.Content>
      </Card.Root>

      <!-- Theme settings -->
      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Palette class="h-4 w-4" /> Appearance
          </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-4">
          <div class="space-y-2">
            <Label>Theme</Label>
            <div class="flex gap-2">
              <Button
                variant={mode.current === 'light' ? 'default' : 'outline'}
                size="sm"
                onclick={() => setMode('light')}
                class="gap-1.5"
              >
                <Sun class="h-3.5 w-3.5" /> Light
              </Button>
              <Button
                variant={mode.current === 'dark' ? 'default' : 'outline'}
                size="sm"
                onclick={() => setMode('dark')}
                class="gap-1.5"
              >
                <Moon class="h-3.5 w-3.5" /> Dark
              </Button>
              <Button
                variant={userPrefersMode.current === 'system' ? 'default' : 'outline'}
                size="sm"
                onclick={() => resetMode()}
                class="gap-1.5"
              >
                <Monitor class="h-3.5 w-3.5" /> System
              </Button>
            </div>
          </div>

          <Separator />

          <Button variant="outline" size="sm" onclick={resetSettings} class="gap-1.5">
            <RotateCcw class="h-3 w-3" /> Reset All Settings
          </Button>
        </Card.Content>
      </Card.Root>
    </div>

    <!-- System info sidebar -->
    <div class="space-y-4">
      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Database class="h-4 w-4" /> System Status
          </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-3">
          {#if loading}
            <p class="text-sm text-muted-foreground">Loading...</p>
          {:else if health}
            <div class="space-y-2">
              {#each [
                { label: 'API', value: health.status },
                { label: 'Database', value: health.database },
                { label: 'Search Index', value: health.search_index },
              ] as item}
                <div class="flex items-center justify-between">
                  <span class="text-sm">{item.label}</span>
                  <Badge variant={item.value === 'healthy' ? 'default' : 'destructive'} class="text-xs gap-1">
                    {#if item.value === 'healthy'}<CheckCircle2 class="h-3 w-3" />{:else}<XCircle class="h-3 w-3" />{/if}
                    {item.value}
                  </Badge>
                </div>
              {/each}
            </div>
          {:else}
            <p class="text-sm text-muted-foreground">Unable to connect to API</p>
          {/if}
        </Card.Content>
      </Card.Root>

      {#if stats}
        <Card.Root>
          <Card.Header><Card.Title>Database</Card.Title></Card.Header>
          <Card.Content class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">Records</span>
              <span class="font-medium">{formatNumber(stats.total_documents || stats.total_studies || 0)}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">Indexed</span>
              <span class="font-medium">{formatNumber(stats.indexed_documents || 0)}</span>
            </div>
            {#if stats.index_size}
              <div class="flex justify-between">
                <span class="text-muted-foreground">Index Size</span>
                <span class="font-medium">{formatBytes(stats.index_size)}</span>
              </div>
            {/if}
          </Card.Content>
        </Card.Root>
      {/if}
    </div>
  </div>
</div>
