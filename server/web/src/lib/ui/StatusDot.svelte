<script lang="ts">
  // Generic status line: dot + label. tone maps to the DS status pastels.
  let { tone = 'ok', children }: { tone?: 'ok' | 'warn' | 'bad' | 'muted'; children: import('svelte').Snippet } = $props()
</script>

<span class="s {tone}"><span class="dot"></span>{@render children()}</span>

<style>
  .s { display: inline-flex; align-items: center; gap: 7px; font: var(--up-type-small); white-space: nowrap; }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; flex-shrink: 0; }
  .ok { color: var(--up-alt-mint); }
  .warn { color: var(--up-degraded-strong); }
  .bad { color: var(--up-outage-strong); }
  .muted { color: var(--up-text-inactive); }
</style>
