// Motion presets. State changes should feel near-instant, so everything is short
// and ease-out, and it all collapses to zero under prefers-reduced-motion.
import { cubicOut } from 'svelte/easing'
import { fade, fly, slide, scale } from 'svelte/transition'
import { flip } from 'svelte/animate'

const reduced = typeof matchMedia !== 'undefined' && matchMedia('(prefers-reduced-motion: reduce)').matches

/** Duration in ms, or 0 when the user prefers reduced motion. */
export const dur = (ms: number) => (reduced ? 0 : ms)

export const pageIn = (node: Element) => fly(node, { y: 6, duration: dur(180), easing: cubicOut })
export const rowIn = (node: Element, { i = 0 }: { i?: number } = {}) => fly(node, { y: 8, duration: dur(200), delay: dur(Math.min(i, 12) * 22), easing: cubicOut })
export const panel = (node: Element) => slide(node, { duration: dur(200), easing: cubicOut })
export const soft = (node: Element) => fade(node, { duration: dur(140) })
export const pop = (node: Element) => scale(node, { start: 0.96, duration: dur(180), easing: cubicOut })
export const reorder = (node: Element, params: { from: DOMRect; to: DOMRect }) => flip(node, params, { duration: dur(220), easing: cubicOut })
