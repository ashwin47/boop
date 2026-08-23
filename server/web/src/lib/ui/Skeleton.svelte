<script lang="ts">
  // Placeholder blocks shown while data loads. `lines` draws stacked bars; `rows` draws list rows.
  let { lines = 0, rows = 0, height = 12, width = '100%', widths = [] as string[] }: { lines?: number; rows?: number; height?: number; width?: string; widths?: string[] } = $props()
  const lineWidths = $derived(Array.from({ length: lines }, (_, i) => widths[i] ?? (i === lines - 1 && lines > 1 ? '60%' : '100%')))
</script>

{#if rows > 0}
  <div class="rows" aria-hidden="true">
    {#each Array(rows) as _, i (i)}
      <div class="row">
        <span class="bar" style="width: 90px; height: 12px"></span>
        <span class="main">
          <span class="bar" style="width: {55 + ((i * 17) % 30)}%; height: 14px"></span>
          <span class="bar" style="width: {35 + ((i * 23) % 40)}%; height: 11px"></span>
        </span>
        <span class="bar" style="width: 56px; height: 11px"></span>
        <span class="bar" style="width: 44px; height: 10px"></span>
      </div>
    {/each}
  </div>
{:else if lines > 0}
  <div class="lines" aria-hidden="true" style="width: {width}">
    {#each lineWidths as w, i (i)}
      <span class="bar" style="width: {w}; height: {height}px"></span>
    {/each}
  </div>
{:else}
  <span class="bar" aria-hidden="true" style="width: {width}; height: {height}px"></span>
{/if}

<style>
  .bar {
    display: block;
    border-radius: var(--up-radius-cell);
    background: linear-gradient(90deg, var(--up-border-hairline) 0%, var(--up-bg-hover) 45%, var(--up-border-hairline) 90%);
    background-size: 240% 100%;
    animation: shimmer 1.4s ease-in-out infinite;
  }
  .lines { display: flex; flex-direction: column; gap: 8px; }
  .rows { display: flex; flex-direction: column; }
  .row { display: grid; grid-template-columns: 120px 1fr auto 64px; gap: var(--up-space-4); align-items: center; padding: 14px var(--up-space-4); }
  .main { display: flex; flex-direction: column; gap: 7px; }
  @keyframes shimmer { from { background-position: 120% 0; } to { background-position: -120% 0; } }
  @media (prefers-reduced-motion: reduce) { .bar { animation: none; } }
  @media (max-width: 600px) { .row { grid-template-columns: 1fr 64px; } .row > :nth-child(1), .row > :nth-child(3) { display: none; } }
</style>
