<script lang="ts">
  import type { Snippet } from 'svelte'
  let { tone = 'info', children }: { tone?: 'info' | 'warn' | 'bad' | 'good'; children: Snippet } = $props()
</script>

<div class="notice {tone}">{@render children()}</div>

<style>
  .notice {
    font: var(--up-type-meta);
    line-height: 1.6;
    padding: 12px 14px;
    border-radius: var(--up-radius-tooltip);
    box-shadow: var(--up-ring-inset);
  }
  .info { background: var(--up-accent-tint); color: var(--up-accent-hover); }
  .warn { background: var(--boop-warning); color: var(--up-degraded-strong); }
  .bad { background: var(--boop-error); color: var(--up-outage-strong); }
  .good { background: var(--boop-success); color: #2F8F6E; }
</style>
