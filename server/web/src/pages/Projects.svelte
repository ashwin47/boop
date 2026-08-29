<script lang="ts">
  import { api, LEVELS, type Project, type ProjectCreated } from '../lib/api'
  import { LEVEL_LABEL } from '../lib/levels'
  import { relative } from '../lib/format'
  import Card from '../lib/ui/Card.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Input from '../lib/ui/Input.svelte'
  import Select from '../lib/ui/Select.svelte'
  import Switch from '../lib/ui/Switch.svelte'
  import SettingRow from '../lib/ui/SettingRow.svelte'
  import CodeBlock from '../lib/ui/CodeBlock.svelte'
  import Notice from '../lib/ui/Notice.svelte'
  import Empty from '../lib/ui/Empty.svelte'
  import ProjectIcon from '../lib/ui/ProjectIcon.svelte'
  import IconPicker from '../lib/ui/IconPicker.svelte'
  import { panel, soft, pop, reorder } from '../lib/motion'
  import Skeleton from '../lib/ui/Skeleton.svelte'

  let projects = $state<Project[]>([])
  let loaded = $state(false)
  let error = $state('')
  let newName = $state('')
  let creating = $state(false)
  let revealed = $state<ProjectCreated | null>(null)
  let editing = $state<string | null>(null)
  let confirmDelete = $state<string | null>(null)

  async function load() {
    try {
      projects = (await api.projects()).projects
    } catch (e: any) {
      error = e.message
    } finally {
      loaded = true
    }
  }
  $effect(() => {
    load()
  })

  async function create() {
    if (!newName.trim()) return
    creating = true
    error = ''
    try {
      revealed = await api.createProject({ name: newName.trim() })
      newName = ''
      await load()
    } catch (e: any) {
      error = e.message
    } finally {
      creating = false
    }
  }

  async function patch(p: Project, patch: Partial<Project>) {
    try {
      const updated = await api.updateProject(p.id, patch)
      projects = projects.map((x) => (x.id === p.id ? updated : x))
    } catch (e: any) {
      error = e.message
    }
  }

  async function rotate(p: Project) {
    try {
      revealed = await api.rotateKey(p.id)
    } catch (e: any) {
      error = e.message
    }
  }

  async function remove(p: Project) {
    try {
      await api.deleteProject(p.id)
      confirmDelete = null
      await load()
    } catch (e: any) {
      error = e.message
    }
  }

  const levelOptions = LEVELS.map((l) => ({ value: l, label: LEVEL_LABEL[l] }))
  const origin = typeof location !== 'undefined' ? location.origin : 'https://boop.example.com'
</script>

<div class="stack">
  {#if error}<div transition:panel><Notice tone="bad">{error}</Notice></div>{/if}

  {#if revealed}
    <div transition:panel>
    <Card title="API key for {revealed.name}">
      <p class="secondary lead">Copy this key now. It is shown once and only a hash is stored.</p>
      <CodeBlock code={revealed.api_key} />
      <p class="muted lead" style="margin-top: 16px">Send your first boop:</p>
      <CodeBlock
        code={`curl ${origin}/api/v1/events \\\n  -H "Authorization: Bearer ${revealed.api_key}" \\\n  -H "Content-Type: application/json" \\\n  -d '{"title": "Hello from ${revealed.name}", "level": "success"}'`}
      />
      <div class="actions"><Button variant="secondary" onclick={() => (revealed = null)}>Done</Button></div>
    </Card>
    </div>
  {/if}

  <Card title="New project">
    <form
      class="new"
      onsubmit={(e) => {
        e.preventDefault()
        create()
      }}
    >
      <Input bind:value={newName} placeholder="Project name" aria-label="Project name" maxlength={80} required />
      <Button type="submit" disabled={creating || !newName.trim()}>Create</Button>
    </form>
    <p class="muted caption" style="margin-top: 8px">Each project gets its own API key and a shape icon you can change in its settings.</p>
  </Card>

  {#if !loaded}
    {#each [0, 1] as i (i)}
      <Card>
        <Skeleton lines={2} height={13} widths={['35%', '55%']} />
      </Card>
    {/each}
  {:else if projects.length === 0}
    <Card><Empty title="No projects yet">Create one above to get an API key.</Empty></Card>
  {/if}

  {#each projects as p (p.id)}
    <div animate:reorder in:soft>
    <Card compact>
      <div class="phead">
        <ProjectIcon icon={p.icon} size={20} />
        <div class="ptext">
          <div class="pname"><span class="n">{p.name}</span><span class="mono muted caption">{p.slug}</span></div>
          <div class="pmeta muted caption">{p.notify ? `notify ≥ ${LEVEL_LABEL[p.min_level].toLowerCase()}` : 'notifications off'} · created {relative(p.created_at)}</div>
        </div>
        <Button variant="ghost" size="sm" onclick={() => (editing = editing === p.id ? null : p.id)}>{editing === p.id ? 'Close' : 'Settings'}</Button>
      </div>

      {#if editing === p.id}
        <div class="edit" transition:panel>
          <SettingRow label="Name">
            <Input value={p.name} onchange={(e) => patch(p, { name: (e.currentTarget as HTMLInputElement).value })} style="width: 200px" />
          </SettingRow>
          <SettingRow label="Icon" hint="An abstract shape from the palette, shown next to the project name in the inbox and on your phone.">
            <IconPicker value={p.icon} onchange={(v) => patch(p, { icon: v })} />
          </SettingRow>
          <SettingRow label="Push notifications" hint="Turn off to store events without notifying your phone.">
            <Switch checked={p.notify} onchange={(v) => patch(p, { notify: v })} label="Push notifications" />
          </SettingRow>
          <SettingRow label="Minimum level" hint="Only events at or above this level trigger a push.">
            <Select value={p.min_level} options={levelOptions} onchange={(e) => patch(p, { min_level: (e.currentTarget as HTMLSelectElement).value as Project['min_level'] })} style="width: 150px" />
          </SettingRow>
          <SettingRow label="API key" hint="Rotating immediately invalidates the current key.">
            <Button variant="secondary" size="sm" onclick={() => rotate(p)}>Rotate key</Button>
          </SettingRow>
          <SettingRow label="Delete project" hint="Removes the project and every event it received.">
            {#if confirmDelete === p.id}
              <div class="row" in:pop>
                <Button variant="secondary" size="sm" onclick={() => (confirmDelete = null)}>Cancel</Button>
                <Button variant="danger" size="sm" onclick={() => remove(p)}>Confirm delete</Button>
              </div>
            {:else}
              <Button variant="danger" size="sm" onclick={() => (confirmDelete = p.id)}>Delete</Button>
            {/if}
          </SettingRow>
        </div>
      {/if}
    </Card>
    </div>
  {/each}
</div>

<style>
  .lead { font: var(--up-type-meta); line-height: 1.6; margin-bottom: 12px; }
  .actions { display: flex; justify-content: flex-end; margin-top: var(--up-space-4); }
  .new { display: flex; gap: var(--up-space-3); align-items: center; }
  .new :global(input:first-child) { flex: 1; }
  .phead { display: flex; align-items: center; gap: 12px; }
  .ptext { flex: 1; min-width: 0; }
  .pname { display: flex; align-items: baseline; gap: 8px; min-width: 0; }
  .n { font: var(--up-type-row-title); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .pmeta { margin-top: 1px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
  .edit { margin-top: var(--up-space-4); }
  @media (max-width: 520px) {
    .new { flex-wrap: wrap; }
  }
</style>
