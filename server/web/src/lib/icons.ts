// Project icon catalogue, mirrored from server/internal/projects/icons.go.
export const ICON_SHAPES = ['circle', 'ring', 'square', 'diamond', 'triangle', 'hexagon', 'pill', 'blob'] as const
export type IconShape = (typeof ICON_SHAPES)[number]

export const ICON_COLORS: Record<string, string> = {
  periwinkle: '#7C83E8',
  mint: '#5FBF9F',
  blush: '#E88CB0',
  amber: '#E8B34C',
  violet: '#9B7BEA',
  slate: '#9C9EAB',
}

export function parseIcon(icon: string): { shape: IconShape; color: string } | null {
  const i = icon.indexOf(':')
  if (i < 0) return null
  const shape = icon.slice(0, i) as IconShape
  const color = icon.slice(i + 1)
  if (!ICON_SHAPES.includes(shape) || !(color in ICON_COLORS)) return null
  return { shape, color }
}
