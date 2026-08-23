<script lang="ts">
  import { api, type Event, type Delivery } from '../lib/api'
  import { link } from '../lib/router.svelte'
  import { fullDate, relative } from '../lib/format'
  import Card from '../lib/ui/Card.svelte'
  import LevelBadge from '../lib/ui/LevelBadge.svelte'
  import JsonTree from '../lib/ui/JsonTree.svelte'
  import CodeBlock from '../lib/ui/CodeBlock.svelte'
  import Notice from '../lib/ui/Notice.svelte'
  import Empty from '../lib/ui/Empty.svelte'
  import StatusDot from '../lib/ui/StatusDot.svelte'

  let { id }: { id: string } = $props()
  let event = $state<Event | null>(null)
  let deliveries = $state<Delivery[]>([])
  let error = $state('')
  let showRaw = $state(false)

  $effect(() => {
    event = null
    error = ''
    api.event(id).then((e) => (event = e)).catch((e) => (error = e.message))
    api.eventDeliveries(id).then((r) => (deliveries = r.deliveries)).catch(() => {})
  })

  // Recognised sections rendered specially; everything else falls back to the JSON tree.
  const KNOWN = ['exception', 'stacktrace', 'tags', 'context', 'breadcrumbs']
  const data = $derived((event?.data ?? {}) as Record<string, any>)
  const exception = $derived(data.exception && typeof data.exception === 'object' ? data.exception : null)
  const frames = $derived(Array.isArray(data.stacktrace) ? (data.stacktrace as any[]) : null)
  const tags = $derived(data.tags && typeof data.tags === 'object' && !Array.isArray(data.tags) ? (data.tags as Record<string, unknown>) : null)
  const context = $derived(data.context && typeof data.context === 'object' && !Array.isArray(data.context) ? (data.context as Record<string, unknown>) : null)
  const breadcrumbs = $derived(Array.isArray(data.breadcrumbs) ? (data.breadcrumbs as any[]) : null)
  const rest = $derived(Object.fromEntries(Object.entries(data).filter(([k]) => !KNOWN.includes(k))))
  const hasRest = $derived(Object.keys(rest).length > 0)

  function str(v: unknown): string {
    return typeof v === 'string' ? v : v === undefined || v === null ? '' : JSON.stringify(v)
  }
</script>

<div class="stack">
  <div class="crumb"><a href="/" onclick={link}>Inbox</a><span class="faint">/</span><span class="muted mono">{id}</span></div>

  {#if error}
    <Notice tone="bad">{error}</Notice>
  {:else if !event}
    <Card><Empty title="Loading" /></Card>
  {:else}
    <Card>
      <div class="head">
        <div class="meta">
          <span class="secondary proj">{#if event.project_icon}<span>{event.project_icon}</span>{/if}<span>{event.project_name}</span></span>
          <span class="faint">·</span>
          <LevelBadge level={event.level} />
          {#if event.source}<span class="faint">·</span><span class="muted">{event.source}</span>{/if}
          {#if event.type}<span class="faint">·</span><span class="muted">{event.type}</span>{/if}
        </div>
        <h1>{event.title}</h1>
        {#if event.body}<p class="body">{event.body}</p>{/if}
      </div>
      <div class="facts">
        <div><span class="k">Occurred</span><span title={event.occurred_at}>{fullDate(event.occurred_at)}</span></div>
        <div><span class="k">Received</span><span title={event.created_at}>{fullDate(event.created_at)} · {relative(event.created_at)}</span></div>
        {#if event.fingerprint}<div><span class="k">Fingerprint</span><span class="mono">{event.fingerprint}</span></div>{/if}
        {#if event.external_id}<div><span class="k">External id</span><span class="mono">{event.external_id}</span></div>{/if}
        <div><span class="k">Event id</span><span class="mono">{event.id}</span></div>
      </div>
    </Card>

    {#if exception}
      <Card title="Exception">
        <div class="exc">
          {#if exception.type}<div class="exc-type">{str(exception.type)}</div>{/if}
          {#if exception.message}<div class="exc-msg">{str(exception.message)}</div>{/if}
          {#each Object.entries(exception).filter(([k]) => k !== 'type' && k !== 'message') as [k, v] (k)}
            <div class="kv"><span class="k">{k}</span><span class="mono">{str(v)}</span></div>
          {/each}
        </div>
      </Card>
    {/if}

    {#if frames}
      <Card title="Stacktrace">
        <div class="frames">
          {#each frames as f, i (i)}
            {#if f && typeof f === 'object'}
              <div class="frame" class:inapp={f.in_app === true}>
                <div class="fn">{str(f.function ?? f.module ?? '—')}</div>
                <div class="loc mono">{str(f.file ?? f.filename ?? '')}{#if f.line !== undefined}<span class="faint">:</span>{str(f.line)}{/if}</div>
              </div>
            {:else}
              <div class="frame"><div class="loc mono">{str(f)}</div></div>
            {/if}
          {/each}
        </div>
      </Card>
    {/if}

    {#if tags}
      <Card title="Tags">
        <div class="pills">
          {#each Object.entries(tags) as [k, v] (k)}
            <span class="pill"><span class="pk">{k}</span>{str(v)}</span>
          {/each}
        </div>
      </Card>
    {/if}

    {#if context}
      <Card title="Context">
        <div class="kvs">
          {#each Object.entries(context) as [k, v] (k)}
            {#if v !== null && typeof v === 'object'}
              <div class="kv block"><span class="k">{k}</span><div class="tree"><JsonTree value={v} open /></div></div>
            {:else}
              <div class="kv"><span class="k">{k}</span><span class="mono v">{str(v)}</span></div>
            {/if}
          {/each}
        </div>
      </Card>
    {/if}

    {#if breadcrumbs}
      <Card title="Breadcrumbs">
        <div class="crumbs">
          {#each breadcrumbs as b, i (i)}
            <div class="bc">
              <span class="bc-t muted">{str(b?.timestamp ?? b?.time ?? '')}</span>
              <span class="bc-c secondary">{str(b?.category ?? b?.type ?? '')}</span>
              <span class="bc-m">{str(b?.message ?? b?.msg ?? (typeof b === 'string' ? b : JSON.stringify(b)))}</span>
            </div>
          {/each}
        </div>
      </Card>
    {/if}

    {#if hasRest}
      <Card title="Data">
        <div class="tree">
          {#each Object.entries(rest) as [k, v] (k)}
            <JsonTree value={v} name={k} open />
          {/each}
        </div>
      </Card>
    {/if}

    <Card title="Raw JSON">
      {#snippet action()}
        <button type="button" class="linkish" onclick={() => (showRaw = !showRaw)}>{showRaw ? 'Hide' : 'Show'}</button>
      {/snippet}
      {#if showRaw}
        <CodeBlock code={JSON.stringify(event, null, 2)} />
      {:else}
        <div class="muted">{Object.keys(data).length} top-level keys</div>
      {/if}
    </Card>

    <Card title="Push delivery">
      {#if deliveries.length === 0}
        <div class="muted">No delivery attempts recorded. Pair a phone to receive pushes.</div>
      {:else}
        <div class="dls">
          {#each deliveries as d (d.id)}
            <div class="dl">
              <span class="dl-n">{d.device_name || d.device_id}</span>
              <StatusDot tone={d.status === 'sent' ? 'ok' : d.status === 'failed' ? 'bad' : 'muted'}>{d.status === 'sent' ? 'Sent' : d.status === 'failed' ? 'Failed' : 'Skipped'}</StatusDot>
              <span class="muted caption">{d.error || d.apns_id || ''}</span>
              <span class="muted caption r">{relative(d.attempted_at)}</span>
            </div>
          {/each}
        </div>
      {/if}
    </Card>
  {/if}
</div>

<style>
  .crumb { display: flex; gap: 8px; align-items: center; font: var(--up-type-meta); }
  .head { display: flex; flex-direction: column; gap: 8px; }
  .proj { display: inline-flex; gap: 6px; align-items: center; }
  .meta { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; font: var(--up-type-meta); }
  h1 { font: var(--up-type-metric); letter-spacing: -0.01em; word-break: break-word; }
  .body { font: var(--up-type-status-line); color: var(--up-text-secondary); white-space: pre-wrap; word-break: break-word; }
  .facts { display: grid; grid-template-columns: 1fr 1fr; gap: 8px 24px; margin-top: var(--up-space-5); padding-top: var(--up-space-4); border-top: 1px solid var(--up-border-hairline); }
  .facts > div { display: flex; flex-direction: column; gap: 2px; font: var(--up-type-meta); min-width: 0; }
  .facts span:last-child { overflow: hidden; text-overflow: ellipsis; }
  .k { font: var(--up-type-caption); color: var(--up-text-muted); }
  .exc { display: flex; flex-direction: column; gap: 6px; }
  .exc-type { font: var(--up-type-row-title); }
  .exc-msg { font: var(--up-type-code); color: var(--up-text-secondary); white-space: pre-wrap; word-break: break-word; }
  .frames { display: flex; flex-direction: column; gap: 3px; }
  .frame { padding: 8px 12px; border-radius: var(--up-radius-row); background: var(--up-bg-hover); display: flex; flex-direction: column; gap: 2px; }
  .frame.inapp { background: var(--up-accent-tint); box-shadow: var(--up-ring-inset); }
  .fn { font: var(--up-type-ui); color: var(--up-ink); word-break: break-all; }
  .loc { color: var(--up-text-secondary); word-break: break-all; }
  .pills { display: flex; flex-wrap: wrap; gap: 8px; }
  .pill { display: inline-flex; gap: 6px; font: var(--up-type-small); padding: 4px 10px; border-radius: var(--up-radius-pill); background: var(--up-bg-hover); box-shadow: var(--up-ring-inset); }
  .pk { color: var(--up-text-muted); font-weight: 500; }
  .kvs { display: flex; flex-direction: column; gap: 10px; }
  .kv { display: grid; grid-template-columns: 140px 1fr; gap: 12px; align-items: baseline; font: var(--up-type-meta); }
  .kv.block { align-items: start; }
  .kv .v { word-break: break-all; }
  .tree { overflow-x: auto; }
  .crumbs { display: flex; flex-direction: column; gap: 6px; }
  .bc { display: grid; grid-template-columns: 110px 110px 1fr; gap: 12px; font: var(--up-type-meta); }
  .bc-m { word-break: break-word; }
  .linkish { background: none; border: none; cursor: pointer; font: var(--up-type-ui); color: var(--up-accent); padding: 0; }
  .linkish:hover { color: var(--up-accent-hover); }
  .dls { display: flex; flex-direction: column; gap: 8px; }
  .dl { display: grid; grid-template-columns: 1fr auto 1fr auto; gap: 12px; align-items: center; font: var(--up-type-meta); }
  .r { text-align: right; }
  @media (max-width: 600px) {
    .facts, .kv, .bc { grid-template-columns: 1fr; }
  }
</style>
