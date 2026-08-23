<script lang="ts">
  import { api, type Status, type Settings, type Project, type Delivery } from '../lib/api'
  import { relative, duration, retentionLabel, compact } from '../lib/format'
  import Card from '../lib/ui/Card.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Select from '../lib/ui/Select.svelte'
  import Input from '../lib/ui/Input.svelte'
  import SettingRow from '../lib/ui/SettingRow.svelte'
  import StatusDot from '../lib/ui/StatusDot.svelte'
  import Notice from '../lib/ui/Notice.svelte'
  import CodeBlock from '../lib/ui/CodeBlock.svelte'
  import Metric from '../lib/ui/Metric.svelte'
  import { panel, pop, reorder, soft } from '../lib/motion'
  import Skeleton from '../lib/ui/Skeleton.svelte'

  let status = $state<Status | null>(null)
  let settings = $state<Settings | null>(null)
  let projects = $state<Project[]>([])
  let error = $state('')
  let testing = $state(false)
  let testResult = $state<{ deliveries: Delivery[]; apns_configured: boolean } | null>(null)
  let newKey = $state('')
  let testProject = $state('')

  async function load() {
    try {
      ;[status, settings] = await Promise.all([api.status(), api.settings()])
      projects = (await api.projects()).projects
    } catch (e: any) {
      error = e.message
    }
  }
  $effect(() => {
    load()
  })

  async function setRetention(v: string) {
    try {
      settings = await api.updateSettings({ retention_days: Number(v) })
    } catch (e: any) {
      error = e.message
    }
  }

  async function addKey() {
    const k = newKey.trim()
    if (!k || !settings) return
    try {
      settings = await api.updateSettings({ redact_keys: [...settings.redact_keys, k] })
      newKey = ''
    } catch (e: any) {
      error = e.message
    }
  }

  async function removeKey(k: string) {
    if (!settings) return
    try {
      settings = await api.updateSettings({ redact_keys: settings.redact_keys.filter((x) => x !== k) })
    } catch (e: any) {
      error = e.message
    }
  }

  async function sendTest() {
    testing = true
    error = ''
    testResult = null
    try {
      testResult = await api.test(testProject || undefined)
      await load()
    } catch (e: any) {
      error = e.message
    } finally {
      testing = false
    }
  }

  const retentionOptions = [
    { value: '7', label: '7 days' },
    { value: '14', label: '14 days' },
    { value: '30', label: '30 days' },
    { value: '90', label: '90 days' },
    { value: '0', label: 'Unlimited' },
  ]
  const lastPush = $derived(status?.last_push ?? null)
</script>

<div class="stack">
  {#if error}<div transition:panel><Notice tone="bad">{error}</Notice></div>{/if}

  {#if !status && !error}
    <div class="metrics">
      {#each [0, 1, 2, 3] as i (i)}
        <Card><Skeleton lines={2} height={12} widths={['50%', '35%']} /></Card>
      {/each}
    </div>
    <Card><Skeleton lines={1} height={13} width="20%" /><div class="status" style="margin-top: 16px">{#each [0, 1, 2, 3, 4, 5] as i (i)}<Skeleton lines={2} height={11} widths={['40%', '70%']} />{/each}</div></Card>
    <Card><Skeleton lines={3} height={12} widths={['35%', '80%', '60%']} /></Card>
  {/if}

  {#if status}
    <div class="metrics" in:soft>
      <Card><Metric label="Events" value={compact(status.events)} /></Card>
      <Card><Metric label="Projects" value={String(status.projects)} /></Card>
      <Card><Metric label="Devices" value={String(status.devices)} delta={status.pushable_devices < status.devices ? `${status.pushable_devices} with push` : undefined} tone="neutral" /></Card>
      <Card><Metric label="Uptime" value={duration(status.uptime_seconds)} /></Card>
    </div>

    <Card title="Status">
      <div class="status">
        <div><span class="k">Server</span><StatusDot tone="ok">Healthy</StatusDot></div>
        <div><span class="k">Database</span>{#if status.database === 'ok'}<StatusDot tone="ok">Healthy</StatusDot>{:else}<StatusDot tone="bad">{status.database}</StatusDot>{/if}</div>
        <div>
          <span class="k">APNs</span>
          {#if status.apns.configured}
            <StatusDot tone="ok">Configured · {status.apns.environment}</StatusDot>
          {:else}
            <StatusDot tone="warn">Not configured</StatusDot>
          {/if}
        </div>
        <div>
          <span class="k">Last push</span>
          {#if lastPush}
            <StatusDot tone={lastPush.status === 'sent' ? 'ok' : lastPush.status === 'failed' ? 'bad' : 'muted'}>
              {lastPush.status === 'sent' ? 'Successful' : lastPush.status === 'failed' ? 'Failed' : 'Skipped'} {relative(lastPush.attempted_at)}
            </StatusDot>
          {:else}
            <span class="muted">None yet</span>
          {/if}
        </div>
        <div><span class="k">Version</span><span>{status.version}</span></div>
        <div><span class="k">Database path</span><span class="mono">{status.database_path}</span></div>
        <div><span class="k">Base URL</span><span class="mono">{status.base_url}</span></div>
        <div><span class="k">Retention</span><span>{retentionLabel(status.retention_days)}</span></div>
        <div>
          <span class="k">Admin login</span>
          {#if status.admin_auth}<StatusDot tone="ok">Enabled</StatusDot>{:else}<StatusDot tone="warn">Off · set BOOP_ADMIN_USER and BOOP_ADMIN_PASSWORD</StatusDot>{/if}
        </div>
      </div>
      {#if !status.apns.configured}
        <div style="margin-top: 16px">
          <Notice tone="warn">
            Pushes are stored but not sent. {status.apns.error ? status.apns.error + '.' : ''} Set the APNS_* environment variables and restart Boop.
          </Notice>
        </div>
      {:else if lastPush?.status === 'failed' && lastPush.error}
        <div style="margin-top: 16px"><Notice tone="bad">Last push failed: {lastPush.error}</Notice></div>
      {/if}
    </Card>

    <Card title="Apple Push Notifications">
      <div class="status">
        <div><span class="k">Team id</span><span class="mono">{status.apns.team_id || '—'}</span></div>
        <div><span class="k">Key id</span><span class="mono">{status.apns.key_id || '—'}</span></div>
        <div><span class="k">Bundle id</span><span class="mono">{status.apns.bundle_id || '—'}</span></div>
        <div><span class="k">Environment</span><span>{status.apns.environment}</span></div>
      </div>
      {#if !status.apns.configured}
        <p class="secondary lead" style="margin-top: 16px">Add these to your container environment. The private key should be mounted as a file, not pasted into an environment variable.</p>
        <CodeBlock code={`APNS_TEAM_ID=YOUR_TEAM_ID\nAPNS_KEY_ID=YOUR_KEY_ID\nAPNS_BUNDLE_ID=com.example.Boop\nAPNS_PRIVATE_KEY_PATH=/run/secrets/apns.p8\nAPNS_ENVIRONMENT=production`} />
      {/if}
    </Card>

    <Card title="Test Boop">
      <p class="secondary lead">Creates a test event and pushes it to every paired phone.</p>
      <div class="row" style="margin-top: 12px; flex-wrap: wrap">
        <Select bind:value={testProject} options={[{ value: '', label: projects.length ? `Project: ${projects[projects.length - 1]?.name}` : 'No project yet' }, ...projects.map((p) => ({ value: p.id, label: p.name }))]} style="width: 220px" aria-label="Project" />
        <Button onclick={sendTest} disabled={testing || projects.length === 0}>{testing ? 'Sending' : 'Send test notification'}</Button>
      </div>
      {#if testResult}
        <div style="margin-top: 16px" transition:panel>
          {#if testResult.deliveries.length === 0}
            <Notice tone="info">Event created. No paired phones with push registered, so nothing was sent.</Notice>
          {:else if !testResult.apns_configured}
            <Notice tone="warn">Event created, but APNs is not configured so {testResult.deliveries.length} delivery{testResult.deliveries.length === 1 ? ' was' : 'ies were'} skipped.</Notice>
          {:else if testResult.deliveries.every((d) => d.status === 'sent')}
            <Notice tone="good">Sent to {testResult.deliveries.length} device{testResult.deliveries.length === 1 ? '' : 's'}.</Notice>
          {:else}
            <Notice tone="bad">
              {#each testResult.deliveries.filter((d) => d.status !== 'sent') as d (d.id)}
                <div>{d.device_name}: {d.error}</div>
              {/each}
            </Notice>
          {/if}
        </div>
      {/if}
    </Card>
  {/if}

  {#if settings}
    <Card title="Retention">
      <SettingRow label="Keep events for" hint="Older events are deleted automatically once an hour. Unlimited keeps everything.">
        <Select value={String(settings.retention_days)} options={retentionOptions.some((o) => o.value === String(settings!.retention_days)) ? retentionOptions : [...retentionOptions, { value: String(settings.retention_days), label: `${settings.retention_days} days` }]} onchange={(e) => setRetention((e.currentTarget as HTMLSelectElement).value)} style="width: 150px" />
      </SettingRow>
    </Card>

    <Card title="Redaction">
      <p class="secondary lead">Values under these keys are replaced with [REDACTED] anywhere in event data before it is stored. Matching ignores case and treats - and _ the same.</p>
      <div class="keys" style="margin-top: 12px">
        {#each settings.default_redact_keys as k (k)}
          <span class="pill muted">{k}</span>
        {/each}
        {#each settings.redact_keys as k (k)}
          <span class="pill custom" in:pop out:soft animate:reorder>{k}<button type="button" aria-label="Remove {k}" onclick={() => removeKey(k)}>×</button></span>
        {/each}
      </div>
      <form
        class="row"
        style="margin-top: 16px"
        onsubmit={(e) => {
          e.preventDefault()
          addKey()
        }}
      >
        <Input bind:value={newKey} placeholder="Add a key, e.g. ssn" aria-label="Redaction key" style="width: 220px" mono />
        <Button variant="secondary" type="submit" disabled={!newKey.trim()}>Add</Button>
      </form>
    </Card>
  {/if}
</div>

<style>
  .metrics { display: grid; grid-template-columns: repeat(4, 1fr); gap: var(--up-space-4); }
  .status { display: grid; grid-template-columns: 1fr 1fr; gap: 12px 24px; }
  .status > div { display: flex; flex-direction: column; gap: 3px; font: var(--up-type-meta); min-width: 0; }
  .status span:last-child { overflow: hidden; text-overflow: ellipsis; }
  .k { font: var(--up-type-caption); color: var(--up-text-muted); }
  .lead { font: var(--up-type-meta); line-height: 1.6; }
  .keys { display: flex; flex-wrap: wrap; gap: 8px; }
  .pill { display: inline-flex; align-items: center; gap: 6px; font: var(--up-type-code); padding: 4px 10px; border-radius: var(--up-radius-pill); background: var(--up-bg-hover); box-shadow: var(--up-ring-inset); }
  .pill.custom { background: var(--up-accent-tint); color: var(--up-accent-hover); }
  .pill button { background: none; border: none; cursor: pointer; color: inherit; font-size: 14px; line-height: 1; padding: 0; }
  @media (max-width: 600px) {
    .metrics { grid-template-columns: 1fr 1fr; }
    .status { grid-template-columns: 1fr; }
  }
</style>
