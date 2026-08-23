<script lang="ts">
  import type { Snippet } from 'svelte'
  let { label, hint, children }: { label: string; hint?: string; children: Snippet } = $props()
</script>

<div class="row">
  <div class="text">
    <div class="label">{label}</div>
    {#if hint}<div class="hint">{hint}</div>{/if}
  </div>
  <div class="control">{@render children()}</div>
</div>

<style>
  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--up-space-4);
    padding: 14px 0;
    border-top: 1px solid var(--up-border-hairline);
  }
  .label { font: var(--up-type-setting); color: var(--up-ink); }
  .hint { font: var(--up-type-caption); color: var(--up-text-muted); margin-top: 3px; line-height: 1.5; }
  .control { flex-shrink: 0; }
  @media (max-width: 520px) {
    .row { flex-direction: column; align-items: stretch; }
  }
</style>
