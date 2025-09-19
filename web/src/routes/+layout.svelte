<script lang="ts">
  import '../app.css';
  import { page } from '$app/stores';
  import {
    Search,
    Home,
    Database,
    FileText,
    Download,
    BarChart3,
    Settings,
    Info
  } from 'lucide-svelte';

  let { children } = $props();
</script>

<svelte:head>
  <title>SRAKE - SRA Metadata Processor</title>
</svelte:head>

<div class="min-h-screen bg-background">
  <header class="border-b">
    <div class="container mx-auto px-4 py-3">
      <nav class="flex items-center justify-between">
        <div class="flex items-center space-x-8">
          <a href="/" class="flex items-center space-x-2">
            <Database class="h-6 w-6" />
            <span class="text-xl font-bold">SRAKE</span>
          </a>

          <div class="hidden md:flex space-x-6">
            <a
              href="/"
              class="flex items-center space-x-1 text-sm font-medium transition-colors hover:text-primary"
              class:text-primary={$page.url.pathname === '/'}
            >
              <Home class="h-4 w-4" />
              <span>Dashboard</span>
            </a>

            <a
              href="/search"
              class="flex items-center space-x-1 text-sm font-medium transition-colors hover:text-primary"
              class:text-primary={$page.url.pathname === '/search'}
            >
              <Search class="h-4 w-4" />
              <span>Search</span>
            </a>

            <a
              href="/browse"
              class="flex items-center space-x-1 text-sm font-medium transition-colors hover:text-primary"
              class:text-primary={$page.url.pathname === '/browse'}
            >
              <FileText class="h-4 w-4" />
              <span>Browse</span>
            </a>

            <a
              href="/export"
              class="flex items-center space-x-1 text-sm font-medium transition-colors hover:text-primary"
              class:text-primary={$page.url.pathname === '/export'}
            >
              <Download class="h-4 w-4" />
              <span>Export</span>
            </a>

            <a
              href="/stats"
              class="flex items-center space-x-1 text-sm font-medium transition-colors hover:text-primary"
              class:text-primary={$page.url.pathname === '/stats'}
            >
              <BarChart3 class="h-4 w-4" />
              <span>Statistics</span>
            </a>
          </div>
        </div>

        <div class="flex items-center space-x-4">
          <a
            href="/settings"
            class="text-sm font-medium transition-colors hover:text-primary"
            aria-label="Settings"
          >
            <Settings class="h-5 w-5" />
          </a>
          <a
            href="/about"
            class="text-sm font-medium transition-colors hover:text-primary"
            aria-label="About"
          >
            <Info class="h-5 w-5" />
          </a>
        </div>
      </nav>
    </div>
  </header>

  <main class="container mx-auto px-4 py-8">
    {@render children?.()}
  </main>

  <footer class="border-t mt-auto">
    <div class="container mx-auto px-4 py-4">
      <p class="text-sm text-muted-foreground text-center">
        SRAKE v0.0.3-alpha · SRA Metadata Processor
      </p>
    </div>
  </footer>
</div>

<style>
  :global(html) {
    font-family: 'Inter', system-ui, sans-serif;
  }
</style>
