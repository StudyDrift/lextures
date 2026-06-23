import type { PaletteNodeType } from './types'

const PALETTE_NODE_TYPES: readonly PaletteNodeType[] = [
  'studentSubmission',
  'activity',
  'ai',
  'codeTestRunner',
  'conditionalRouter',
]

/** In-memory drag payload — React Flow recommends this over dataTransfer for palette DnD. */
let pendingPaletteDragType: PaletteNodeType | null = null

export function parsePaletteNodeType(raw: string | null | undefined): PaletteNodeType | null {
  if (!raw) return null
  return PALETTE_NODE_TYPES.includes(raw as PaletteNodeType) ? (raw as PaletteNodeType) : null
}

export function beginPaletteDrag(type: PaletteNodeType): void {
  pendingPaletteDragType = type
}

export function endPaletteDrag(): void {
  pendingPaletteDragType = null
}

export function peekPaletteDragType(): PaletteNodeType | null {
  return pendingPaletteDragType
}

export function consumePaletteDragType(): PaletteNodeType | null {
  const type = pendingPaletteDragType
  pendingPaletteDragType = null
  return type
}
