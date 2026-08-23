<script lang="ts">
  import { link } from '../router.svelte'
  let { items, active }: { items: { label: string; href: string }[]; active: string } = $props()
  let nav = $state<HTMLElement | null>(null)
  let links = $state<Record<string, HTMLElement>>({})
  let indicator = $state({ left: 0, width: 0, ready: false })

  // Sliding underline: measure the active link and move the indicator to it.
  $effect(() => {
    const el = links[active]
    if (!el || !nav) return
    const measure = () => {
      indicator = { left: el.offsetLeft, width: el.offsetWidth, ready: true }
    }
    measure()
    if (document.fonts?.ready) document.fonts.ready.then(measure)
    const ro = new ResizeObserver(measure)
    ro.observe(el)
    return () => ro.disconnect()
  })
</script>

<nav bind:this={nav}>
  {#each items as it (it.href)}
    <a href={it.href} class:active={it.href === active} onclick={link} bind:this={links[it.href]}>{it.label}</a>
  {/each}
  <span class="indicator" class:ready={indicator.ready} style="left: {indicator.left}px; width: {indicator.width}px"></span>
</nav>

<style>
  nav {
    position: relative;
    display: flex;
    gap: var(--up-space-5);
    border-bottom: 1px solid var(--up-border-hairline);
    overflow: visible;
  }
  a {
    font: var(--up-type-ui);
    padding: 0 0 10px;
    color: var(--up-text-muted);
    white-space: nowrap;
    transition: color 120ms ease-out;
  }
  a:hover { color: var(--up-ink); }
  a.active { color: var(--up-ink); }
  .indicator {
    position: absolute;
    bottom: -1px;
    height: 2px;
    pointer-events: none;
    background: var(--up-accent);
    opacity: 0;
    transition: left 200ms cubic-bezier(0.2, 0, 0, 1), width 200ms cubic-bezier(0.2, 0, 0, 1), opacity 120ms ease-out;
  }
  .indicator.ready { opacity: 1; }
  @media (prefers-reduced-motion: reduce) {
    .indicator { transition: none; }
  }
</style>
