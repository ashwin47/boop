<script lang="ts">
  import QRCode from 'qrcode'
  let { data, size = 220 }: { data: string; size?: number } = $props()
  let svg = $state('')
  $effect(() => {
    QRCode.toString(data, { type: 'svg', margin: 1, width: size, color: { dark: '#1A1B25', light: '#FDFDFE' } }).then((s) => (svg = s))
  })
</script>

<div class="qr" style="width: {size}px; height: {size}px">{@html svg}</div>

<style>
  .qr { border: 1px solid var(--up-border-hairline); border-radius: var(--up-radius-tooltip); padding: 8px; background: var(--up-bg); }
  .qr :global(svg) { width: 100%; height: 100%; display: block; }
</style>
