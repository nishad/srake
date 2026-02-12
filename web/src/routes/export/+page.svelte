<script lang="ts">
  import { page } from '$app/stores';
  import { ApiService } from '$lib/api';
  import * as Card from '$lib/components/ui/card';
  import { Button } from '$lib/components/ui/button';
  import { Input } from '$lib/components/ui/input';
  import { Label } from '$lib/components/ui/label';
  import * as Select from '$lib/components/ui/select';
  import { Badge } from '$lib/components/ui/badge';
  import { Download, FileJson, FileSpreadsheet, FileText, AlertCircle, CheckCircle2 } from 'lucide-svelte';

  let query = $state('');
  let format = $state<{ value: string; label: string }>({ value: 'json', label: 'JSON' });
  let limit = $state(1000);
  let exporting = $state(false);
  let exportResult = $state<{ success: boolean; message: string } | null>(null);

  const formats = [
    { value: 'json', label: 'JSON', description: 'JavaScript Object Notation - structured data format' },
    { value: 'csv', label: 'CSV', description: 'Comma-Separated Values - spreadsheet compatible' },
    { value: 'tsv', label: 'TSV', description: 'Tab-Separated Values - analysis tools compatible' }
  ];

  async function handleExport() {
    if (!query.trim()) {
      exportResult = { success: false, message: 'Please enter a search query' };
      return;
    }

    exporting = true;
    exportResult = null;

    try {
      const searchParams = {
        query: query.trim(),
        limit,
        searchMode: 'database' as const
      };

      const exportOptions = {
        format: format.value as 'json' | 'csv' | 'tsv'
      };

      const blob = await ApiService.exportResults(searchParams, exportOptions);

      // Create download link
      const url = window.URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `srake-export-${new Date().toISOString().slice(0, 10)}.${format.value}`;
      document.body.appendChild(a);
      a.click();
      window.URL.revokeObjectURL(url);
      document.body.removeChild(a);

      exportResult = { success: true, message: `Successfully exported ${limit} results in ${format.label} format` };
    } catch (e) {
      exportResult = {
        success: false,
        message: e instanceof Error ? e.message : 'Export failed'
      };
    } finally {
      exporting = false;
    }
  }

  // Pre-populate query from URL if present
  $effect(() => {
    const urlQuery = $page.url.searchParams.get('query');
    if (urlQuery) {
      query = urlQuery;
    }
  });
</script>

<div class="space-y-8">
  <div>
    <h1 class="text-3xl font-bold">Export Data</h1>
    <p class="text-muted-foreground mt-2">
      Download search results in various formats
    </p>
  </div>

  <div class="grid gap-6 lg:grid-cols-3">
    <!-- Export Form -->
    <div class="lg:col-span-2">
      <Card.Root>
        <Card.Header>
          <Card.Title>Export Configuration</Card.Title>
          <Card.Description>
            Configure your export settings and download results
          </Card.Description>
        </Card.Header>
        <Card.Content class="space-y-6">
          <div class="space-y-2">
            <Label for="query">Search Query</Label>
            <Input
              id="query"
              bind:value={query}
              placeholder="Enter search terms (e.g., 'RNA-seq human cancer')"
            />
            <p class="text-xs text-muted-foreground">
              Results matching this query will be exported
            </p>
          </div>

          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-2">
              <Label>Export Format</Label>
              <Select.Root bind:selected={format}>
                <Select.Trigger class="w-full">
                  <Select.Value placeholder="Select format" />
                </Select.Trigger>
                <Select.Content>
                  {#each formats as fmt}
                    <Select.Item value={fmt.value} label={fmt.label}>
                      {fmt.label}
                    </Select.Item>
                  {/each}
                </Select.Content>
              </Select.Root>
            </div>

            <div class="space-y-2">
              <Label for="limit">Maximum Results</Label>
              <Input
                id="limit"
                type="number"
                bind:value={limit}
                min="1"
                max="100000"
              />
            </div>
          </div>

          {#if exportResult}
            <div class="rounded-md p-4 {exportResult.success ? 'bg-green-50 border border-green-200' : 'bg-red-50 border border-red-200'}">
              <div class="flex items-center gap-2">
                {#if exportResult.success}
                  <CheckCircle2 class="h-5 w-5 text-green-600" />
                  <span class="text-green-800">{exportResult.message}</span>
                {:else}
                  <AlertCircle class="h-5 w-5 text-red-600" />
                  <span class="text-red-800">{exportResult.message}</span>
                {/if}
              </div>
            </div>
          {/if}

          <Button onclick={handleExport} disabled={exporting} class="w-full">
            {#if exporting}
              <span class="mr-2 animate-spin">Loading...</span>
              Exporting...
            {:else}
              <Download class="mr-2 h-4 w-4" />
              Export Data
            {/if}
          </Button>
        </Card.Content>
      </Card.Root>
    </div>

    <!-- Format Info -->
    <div class="space-y-4">
      <Card.Root>
        <Card.Header>
          <Card.Title>Available Formats</Card.Title>
        </Card.Header>
        <Card.Content class="space-y-4">
          {#each formats as fmt}
            <div class="flex gap-3 p-3 rounded-lg border {format.value === fmt.value ? 'border-primary bg-primary/5' : 'border-border'}">
              {#if fmt.value === 'json'}
                <FileJson class="h-5 w-5 mt-0.5 {format.value === fmt.value ? 'text-primary' : 'text-muted-foreground'}" />
              {:else if fmt.value === 'csv'}
                <FileSpreadsheet class="h-5 w-5 mt-0.5 {format.value === fmt.value ? 'text-primary' : 'text-muted-foreground'}" />
              {:else}
                <FileText class="h-5 w-5 mt-0.5 {format.value === fmt.value ? 'text-primary' : 'text-muted-foreground'}" />
              {/if}
              <div>
                <div class="font-medium flex items-center gap-2">
                  {fmt.label}
                  {#if format.value === fmt.value}
                    <Badge variant="secondary" class="text-xs">Selected</Badge>
                  {/if}
                </div>
                <p class="text-sm text-muted-foreground">{fmt.description}</p>
              </div>
            </div>
          {/each}
        </Card.Content>
      </Card.Root>

      <Card.Root>
        <Card.Header>
          <Card.Title>Export Tips</Card.Title>
        </Card.Header>
        <Card.Content class="text-sm text-muted-foreground space-y-2">
          <p>Use JSON for programmatic access and data analysis.</p>
          <p>CSV works best with Excel and Google Sheets.</p>
          <p>TSV is compatible with most bioinformatics tools.</p>
        </Card.Content>
      </Card.Root>
    </div>
  </div>
</div>
