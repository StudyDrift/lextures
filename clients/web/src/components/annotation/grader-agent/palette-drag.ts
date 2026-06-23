import type { PaletteNodeType } from './types'

/** In-memory drag payload — React Flow recommends this over dataTransfer for palette DnD. */
let pendingPaletteDragType: PaletteNodeType | null = null

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
