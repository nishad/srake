<script lang="ts">
  import '../app.css';
  import { page } from '$app/stores';
  import { ModeWatcher, toggleMode } from 'mode-watcher';
  import { Toaster } from '$lib/components/ui/sonner';
  import { Button } from '$lib/components/ui/button';
  import { Separator } from '$lib/components/ui/separator';
  import * as Sheet from '$lib/components/ui/sheet';
  import * as Tooltip from '$lib/components/ui/tooltip';
  import {
    Search,
    Home,
    Database,
    FileText,
    Download,
    BarChart3,
    Settings,
    Info,
    Sun,
    Moon,
    Menu
  } from 'lucide-svelte';

  let { children } = $props();
  let mobileOpen = $state(false);

  const navLinks = [
    { href: '/', label: 'Dashboard', icon: Home },
    { href: '/search', label: 'Search', icon: Search },
    { href: '/browse', label: 'Browse', icon: FileText },
    { href: '/export', label: 'Export', icon: Download },
    { href: '/stats', label: 'Statistics', icon: BarChart3 },
  ];

  function isActive(href: string): boolean {
    if (href === '/') return $page.url.pathname === '/';
    return $page.url.pathname.startsWith(href);
  }

  function closeMobile() {
    mobileOpen = false;
  }
</script>

<svelte:head>
  <title>SRAKE - SRA Knowledge Engine</title>
</svelte:head>

<ModeWatcher />
<Toaster />

<Tooltip.Provider>
  <div class="min-h-screen bg-background flex flex-col">
    <!-- Header -->
    <header class="sticky top-0 z-50 w-full border-b bg-background/95 backdrop-blur supports-[backdrop-filter]:bg-background/60">
      <div class="container mx-auto px-4 flex h-14 items-center">
        <!-- Mobile menu button -->
        <Sheet.Root bind:open={mobileOpen}>
          <Sheet.Trigger class="mr-2 md:hidden">
            <Button variant="ghost" size="icon" class="md:hidden">
              <Menu class="h-5 w-5" />
              <span class="sr-only">Toggle menu</span>
            </Button>
          </Sheet.Trigger>
          <Sheet.Content side="left" class="w-72">
            <Sheet.Header>
              <Sheet.Title class="flex items-center gap-2">
                <Database class="h-5 w-5" />
                SRAKE
              </Sheet.Title>
              <Sheet.Description>SRA Knowledge Engine</Sheet.Description>
            </Sheet.Header>
            <nav class="grid gap-1 py-4">
              {#each navLinks as link}
                {@const Icon = link.icon}
                <a
                  href={link.href}
                  onclick={closeMobile}
                  class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
                  class:bg-accent={isActive(link.href)}
                  class:text-accent-foreground={isActive(link.href)}
                >
                  <Icon class="h-4 w-4" />
                  {link.label}
                </a>
              {/each}
              <Separator class="my-2" />
              <a
                href="/settings"
                onclick={closeMobile}
                class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
                class:bg-accent={isActive('/settings')}
              >
                <Settings class="h-4 w-4" />
                Settings
              </a>
              <a
                href="/about"
                onclick={closeMobile}
                class="flex items-center gap-3 rounded-lg px-3 py-2 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
                class:bg-accent={isActive('/about')}
              >
                <Info class="h-4 w-4" />
                About
              </a>
            </nav>
          </Sheet.Content>
        </Sheet.Root>

        <!-- Logo -->
        <a href="/" class="flex items-center space-x-2 mr-6">
          <Database class="h-5 w-5" />
          <span class="text-lg font-bold hidden sm:inline">SRAKE</span>
        </a>

        <!-- Desktop nav -->
        <nav class="hidden md:flex items-center space-x-1 flex-1">
          {#each navLinks as link}
            {@const Icon = link.icon}
            <a
              href={link.href}
              class="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors hover:bg-accent hover:text-accent-foreground"
              class:bg-accent={isActive(link.href)}
              class:text-accent-foreground={isActive(link.href)}
              class:text-muted-foreground={!isActive(link.href)}
            >
              <Icon class="h-4 w-4" />
              {link.label}
            </a>
          {/each}
        </nav>

        <!-- Right side actions -->
        <div class="ml-auto flex items-center space-x-1">
          <Tooltip.Root>
            <Tooltip.Trigger>
              <Button variant="ghost" size="icon" onclick={toggleMode}>
                <Sun class="h-4 w-4 rotate-0 scale-100 transition-all dark:-rotate-90 dark:scale-0" />
                <Moon class="absolute h-4 w-4 rotate-90 scale-0 transition-all dark:rotate-0 dark:scale-100" />
                <span class="sr-only">Toggle theme</span>
              </Button>
            </Tooltip.Trigger>
            <Tooltip.Content>Toggle theme</Tooltip.Content>
          </Tooltip.Root>

          <Tooltip.Root>
            <Tooltip.Trigger>
              <a href="/settings">
                <Button variant="ghost" size="icon">
                  <Settings class="h-4 w-4" />
                  <span class="sr-only">Settings</span>
                </Button>
              </a>
            </Tooltip.Trigger>
            <Tooltip.Content>Settings</Tooltip.Content>
          </Tooltip.Root>

          <Tooltip.Root>
            <Tooltip.Trigger>
              <a href="/about">
                <Button variant="ghost" size="icon">
                  <Info class="h-4 w-4" />
                  <span class="sr-only">About</span>
                </Button>
              </a>
            </Tooltip.Trigger>
            <Tooltip.Content>About</Tooltip.Content>
          </Tooltip.Root>
        </div>
      </div>
    </header>

    <!-- Main content -->
    <main class="container mx-auto px-4 py-6 flex-1">
      {@render children?.()}
    </main>

    <!-- Footer -->
    <footer class="border-t">
      <div class="container mx-auto px-4 py-3">
        <p class="text-xs text-muted-foreground text-center">
          SRAKE &middot; SRA Knowledge Engine
        </p>
      </div>
    </footer>
  </div>
</Tooltip.Provider>

<style>
  :global(html) {
    font-family: 'Inter', system-ui, sans-serif;
  }
</style>
