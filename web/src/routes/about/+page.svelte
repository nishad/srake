<script lang="ts">
  import * as Card from '$lib/components/ui/card';
  import { Badge } from '$lib/components/ui/badge';
  import { Separator } from '$lib/components/ui/separator';
  import { Button } from '$lib/components/ui/button';
  import { Database, Search, Download, BarChart3, ExternalLink, Sparkles } from 'lucide-svelte';
</script>

<div class="space-y-8 max-w-3xl mx-auto">
  <div>
    <h1 class="text-3xl font-bold tracking-tight">About SRAKE</h1>
    <p class="text-muted-foreground mt-1">SRA Knowledge Engine</p>
  </div>

  <Card.Root>
    <Card.Content class="pt-6 space-y-4">
      <p class="text-sm leading-relaxed">
        SRAKE is a comprehensive tool for processing, searching, and analyzing metadata from the
        NCBI Sequence Read Archive (SRA). It provides fast local search across millions of
        sequencing studies, experiments, samples, and runs.
      </p>
      <Separator />
      <div class="grid gap-4 md:grid-cols-2">
        <div class="flex items-start gap-3">
          <Search class="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div>
            <p class="font-medium text-sm">Multi-Modal Search</p>
            <p class="text-xs text-muted-foreground">Full-text (FTS5), semantic vector, and hybrid search modes</p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <Sparkles class="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div>
            <p class="font-medium text-sm">Semantic Similarity</p>
            <p class="text-xs text-muted-foreground">Find related studies and samples via vector embeddings</p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <Database class="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div>
            <p class="font-medium text-sm">Comprehensive Metadata</p>
            <p class="text-xs text-muted-foreground">Studies, experiments, samples, runs with full SRA metadata</p>
          </div>
        </div>
        <div class="flex items-start gap-3">
          <Download class="h-5 w-5 text-primary shrink-0 mt-0.5" />
          <div>
            <p class="font-medium text-sm">Flexible Export</p>
            <p class="text-xs text-muted-foreground">JSON, CSV, TSV, XML, JSONL export formats</p>
          </div>
        </div>
      </div>
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header><Card.Title>Technology Stack</Card.Title></Card.Header>
    <Card.Content>
      <div class="flex flex-wrap gap-2">
        <Badge variant="secondary">Go</Badge>
        <Badge variant="secondary">SQLite + FTS5</Badge>
        <Badge variant="secondary">Bleve Search</Badge>
        <Badge variant="secondary">SvelteKit 5</Badge>
        <Badge variant="secondary">Tailwind CSS</Badge>
        <Badge variant="secondary">shadcn-svelte</Badge>
      </div>
    </Card.Content>
  </Card.Root>

  <Card.Root>
    <Card.Header><Card.Title>Links</Card.Title></Card.Header>
    <Card.Content class="space-y-2">
      <Button variant="outline" size="sm" href="https://www.ncbi.nlm.nih.gov/sra" target="_blank" class="gap-1.5">
        <ExternalLink class="h-3 w-3" /> NCBI SRA
      </Button>
      <p class="text-xs text-muted-foreground mt-3">
        SRAKE is not affiliated with NCBI. SRA data is publicly available.
      </p>
    </Card.Content>
  </Card.Root>
</div>
