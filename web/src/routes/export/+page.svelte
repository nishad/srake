<script lang="ts">
  import { page } from '$app/stores';
  import { ApiService } from '$lib/api';
  import type { ExportRequest } from '$lib/utils';
  import * as Card from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import { Badge } from '$lib/components/ui/badge';
  import {
    Download,
    FileJson,
    FileSpreadsheet,
    FileText,
    FileCode,
    AlertCircle,
    CheckCircle2
  } from 'lucide-svelte';

  let query = $state('');
  let selectedFormat = $state('json');
  let limit = $state(1000);
  let exporting = $state(false);
  let exportResult = $state<{ success: boolean; message: string } | null>(null);

  const formats = [
    { value: 'json', label: 'JSON', description: 'JavaScript Object Notation', icon: FileJson },
    { value: 'csv', label: 'CSV', description: 'Comma-Separated Values', icon: FileSpreadsheet },
    { value: 'tsv', label: 'TSV', description: 'Tab-Separated Values', icon: FileText },
    { value: 'xml', label: 'XML', description: 'Extensible Markup Language', icon: FileCode },
    { value: 'jsonl', label: 'JSONL', description: 'JSON Lines (one record per line)', icon: FileJson },
  ];

  async function handleExport() {
    if (!query.trim()) {
      exportResult = { success: false, message: 'Please enter a search query' };
      return;
    }
    exporting = true;
    exportResult = null;

    try {
      const req: ExportRequest = {
        query: query.trim(),
        format: selectedFormat,
        limit,
      };

      const blob = await ApiService.exportResults(req);
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `srake-export-${new Date().toISOString().slice(0, 10)}.${selectedFormat}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      exportResult = { success: true, message: `Exported in ${selectedFormat.toUpperCase()} format` };
    } catch (e) {
      exportResult = { success: false, message: e instanceof Error ? e.message : 'Export failed' };
    } finally {
      exporting = false;
    }
  }

  // Pre-populate from URL
  $effect(() => {
    const urlQuery = $page.url.searchParams.get('q') || $page.url.searchParams.get('query');
    if (urlQuery) query = urlQuery;
  });
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold tracking-tight">Export Data</h1>
    <p class="text-muted-foreground mt-1">Download search results in various formats</p>
  </div>

  <div class="grid gap-6 lg:grid-cols-3">
    <div class="lg:col-span-2">
      <Card.Root>
        <Card.Header>
          <Card.Title>Export Configuration</Card.Title>
          <Card.Description>Search and download matching records</Card.Description>
        </Card.Header>
        <Card.Content class="space-y-6">
          <div class="space-y-2">
            <Label for="query">Search Query</Label>
            <Input id="query" bind:value={query} placeholder="Enter search terms..." />
          </div>

          <div class="space-y-2">
            <Label>Format</Label>
            <div class="grid grid-cols-3 md:grid-cols-5 gap-2">
              {#each formats as fmt}
                {@const Icon = fmt.icon}
                <button
                  onclick={() => selectedFormat = fmt.value}
                  class="rounded-lg border p-3 text-center text-sm transition-all {selectedFormat === fmt.value ? 'border-primary bg-primary/5 ring-1 ring-primary/20' : 'hover:bg-accent'}"
                >
                  <Icon class="h-5 w-5 mx-auto mb-1 {selectedFormat === fmt.value ? 'text-primary' : 'text-muted-foreground'}" />
                  <span class="text-xs font-medium">{fmt.label}</span>
                </button>
              {/each}
            </div>
          </div>

          <div class="space-y-2">
            <Label for="limit">Max Results</Label>
            <Input id="limit" type="number" bind:value={limit} min="1" max="100000" class="max-w-[200px]" />
          </div>

          {#if exportResult}
            <div class="rounded-md p-3 flex items-center gap-2 {exportResult.success ? 'bg-green-50 dark:bg-green-950 border border-green-200 dark:border-green-800' : 'bg-red-50 dark:bg-red-950 border border-red-200 dark:border-red-800'}">
              {#if exportResult.success}
                <CheckCircle2 class="h-4 w-4 text-green-600 dark:text-green-400 shrink-0" />
                <span class="text-sm text-green-800 dark:text-green-200">{exportResult.message}</span>
              {:else}
                <AlertCircle class="h-4 w-4 text-red-600 dark:text-red-400 shrink-0" />
                <span class="text-sm text-red-800 dark:text-red-200">{exportResult.message}</span>
              {/if}
            </div>
          {/if}

          <Button onclick={handleExport} disabled={exporting} class="w-full">
            {#if exporting}
              Exporting...
            {:else}
              <Download class="mr-2 h-4 w-4" /> Export Data
            {/if}
          </Button>
        </Card.Content>
      </Card.Root>
    </div>

    <div class="space-y-4">
      <Card.Root>
        <Card.Header><Card.Title>Format Guide</Card.Title></Card.Header>
        <Card.Content class="space-y-3 text-sm text-muted-foreground">
          <p><strong>JSON</strong> — Structured data for programmatic use</p>
          <p><strong>CSV</strong> — Works with Excel, Google Sheets</p>
          <p><strong>TSV</strong> — Compatible with bioinformatics tools</p>
          <p><strong>XML</strong> — Standard exchange format</p>
          <p><strong>JSONL</strong> — Streaming JSON, one record per line</p>
        </Card.Content>
      </Card.Root>
    </div>
  </div>
</div>
