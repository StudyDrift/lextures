import { useRef, type DragEvent, type KeyboardEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { beginPaletteDrag } from './palette-drag'
import type { PaletteNodeType } from './types'

export const GRADER_AGENT_DRAG_MIME = 'text/plain'

type NodePaletteProps = {
  disabled?: boolean
  onAddNode: (type: PaletteNodeType) => void
}

function paletteItemClass(kind: PaletteNodeType): string {
  if (kind === 'studentSubmission') {
    return 'border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800'
  }
  if (kind === 'ai') {
    return 'border-indigo-200 text-indigo-900 hover:bg-indigo-50 dark:border-indigo-900 dark:text-indigo-200 dark:hover:bg-indigo-950/40'
  }
  return 'border-amber-200 text-amber-900 hover:bg-amber-50 dark:border-amber-900 dark:text-amber-200 dark:hover:bg-amber-950/40'
}

function PaletteGroup({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-slate-400 dark:text-neutral-500">
        {title}
      </p>
      <div className="flex flex-col gap-2">{children}</div>
    </div>
  )
}

function PaletteItem({
  type,
  label,
  disabled,
  onAddNode,
}: {
  type: PaletteNodeType
  label: string
  disabled?: boolean
  onAddNode: (type: PaletteNodeType) => void
}) {
  const draggedRef = useRef(false)

  const onDragStart = (event: DragEvent<HTMLDivElement>) => {
    if (disabled) {
      event.preventDefault()
      return
    }
    draggedRef.current = true
    beginPaletteDrag(type)
    event.dataTransfer.setData(GRADER_AGENT_DRAG_MIME, type)
    event.dataTransfer.effectAllowed = 'move'
  }

  const onDragEnd = () => {
    window.setTimeout(() => {
      draggedRef.current = false
    }, 100)
  }

  return (
    <div
      role="button"
      tabIndex={disabled ? -1 : 0}
      draggable={!disabled}
      aria-disabled={disabled}
      onDragStart={onDragStart}
      onDragEnd={onDragEnd}
      onClick={() => {
        if (disabled || draggedRef.current) return
        onAddNode(type)
      }}
      onKeyDown={(event: KeyboardEvent<HTMLDivElement>) => {
        if (disabled) return
        if (event.key === 'Enter' || event.key === ' ') {
          event.preventDefault()
          onAddNode(type)
        }
      }}
      className={`cursor-grab rounded-lg border px-3 py-2 text-start text-sm font-medium active:cursor-grabbing aria-disabled:cursor-not-allowed aria-disabled:opacity-50 ${paletteItemClass(type)}`}
    >
      {label}
    </div>
  )
}

export function NodePalette({ disabled, onAddNode }: NodePaletteProps) {
  const { t } = useTranslation('common')

  return (
    <>
      <p className="mb-3 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.canvas.palette.title')}
      </p>
      {!disabled ? (
        <div className="flex flex-col gap-4">
          <PaletteGroup title={t('gradingAgent.canvas.palette.groupInput')}>
            <PaletteItem
              type="studentSubmission"
              label={t('gradingAgent.canvas.palette.studentSubmission')}
              disabled={disabled}
              onAddNode={onAddNode}
            />
            <PaletteItem
              type="activity"
              label={t('gradingAgent.canvas.palette.activity')}
              disabled={disabled}
              onAddNode={onAddNode}
            />
          </PaletteGroup>
          <PaletteGroup title={t('gradingAgent.canvas.palette.groupProcessing')}>
            <PaletteItem
              type="ai"
              label={t('gradingAgent.canvas.palette.ai')}
              disabled={disabled}
              onAddNode={onAddNode}
            />
          </PaletteGroup>
        </div>
      ) : null}
    </>
  )
}