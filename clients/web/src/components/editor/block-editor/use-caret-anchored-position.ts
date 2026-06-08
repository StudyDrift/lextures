import type { Editor } from '@tiptap/core'
import { useCallback, useEffect, useState } from 'react'

export const CARET_TOOLBAR_HEIGHT = 36
export const CARET_TOOLBAR_GAP = 8
export const CARET_VIEWPORT_INSET = 16

export type CaretAnchoredPosition = {
  /** Horizontal center of the caret in viewport coordinates. */
  centerX: number
  top: number
  visible: boolean
}

function getScrollParents(element: HTMLElement | null): HTMLElement[] {
  const parents: HTMLElement[] = []
  let node = element?.parentElement ?? null
  while (node) {
    const style = getComputedStyle(node)
    const overflow = `${style.overflow} ${style.overflowY} ${style.overflowX}`
    if (/(auto|scroll|overlay)/.test(overflow)) {
      parents.push(node)
    }
    node = node.parentElement
  }
  return parents
}

/**
 * Maps the TipTap caret to fixed viewport coordinates for a toolbar placed above (or below) the line.
 */
export function resolveCaretAnchoredPosition(
  editor: Editor,
  toolbarHeight = CARET_TOOLBAR_HEIGHT,
): CaretAnchoredPosition | null {
  try {
    const { from } = editor.state.selection
    const coords = editor.view.coordsAtPos(from)

    const centerX = (coords.left + coords.right) / 2

    if (coords.bottom < 0 || coords.top > window.innerHeight) {
      return { centerX, top: 0, visible: false }
    }

    const aboveTop = coords.top - toolbarHeight - CARET_TOOLBAR_GAP
    const top =
      aboveTop < CARET_VIEWPORT_INSET ? coords.bottom + CARET_TOOLBAR_GAP : aboveTop

    return {
      centerX,
      top,
      visible: true,
    }
  } catch {
    return null
  }
}

export function clampCenteredToolbarLeft(centerX: number, toolbarWidth: number): number {
  const idealLeft = centerX - toolbarWidth / 2
  const maxLeft = window.innerWidth - CARET_VIEWPORT_INSET - toolbarWidth
  return Math.max(CARET_VIEWPORT_INSET, Math.min(idealLeft, maxLeft))
}

export function useCaretAnchoredPosition(
  editor: Editor | null | undefined,
  enabled: boolean,
): CaretAnchoredPosition | null {
  const [position, setPosition] = useState<CaretAnchoredPosition | null>(null)

  const sync = useCallback(() => {
    if (!enabled || !editor || editor.isDestroyed) {
      setPosition(null)
      return
    }
    setPosition(resolveCaretAnchoredPosition(editor))
  }, [editor, enabled])

  useEffect(() => {
    if (!enabled || !editor || editor.isDestroyed) {
      setPosition(null)
      return
    }

    sync()
    editor.on('selectionUpdate', sync)
    editor.on('update', sync)

    const dom = editor.view.dom as HTMLElement
    const scrollParents = getScrollParents(dom)
    let raf = 0
    const onScroll = () => {
      cancelAnimationFrame(raf)
      raf = requestAnimationFrame(sync)
    }

    scrollParents.forEach((el) => el.addEventListener('scroll', onScroll, { passive: true }))
    window.addEventListener('scroll', onScroll, { passive: true })
    window.addEventListener('resize', onScroll)

    return () => {
      cancelAnimationFrame(raf)
      editor.off('selectionUpdate', sync)
      editor.off('update', sync)
      scrollParents.forEach((el) => el.removeEventListener('scroll', onScroll))
      window.removeEventListener('scroll', onScroll)
      window.removeEventListener('resize', onScroll)
    }
  }, [editor, enabled, sync])

  return position
}
