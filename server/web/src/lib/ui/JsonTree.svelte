<script lang="ts">
  // Expandable JSON viewer for unrecognised event data.
  import JsonTree from './JsonTree.svelte'
  let { value, name, depth = 0, open = depth < 1 }: { value: unknown; name?: string; depth?: number; open?: boolean } = $props()
  // svelte-ignore state_referenced_locally
  let expanded = $state(open)
  const isObj = $derived(value !== null && typeof value === 'object')
  const entries = $derived(isObj ? (Array.isArray(value) ? value.map((v, i) => [String(i), v] as const) : Object.entries(value as object)) : [])
  const summary = $derived(Array.isArray(value) ? `[${(value as unknown[]).length}]` : isObj ? `{${entries.length}}` : '')
</script>

<div class="node" style="--d: {depth}">
  {#if isObj}
    <button type="button" class="line" onclick={() => (expanded = !expanded)}>
      <span class="chev" class:open={expanded}>▸</span>
      {#if name !== undefined}<span class="key">{name}</span><span class="colon">:</span>{/if}
      <span class="summary">{summary}</span>
    </button>
    {#if expanded}
      {#each entries as [k, v] (k)}
        <JsonTree value={v} name={k} depth={depth + 1} />
      {/each}
    {/if}
  {:else}
    <div class="line leaf">
      {#if name !== undefined}<span class="key">{name}</span><span class="colon">:</span>{/if}
      <span class="val {typeof value}" class:null={value === null}>{value === null ? 'null' : typeof value === 'string' ? JSON.stringify(value) : String(value)}</span>
    </div>
  {/if}
</div>

<style>
  .node { font: var(--up-type-code); }
  .line {
    display: flex;
    align-items: baseline;
    gap: 6px;
    padding: 2px 0 2px calc(var(--d) * 16px);
    background: none;
    border: none;
    text-align: left;
    width: 100%;
    color: var(--up-ink);
    cursor: pointer;
    border-radius: var(--up-radius-cell);
    font: inherit;
  }
  .leaf { cursor: default; }
  button.line:hover { background: var(--up-bg-hover); }
  .chev { color: var(--up-text-inactive); width: 10px; display: inline-block; font-size: 10px; }
  .chev.open { transform: rotate(90deg); display: inline-block; }
  .key { color: var(--up-text-secondary); }
  .colon { color: var(--up-text-faint); }
  .summary { color: var(--up-text-muted); }
  .val { word-break: break-all; white-space: pre-wrap; }
  .val.string { color: var(--up-ink); }
  .val.number, .val.boolean { color: var(--up-accent-hover); }
  .val.null { color: var(--up-text-muted); }
</style>
