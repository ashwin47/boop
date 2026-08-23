<script lang="ts">
  let { label, value, delta, tone = 'neutral' }: { label: string; value: string; delta?: string; tone?: 'good' | 'bad' | 'neutral' } = $props()
</script>

<div class="metric">
  <div class="label">{label}</div>
  <div class="row">
    <div class="value">{value}</div>
    {#if delta}<div class="delta {tone}">{delta}</div>{/if}
  </div>
</div>

<style>
  .metric { display: flex; flex-direction: column; gap: 4px; }
  .label { font: var(--up-type-meta); color: var(--up-text-muted); }
  .row { display: flex; align-items: baseline; gap: var(--up-space-2); }
  .value { font: var(--up-type-metric); letter-spacing: -0.01em; color: var(--up-ink); }
  .delta { font: var(--up-type-small); }
  .good { color: var(--up-alt-mint); }
  .bad { color: var(--up-degraded-strong); }
  .neutral { color: var(--up-text-muted); }
</style>
