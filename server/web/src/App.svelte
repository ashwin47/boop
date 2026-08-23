<script lang="ts">
  import { api } from './lib/api'
  import { router, link } from './lib/router.svelte'
  import Tabs from './lib/ui/Tabs.svelte'
  import StatusDot from './lib/ui/StatusDot.svelte'
  import Inbox from './pages/Inbox.svelte'
  import EventDetail from './pages/EventDetail.svelte'
  import Projects from './pages/Projects.svelte'
  import Devices from './pages/Devices.svelte'
  import Settings from './pages/Settings.svelte'
  import Setup from './pages/Setup.svelte'

  let setupCompleted = $state<boolean | null>(null)
  let apnsConfigured = $state<boolean | null>(null)
  let unreachable = $state(false)

  async function boot() {
    try {
      const s = await api.status()
      setupCompleted = s.setup_completed
      apnsConfigured = s.apns.configured
      unreachable = false
      if (!s.setup_completed && router.route.name !== 'setup') router.navigate('/setup', true)
    } catch {
      unreachable = true
    }
  }
  $effect(() => {
    boot()
    const t = setInterval(boot, 60_000)
    return () => clearInterval(t)
  })

  const route = $derived(router.route)
  const tabs = [
    { label: 'Inbox', href: '/' },
    { label: 'Projects', href: '/projects' },
    { label: 'Devices', href: '/devices' },
    { label: 'Settings', href: '/settings' },
  ]
  const activeTab = $derived(route.name === 'event' ? '/' : router.path)
</script>

<div class="page">
  <header>
    <a class="brand" href="/" onclick={link}>
      <span class="mark"></span>
      <span class="wordmark">Boop</span>
    </a>
    <div class="right">
      {#if unreachable}
        <StatusDot tone="bad">Server unreachable</StatusDot>
      {:else if apnsConfigured === false}
        <a href="/settings" onclick={link} class="plain"><StatusDot tone="warn">APNs not configured</StatusDot></a>
      {:else if apnsConfigured}
        <StatusDot tone="ok">Push ready</StatusDot>
      {/if}
    </div>
  </header>

  {#if route.name === 'setup'}
    <Setup onfinished={() => (setupCompleted = true)} />
  {:else}
    <div class="tabs"><Tabs items={tabs} active={activeTab} /></div>
    {#if route.name === 'inbox'}
      <Inbox />
    {:else if route.name === 'event'}
      {#key route.params.id}<EventDetail id={route.params.id} />{/key}
    {:else if route.name === 'projects'}
      <Projects />
    {:else if route.name === 'devices'}
      <Devices />
    {:else if route.name === 'settings'}
      <Settings />
    {:else}
      <p class="muted">Nothing here. <a href="/" onclick={link}>Back to the inbox</a>.</p>
    {/if}
  {/if}

  <footer class="caption faint">Boop · self-hosted · <a href="/settings" onclick={link}>status</a></footer>
</div>

<style>
  header { display: flex; align-items: center; justify-content: space-between; padding: 22px 0; }
  .brand { display: inline-flex; align-items: center; gap: 10px; color: var(--up-ink); }
  .brand:hover { color: var(--up-ink); }
  .mark { width: 22px; height: 22px; border-radius: var(--up-radius-pill); background: var(--up-accent); position: relative; }
  .mark::after { content: ''; position: absolute; inset: 7px; border-radius: 50%; background: var(--up-bg); }
  .wordmark { font: var(--up-type-wordmark); letter-spacing: 0.02em; }
  .right { display: flex; align-items: center; gap: 18px; }
  .plain { color: inherit; }
  .tabs { margin-bottom: var(--up-space-5); }
  footer { margin-top: var(--up-space-6); padding-top: var(--up-space-4); border-top: 1px solid var(--up-border-hairline); }
</style>
