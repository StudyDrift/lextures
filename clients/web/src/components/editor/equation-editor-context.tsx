/* eslint-disable react-refresh/only-export-components -- provider + hook live together */
import type { Editor } from '@tiptap/core'
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'
import { postCourseContext } from '../../lib/courses-api'
import { EquationEditorDialog, type EquationEditorDialogProps } from './EquationEditorDialog'

export type EquationEditTarget = {
  editor: Editor
  pos: number
  nodeType: 'math_inline' | 'math_block'
  latex: string
}

export type EquationEditorContextValue = {
  openInsert: (editor: Editor) => void
  openEdit: (target: EquationEditTarget) => void
}

const EquationEditorCtx = createContext<EquationEditorContextValue | null>(null)

export function useEquationEditor(): EquationEditorContextValue | null {
  return useContext(EquationEditorCtx)
}

export type EquationEditorProviderProps = {
  children: ReactNode
  disabled?: boolean
  courseCode?: string
  structureItemId?: string
  onAuditOpen?: () => void
  onAuditInsert?: () => void
}

export function EquationEditorProvider({
  children,
  disabled,
  courseCode,
  structureItemId,
  onAuditOpen,
  onAuditInsert,
}: EquationEditorProviderProps) {
  const [open, setOpen] = useState(false)
  const [editor, setEditor] = useState<Editor | null>(null)
  const [latex, setLatex] = useState('\\frac{a}{b}')
  const [display, setDisplay] = useState(false)
  const [editTarget, setEditTarget] = useState<EquationEditTarget | null>(null)

  const close = useCallback(() => {
    setOpen(false)
    setEditTarget(null)
    setEditor(null)
  }, [])

  const recordEditorOpen = useCallback(() => {
    onAuditOpen?.()
    if (!courseCode) return
    void postCourseContext(courseCode, {
      kind: 'equation_editor_open',
      ...(structureItemId ? { structureItemId } : {}),
    }).catch(() => {
      /* best-effort */
    })
  }, [onAuditOpen, courseCode, structureItemId])

  const openInsert = useCallback(
    (ed: Editor) => {
      if (disabled) return
      setEditTarget(null)
      setEditor(ed)
      setLatex('\\frac{a}{b}')
      setDisplay(false)
      setOpen(true)
      recordEditorOpen()
    },
    [disabled, recordEditorOpen],
  )

  const openEdit = useCallback(
    (target: EquationEditTarget) => {
      if (disabled) return
      setEditTarget(target)
      setEditor(target.editor)
      setLatex(target.latex)
      setDisplay(target.nodeType === 'math_block')
      setOpen(true)
      recordEditorOpen()
    },
    [disabled, recordEditorOpen],
  )

  const ctx = useMemo(
    (): EquationEditorContextValue => ({ openInsert, openEdit }),
    [openInsert, openEdit],
  )

  const dialogProps: EquationEditorDialogProps = {
    open,
    onClose: close,
    editor,
    latex,
    onLatexChange: setLatex,
    display,
    onDisplayChange: setDisplay,
    editTarget,
    courseCode,
    structureItemId,
    onInserted: onAuditInsert,
  }

  return (
    <EquationEditorCtx.Provider value={ctx}>
      {children}
      <EquationEditorDialog {...dialogProps} />
    </EquationEditorCtx.Provider>
  )
}
