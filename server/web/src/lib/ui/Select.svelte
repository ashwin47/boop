<script lang="ts">
  import type { HTMLSelectAttributes } from 'svelte/elements'
  interface Props extends HTMLSelectAttributes {
    value?: string
    options: { value: string; label: string }[]
  }
  let { value = $bindable(''), options, ...rest }: Props = $props()
</script>

<span class="wrap">
  <select bind:value {...rest}>
    {#each options as o (o.value)}
      <option value={o.value}>{o.label}</option>
    {/each}
  </select>
  <span class="chev" aria-hidden="true">▾</span>
</span>

<style>
  .wrap { position: relative; display: inline-block; }
  select {
    font: var(--up-type-meta);
    color: var(--up-ink);
    background: var(--up-bg);
    height: 34px;
    padding: 0 30px 0 12px;
    border-radius: var(--up-radius-control);
    border: 1px solid var(--up-border-control);
    outline: none;
    appearance: none;
    -webkit-appearance: none;
    cursor: pointer;
    width: 100%;
    transition: border-color 120ms ease-out;
  }
  select:focus { border-color: var(--up-accent-line); }
  .chev {
    position: absolute;
    right: 11px;
    top: 50%;
    transform: translateY(-54%);
    pointer-events: none;
    color: var(--up-text-muted);
    font-size: 10px;
  }
</style>
