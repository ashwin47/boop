<script lang="ts">
  import type { Snippet } from 'svelte'
  let { title, action, children, flush = false }: { title?: string; action?: Snippet; children: Snippet; flush?: boolean } = $props()
</script>

<section class="card" class:flush>
  {#if title || action}
    <header>
      {#if title}<h2>{title}</h2>{/if}
      {#if action}<div>{@render action()}</div>{/if}
    </header>
  {/if}
  {@render children()}
</section>

<style>
  .card {
    background: var(--up-bg);
    border: 1px solid var(--up-border-hairline);
    border-radius: var(--up-radius-card);
    padding: var(--up-space-5);
  }
  .flush { padding: var(--up-space-2); padding-top: var(--up-space-4); }
  header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--up-space-3);
    margin-bottom: var(--up-space-4);
  }
  .flush header { padding: 0 var(--up-space-4); }
  h2 { font: var(--up-type-setting); color: var(--up-ink); }
</style>
