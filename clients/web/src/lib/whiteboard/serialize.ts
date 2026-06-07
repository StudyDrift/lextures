import type { DrawEl } from './types'

export function parseWhiteboardElements(raw: unknown): DrawEl[] {
  if (typeof raw === 'string') {
    const trimmed = raw.trim()
    if (!trimmed) return []
    try {
      return parseWhiteboardElements(JSON.parse(trimmed))
    } catch {
      return []
    }
  }
  if (!Array.isArray(raw)) return []
  return raw.filter(isDrawEl)
}

function isDrawEl(value: unknown): value is DrawEl {
  if (!value || typeof value !== 'object') return false
  const el = value as { type?: string }
  switch (el.type) {
    case 'stroke':
    case 'rect':
    case 'circle':
    case 'triangle':
    case 'line':
      return true
    default:
      return false
  }
}

export function serializeWhiteboardElements(elements: DrawEl[]): string {
  return JSON.stringify(elements)
}
