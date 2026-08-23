<script lang="ts">
  import { LEVEL_LABEL } from '../levels'
  import type { Level } from '../api'
  let { level, dot = true }: { level: Level | string; dot?: boolean } = $props()
  const l = $derived((level in LEVEL_LABEL ? level : 'info') as Level)
</script>

<span class="badge {l}">
  {#if dot}<span class="dot"></span>{/if}
  {LEVEL_LABEL[l]}
</span>

<style>
  .badge {
    display: inline-flex;
    align-items: center;
    gap: 7px;
    font: var(--up-type-small);
    white-space: nowrap;
  }
  .dot { width: 8px; height: 8px; border-radius: 50%; background: currentColor; flex-shrink: 0; }
  .info { color: var(--boop-info-strong); }
  .success { color: var(--boop-success-strong); }
  .warning { color: var(--boop-warning-strong); }
  .error { color: var(--boop-error-strong); }
  .critical { color: var(--boop-critical-strong); }
</style>
