<script lang="ts">
  import type { Snippet } from 'svelte'
  let { title, children }: { title: string; children?: Snippet } = $props()
</script>

<div class="empty">
  <div class="t">{title}</div>
  {#if children}<div class="b">{@render children()}</div>{/if}
</div>

<style>
  .empty { padding: var(--up-space-6) var(--up-space-4); text-align: center; }
  .t { font: var(--up-type-setting); color: var(--up-text-secondary); }
  .b { font: var(--up-type-meta); color: var(--up-text-muted); margin-top: 6px; line-height: 1.6; }
</style>
