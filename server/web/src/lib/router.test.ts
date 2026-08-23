import { describe, it, expect } from 'vitest'
import { parseRoute } from './router.svelte'

describe('parseRoute', () => {
  it('maps paths to routes', () => {
    expect(parseRoute('/')).toEqual({ name: 'inbox', params: {} })
    expect(parseRoute('/events/evt_abc')).toEqual({ name: 'event', params: { id: 'evt_abc' } })
    expect(parseRoute('/projects/')).toEqual({ name: 'projects', params: {} })
    expect(parseRoute('/settings')).toEqual({ name: 'settings', params: {} })
    expect(parseRoute('/setup')).toEqual({ name: 'setup', params: {} })
    expect(parseRoute('/nope').name).toBe('notfound')
  })
})
