import type { Editor } from '@tiptap/core'
import { createPortal } from 'react-dom'
import { useLayoutEffect, useRef, useState, type ReactNode } from 'react'
import {
  clampCenteredToolbarLeft,
  useCaretAnchoredPosition,
} from './use-caret-anchored-position'

export type CaretAnchoredToolbarPortalProps = {
  editor: Editor | null | undefined
  enabled: boolean
  children: ReactNode
}

/**
 * Renders a toolbar in a fixed-position portal aligned to the TipTap caret.
 */
export function CaretAnchoredToolbarPortal({
  editor,
  enabled,
  children,
}: CaretAnchoredToolbarPortalProps) {
  const position = useCaretAnchoredPosition(editor, enabled)
  const containerRef = useRef<HTMLDivElement>(null)
  const [clampedLeft, setClampedLeft] = useState(0)

  useLayoutEffect(() => {
    if (!position?.visible || !containerRef.current) return
    const width = containerRef.current.offsetWidth
    setClampedLeft(clampCenteredToolbarLeft(position.centerX, width))
  }, [position?.centerX, position?.top, position?.visible, children])

  if (!position?.visible) {
    return null
  }

  return createPortal(
    <div
      ref={containerRef}
      className="pointer-events-auto w-max max-w-[calc(100vw-2rem)]"
      style={{
        position: 'fixed',
        left: clampedLeft,
        top: position.top,
        zIndex: 50,
      }}
    >
      {children}
    </div>,
    document.body,
  )
}
