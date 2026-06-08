import type { Editor } from '@tiptap/core'
import { describe, expect, it, vi } from 'vitest'
import {
  CARET_TOOLBAR_GAP,
  CARET_TOOLBAR_HEIGHT,
  CARET_VIEWPORT_INSET,
  clampCenteredToolbarLeft,
  resolveCaretAnchoredPosition,
} from '../use-caret-anchored-position'

function mockEditor(coords: {
  top: number
  bottom: number
  left: number
  right?: number
}): Editor {
  return {
    state: { selection: { from: 1 } },
    view: {
      coordsAtPos: () => ({
        ...coords,
        right: coords.right ?? coords.left + 2,
      }),
    },
  } as unknown as Editor
}

describe('resolveCaretAnchoredPosition', () => {
  it('places the toolbar above the caret, centered on the caret', () => {
    const editor = mockEditor({ top: 200, bottom: 220, left: 48, right: 52 })
    const pos = resolveCaretAnchoredPosition(editor)
    expect(pos).toEqual({
      centerX: 50,
      top: 200 - CARET_TOOLBAR_HEIGHT - CARET_TOOLBAR_GAP,
      visible: true,
    })
  })

  it('flips below the caret when there is not enough room above', () => {
    const editor = mockEditor({ top: 20, bottom: 36, left: 80, right: 84 })
    const pos = resolveCaretAnchoredPosition(editor)
    expect(pos).toEqual({
      centerX: 82,
      top: 36 + CARET_TOOLBAR_GAP,
      visible: true,
    })
  })

  it('hides when the caret is scrolled out of the viewport', () => {
    vi.stubGlobal('innerHeight', 800)
    const editor = mockEditor({ top: 900, bottom: 920, left: 40, right: 44 })
    const pos = resolveCaretAnchoredPosition(editor)
    expect(pos).toEqual({
      centerX: 42,
      top: 0,
      visible: false,
    })
    vi.unstubAllGlobals()
  })

  it('returns null when coordinate lookup fails', () => {
    const editor = {
      state: { selection: { from: 0 } },
      view: {
        coordsAtPos: () => {
          throw new Error('invalid position')
        },
      },
    } as unknown as Editor
    expect(resolveCaretAnchoredPosition(editor)).toBeNull()
  })
})

describe('clampCenteredToolbarLeft', () => {
  it('centers the toolbar on the caret when there is room', () => {
    vi.stubGlobal('innerWidth', 800)
    expect(clampCenteredToolbarLeft(400, 120)).toBe(340)
    vi.unstubAllGlobals()
  })

  it('clamps to the viewport inset when centered position would overflow', () => {
    vi.stubGlobal('innerWidth', 400)
    expect(clampCenteredToolbarLeft(360, 120)).toBe(400 - CARET_VIEWPORT_INSET - 120)
    expect(clampCenteredToolbarLeft(20, 120)).toBe(CARET_VIEWPORT_INSET)
    vi.unstubAllGlobals()
  })
})
