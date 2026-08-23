<script lang="ts">
  let { code, copyable = true, wrap = false }: { code: string; copyable?: boolean; wrap?: boolean } = $props()
  let copied = $state(false)
  function copy() {
    navigator.clipboard?.writeText(code)
    copied = true
    setTimeout(() => (copied = false), 1600)
  }
</script>

<div class="block">
  <code class:wrap>{code}</code>
  {#if copyable}
    <button type="button" onclick={copy} class:copied>{copied ? 'Copied' : 'Copy'}</button>
  {/if}
</div>

<style>
  .block {
    background: var(--up-surface-dark);
    border-radius: var(--up-radius-tooltip);
    padding: 14px 16px;
    display: flex;
    align-items: flex-start;
    gap: var(--up-space-4);
    min-width: 0;
  }
  code {
    font: var(--up-type-code);
    color: var(--up-text-on-dark);
    flex: 1;
    min-width: 0;
    overflow-x: auto;
    white-space: pre;
  }
  code.wrap { white-space: pre-wrap; word-break: break-all; }
  button {
    font: var(--up-type-small);
    color: var(--up-text-on-dark-muted);
    background: none;
    border: 1px solid var(--up-divider-on-dark);
    border-radius: var(--up-radius-control);
    padding: 5px 10px;
    cursor: pointer;
    flex-shrink: 0;
    transition: color 120ms ease-out;
  }
  button:hover, button.copied { color: var(--up-text-on-dark); }
</style>
