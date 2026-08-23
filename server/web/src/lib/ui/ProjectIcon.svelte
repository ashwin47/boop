<script lang="ts">
  // Renders a project icon: "<shape>:<color>" as an inline SVG from the palette, anything else as text.
  import { ICON_COLORS, parseIcon } from '../icons'
  let { icon, size = 16 }: { icon: string; size?: number } = $props()
  const parsed = $derived(parseIcon(icon))
  const fill = $derived(parsed ? ICON_COLORS[parsed.color] ?? ICON_COLORS.periwinkle : '')
</script>

{#if parsed}
  <svg width={size} height={size} viewBox="0 0 20 20" aria-hidden="true" class="icon">
    {#if parsed.shape === 'circle'}
      <circle cx="10" cy="10" r="8" {fill} />
    {:else if parsed.shape === 'ring'}
      <circle cx="10" cy="10" r="6.5" fill="none" stroke={fill} stroke-width="3.5" />
    {:else if parsed.shape === 'square'}
      <rect x="2.5" y="2.5" width="15" height="15" rx="3" {fill} />
    {:else if parsed.shape === 'diamond'}
      <rect x="4.5" y="4.5" width="11" height="11" rx="2" transform="rotate(45 10 10)" {fill} />
    {:else if parsed.shape === 'triangle'}
      <path d="M10 2.5 L18 16.5 Q18.6 17.6 17.4 17.6 L2.6 17.6 Q1.4 17.6 2 16.5 Z" {fill} stroke={fill} stroke-width="1.5" stroke-linejoin="round" />
    {:else if parsed.shape === 'hexagon'}
      <path d="M10 1.8 L17.2 6 L17.2 14 L10 18.2 L2.8 14 L2.8 6 Z" {fill} stroke={fill} stroke-width="1.2" stroke-linejoin="round" />
    {:else if parsed.shape === 'pill'}
      <rect x="1.5" y="5.5" width="17" height="9" rx="4.5" {fill} />
    {:else if parsed.shape === 'blob'}
      <path d="M10.5 2 C15 2 18.5 5 18 9.5 C17.6 13.5 15 18 10 18 C5.5 18 2 15 2 10.5 C2 6 6 2 10.5 2 Z" {fill} />
    {/if}
  </svg>
{:else if icon}
  <span class="txt" style="font-size: {size}px; line-height: 1">{icon}</span>
{/if}

<style>
  .icon { display: inline-block; vertical-align: -3px; flex-shrink: 0; }
  .txt { display: inline-block; }
</style>
