import type { Level } from './api'

export const LEVEL_LABEL: Record<Level, string> = {
  info: 'Info',
  success: 'Success',
  warning: 'Warning',
  error: 'Error',
  critical: 'Critical',
}

/** CSS custom-property names for a level's pastel fill and strong pair. */
export function levelColors(level: string): { fill: string; strong: string } {
  const l = (['info', 'success', 'warning', 'error', 'critical'] as const).includes(level as Level) ? level : 'info'
  return { fill: `var(--boop-${l})`, strong: `var(--boop-${l}-strong)` }
}
