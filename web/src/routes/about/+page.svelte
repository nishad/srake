<script lang="ts">
  import * as Card from '$lib/components/ui/card';
</script>

<div class="space-y-6">
  <div>
    <h1 class="text-3xl font-bold">About SRAKE</h1>
    <p class="text-muted-foreground mt-2">
      SRA Metadata Processor
    </p>
  </div>

  <Card.Root>
    <Card.Header>
      <Card.Title>Version 0.0.3-alpha</Card.Title>
    </Card.Header>
    <Card.Content class="space-y-4">
      <p>
        SRAKE is a comprehensive tool for processing and searching SRA (Sequence Read Archive) metadata.
      </p>
      <p>
        It provides advanced search capabilities including full-text search, semantic search, and hybrid search modes.
      </p>
      <div class="pt-4 border-t">
        <p class="text-sm text-muted-foreground">
          Built with Go, SvelteKit, and modern web technologies.
        </p>
      </div>
    </Card.Content>
  </Card.Root>
</div>
