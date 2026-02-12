<script lang="ts">
  import { onMount } from 'svelte';
  import { ApiService } from '$lib/api';
  import * as Card from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import * as Select from '$lib/components/ui/select';
  import { Badge } from '$lib/components/ui/badge';
  import { Database, Search, Palette, CheckCircle2, AlertCircle } from 'lucide-svelte';

  // Settings state (persisted to localStorage)
  let searchLimit = $state(25);
  let defaultSearchMode = $state<{ value: string; label: string }>({ value: 'hybrid', label: 'Hybrid' });
  let showConfidence = $state(true);
  let theme = $state<{ value: string; label: string }>({ value: 'system', label: 'System' });
  let compactView = $state(false);

  // System info
  let health = $state<any>(null);
  let stats = $state<any>(null);
  let loading = $state(true);

  const searchModes = [
    { value: 'database', label: 'Database', description: 'Direct SQLite queries' },
    { value: 'fts', label: 'Full-Text Search', description: 'FTS5 full-text search' },
    { value: 'hybrid', label: 'Hybrid', description: 'Combined text and vector search' },
    { value: 'vector', label: 'Vector', description: 'Semantic similarity search' }
  ];

  const themes = [
    { value: 'light', label: 'Light' },
    { value: 'dark', label: 'Dark' },
    { value: 'system', label: 'System' }
  ];

  onMount(async () => {
    // Load saved settings from localStorage
    const saved = localStorage.getItem('srake-settings');
    if (saved) {
      try {
        const settings = JSON.parse(saved);
        searchLimit = settings.searchLimit ?? 25;
        defaultSearchMode = searchModes.find(m => m.value === settings.defaultSearchMode) ?? defaultSearchMode;
        showConfidence = settings.showConfidence ?? true;
        theme = themes.find(t => t.value === settings.theme) ?? theme;
        compactView = settings.compactView ?? false;
      } catch (e) {
        console.error('Failed to load settings:', e);
      }
    }

    // Load system info
    try {
      const [healthData, statsData] = await Promise.all([
        ApiService.getHealth(),
        ApiService.getStats()
      ]);
      health = healthData;
      stats = statsData;
    } catch (e) {
      console.error('Failed to load system info:', e);
    } finally {
      loading = false;
    }
  });

  function saveSettings() {
    const settings = {
      searchLimit,
      defaultSearchMode: defaultSearchMode.value,
      showConfidence,
      theme: theme.value,
      compactView
    };
    localStorage.setItem('srake-settings', JSON.stringify(settings));

    // Apply theme
    if (theme.value === 'dark') {
      document.documentElement.classList.add('dark');
    } else if (theme.value === 'light') {
      document.documentElement.classList.remove('dark');
    } else {
      // System preference
      if (window.matchMedia('(prefers-color-scheme: dark)').matches) {
        document.documentElement.classList.add('dark');
      } else {
        document.documentElement.classList.remove('dark');
      }
    }
  }

  function resetSettings() {
    searchLimit = 25;
    defaultSearchMode = { value: 'hybrid', label: 'Hybrid' };
    showConfidence = true;
    theme = { value: 'system', label: 'System' };
    compactView = false;
    saveSettings();
  }
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold">Settings</h1>
    <p class="text-muted-foreground mt-2">
      Configure your preferences and view system information
    </p>
  </div>

  <div class="grid gap-6 lg:grid-cols-3">
    <!-- Settings -->
    <div class="lg:col-span-2 space-y-6">
      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Search class="h-5 w-5" />
            Search Settings
          </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-6">
          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label for="searchLimit">Default Results Limit</Label>
              <Input
                id="searchLimit"
                type="number"
                bind:value={searchLimit}
                min="10"
                max="100"
                onchange={saveSettings}
              />
              <p class="text-xs text-muted-foreground">
                Number of results per page (10-100)
              </p>
            </div>

            <div class="space-y-2">
              <Label>Default Search Mode</Label>
              <Select.Root bind:selected={defaultSearchMode} onSelectedChange={saveSettings}>
                <Select.Trigger class="w-full">
                  <Select.Value placeholder="Select mode" />
                </Select.Trigger>
                <Select.Content>
                  {#each searchModes as mode}
                    <Select.Item value={mode.value} label={mode.label}>
                      {mode.label}
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </div>
          </div>

          <div class="flex items-center justify-between">
            <div class="flex-1">
              <Label for="showConfidence">Show Confidence Scores</Label>
              <p class="text-xs text-muted-foreground">
                Display confidence levels for search results
              </p>
            </div>
            <input
              type="checkbox"
              id="showConfidence"
              bind:checked={showConfidence}
              onchange={saveSettings}
              class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
            />
          </div>
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Palette class="h-5 w-5" />
            Display Settings
          </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-6">
          <div class="space-y-2">
            <Label>Theme</Label>
            <Select.Root bind:selected={theme} onSelectedChange={saveSettings}>
              <Select.Trigger class="w-[200px]">
                <Select.Value placeholder="Select theme" />
              </Select.Trigger>
              <Select.Content>
                {#each themes as t}
                  <Select.Item value={t.value} label={t.label}>
                    {t.label}
                  </Select.Item>
                {/each}
              </Select.Content>
            </Select.Root>
          </div>

          <div class="flex items-center justify-between">
            <div class="flex-1">
              <Label for="compactView">Compact View</Label>
              <p class="text-xs text-muted-foreground">
                Reduce spacing in search results
              </p>
            </div>
            <input
              type="checkbox"
              id="compactView"
              bind:checked={compactView}
              onchange={saveSettings}
              class="h-4 w-4 rounded border-gray-300 text-primary focus:ring-primary"
            />
          </div>

          <div class="pt-4 border-t">
            <Button variant="outline" onclick={resetSettings}>
              Reset to Defaults
            </Button>
          </div>
        </Card.Content>
      </Card.Root>
    </div>

    <!-- System Info -->
    <div class="space-y-4">
      <Card.Root>
        <Card.Header>
          <Card.Title class="flex items-center gap-2">
            <Database class="h-5 w-5" />
            System Status
          </Card.Title>
        </Card.Header>
        <Card.Content class="space-y-4">
          {#if loading}
            <p class="text-sm text-muted-foreground">Loading...</p>
          {:else if health}
            <div class="space-y-3">
              <div class="flex items-center justify-between">
                <span class="text-sm">Database</span>
                <Badge variant={health.database === 'healthy' ? 'default' : 'destructive'}>
                  {#if health.database === 'healthy'}
                    <CheckCircle2 class="h-3 w-3 mr-1" />
                  {:else}
                    <AlertCircle class="h-3 w-3 mr-1" />
                  {/if}
                  {health.database}
                </Badge>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-sm">Search Index</span>
                <Badge variant={health.search_index === 'healthy' ? 'default' : 'destructive'}>
                  {#if health.search_index === 'healthy'}
                    <CheckCircle2 class="h-3 w-3 mr-1" />
                  {:else}
                    <AlertCircle class="h-3 w-3 mr-1" />
                  {/if}
                  {health.search_index}
                </Badge>
              </div>
              <div class="flex items-center justify-between">
                <span class="text-sm">API Status</span>
                <Badge variant="default">
                  <CheckCircle2 class="h-3 w-3 mr-1" />
                  {health.status}
                </Badge>
              </div>
            </div>
          {:else}
            <p class="text-sm text-muted-foreground">Unable to load status</p>
          {/if}
        </Card.Content>
      </Card.Root>

      {#if stats}
        <Card.Root>
          <Card.Header>
            <Card.Title>Database Info</Card.Title>
          </Card.Header>
          <Card.Content class="space-y-2 text-sm">
            <div class="flex justify-between">
              <span class="text-muted-foreground">Total Records</span>
              <span class="font-medium">{stats.total_documents?.toLocaleString() ?? 'N/A'}</span>
            </div>
            <div class="flex justify-between">
              <span class="text-muted-foreground">Indexed</span>
              <span class="font-medium">{stats.indexed_documents?.toLocaleString() ?? 'N/A'}</span>
            </div>
            {#if stats.last_updated}
              <div class="flex justify-between">
                <span class="text-muted-foreground">Last Updated</span>
                <span class="font-medium">{new Date(stats.last_updated).toLocaleDateString()}</span>
              </div>
            {/if}
          </Card.Content>
        </Card.Root>
      {/if}

      <Card.Root>
        <Card.Header>
          <Card.Title>About SRAKE</Card.Title>
        </Card.Header>
        <Card.Content class="text-sm text-muted-foreground space-y-2">
          <p>SRAKE (SRA Knowledge Engine) is a comprehensive tool for searching and analyzing SRA metadata.</p>
          <p>Version: 0.0.1-alpha</p>
        </Card.Content>
      </Card.Root>
    </div>
  </div>
</div>
