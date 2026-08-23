<script lang="ts">
  import { ICON_SHAPES, ICON_COLORS, parseIcon } from '../icons'
  import ProjectIcon from './ProjectIcon.svelte'
  let { value, onchange }: { value: string; onchange: (v: string) => void } = $props()
  const current = $derived(parseIcon(value) ?? { shape: 'circle', color: 'periwinkle' })
</script>

<div class="picker">
  <div class="row">
    {#each ICON_SHAPES as shape (shape)}
      <button type="button" class:on={current.shape === shape} aria-label={shape} title={shape} onclick={() => onchange(`${shape}:${current.color}`)}>
        <ProjectIcon icon="{shape}:{current.color}" size={18} />
      </button>
    {/each}
  </div>
  <div class="row">
    {#each Object.entries(ICON_COLORS) as [name, hex] (name)}
      <button type="button" class="swatch" class:on={current.color === name} aria-label={name} title={name} style="--c: {hex}" onclick={() => onchange(`${current.shape}:${name}`)}></button>
    {/each}
  </div>
</div>

<style>
  .picker { display: flex; flex-direction: column; gap: 8px; align-items: flex-end; }
  .row { display: flex; gap: 6px; flex-wrap: wrap; justify-content: flex-end; }
  button {
    width: 32px; height: 32px; display: inline-flex; align-items: center; justify-content: center;
    background: var(--up-bg); border: 1px solid var(--up-border-control); border-radius: var(--up-radius-control);
    cursor: pointer; padding: 0; transition: background 120ms ease-out, border-color 120ms ease-out;
  }
  button:hover { background: var(--up-bg-hover); }
  button.on { border-color: var(--up-accent); background: var(--up-accent-tint); }
  .swatch { width: 24px; height: 24px; border-radius: var(--up-radius-pill); background: var(--c); border: 2px solid transparent; box-shadow: var(--up-ring-inset); }
  .swatch:hover { background: var(--c); }
  .swatch.on { border-color: var(--up-ink); background: var(--c); }
</style>
