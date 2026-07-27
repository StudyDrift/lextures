import type { CSSProperties } from 'react'

export type Tag = { id: string; label: string; color: string; description?: string }

export type Anchor = {
  prefix: string
  suffix: string
  approxOffset: number
  unitIndex?: number
}

export type Annotation = {
  id: string
  tagId: string
  quote: string
  anchor: Anchor
  note?: string
  createdAt: string
  orphaned?: boolean
}

export type FilterNoteResult = {
  ok?: boolean
  error?: string
  message?: string
  preserveInput?: boolean
  crisis?: boolean
  flagged?: boolean
}

export function newAnnotationId(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) {
    return crypto.randomUUID()
  }
  return `ann_${Math.random().toString(36).slice(2, 11)}`
}

export function asTags(config: Record<string, unknown>): Tag[] {
  if (!Array.isArray(config.tags)) return []
  return config.tags as Tag[]
}

export function asAnnotations(state: Record<string, unknown>): Annotation[] {
  if (!Array.isArray(state.annotations)) return []
  return state.annotations as Annotation[]
}

export function underlineStyle(color: string, patternIndex: number): CSSProperties {
  const styles = ['solid', 'double', 'dashed', 'dotted', 'wavy'] as const
  return {
    backgroundColor: `${color}33`,
    borderBottom: `2px ${styles[patternIndex % styles.length]} ${color}`,
    boxDecorationBreak: 'clone',
    WebkitBoxDecorationBreak: 'clone',
  }
}
