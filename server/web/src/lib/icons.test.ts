import { describe, it, expect } from 'vitest'
import { parseIcon } from './icons'

describe('parseIcon', () => {
  it('parses shape:color', () => {
    expect(parseIcon('triangle:mint')).toEqual({ shape: 'triangle', color: 'mint' })
  })
  it('rejects unknown values and free text', () => {
    expect(parseIcon('star:mint')).toBeNull()
    expect(parseIcon('circle:red')).toBeNull()
    expect(parseIcon('🚀')).toBeNull()
    expect(parseIcon('')).toBeNull()
  })
})
