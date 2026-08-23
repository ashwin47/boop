<script lang="ts">
  import { link } from '../router.svelte'
  let { items, active }: { items: { label: string; href: string }[]; active: string } = $props()
</script>

<nav>
  {#each items as it (it.href)}
    <a href={it.href} class:active={it.href === active} onclick={link}>{it.label}</a>
  {/each}
</nav>

<style>
  nav {
    display: flex;
    gap: var(--up-space-5);
    border-bottom: 1px solid var(--up-border-hairline);
    overflow-x: auto;
  }
  a {
    font: var(--up-type-ui);
    padding: 0 0 10px;
    color: var(--up-text-muted);
    white-space: nowrap;
    transition: color 120ms ease-out;
  }
  a:hover { color: var(--up-ink); }
  a.active { color: var(--up-ink); box-shadow: inset 0 -2px 0 var(--up-accent); }
</style>
