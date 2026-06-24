import { useRef, type DragEvent, type KeyboardEvent, type ReactNode } from 'react'
import { useTranslation } from 'react-i18next'
import { beginPaletteDrag } from './palette-drag'
import type { PaletteNodeType } from './types'

export const GRADER_AGENT_DRAG_MIME = 'text/plain'

type NodePaletteProps = {
  disabled?: boolean
  codeExecutionEnabled?: boolean
  onAddNode: (type: PaletteNodeType) => void
}

function paletteItemClass(kind: PaletteNodeType): string {
  if (kind === 'studentSubmission' || kind === 'reference') {
    return 'border-slate-300 text-slate-700 hover:bg-slate-100 dark:border-neutral-600 dark:text-neutral-200 dark:hover:bg-neutral-800'
  }
  if (kind === 'ai' || kind === 'criterionGrader') {
    return 'border-indigo-200 text-indigo-900 hover:bg-indigo-50 dark:border-indigo-900 dark:text-indigo-200 dark:hover:bg-indigo-950/40'
  }
  if (kind === 'codeTestRunner') {
    return 'border-cyan-200 text-cyan-900 hover:bg-cyan-50 dark:border-cyan-900 dark:text-cyan-200 dark:hover:bg-cyan-950/40'
  }
  if (kind === 'conditionalRouter') {
    return 'border-slate-300 text-slate-800 hover:bg-slate-100 dark:border-neutral-600 dark:text-neutral-100 dark:hover:bg-neutral-800'
  }
  if (kind === 'flagForReview') {
    return 'border-rose-200 text-rose-900 hover:bg-rose-50 dark:border-rose-900 dark:text-rose-200 dark:hover:bg-rose-950/40'
  }
  if (kind === 'humanReviewGate') {
    return 'border-slate-300 text-slate-800 hover:bg-slate-100 dark:border-neutral-600 dark:text-neutral-100 dark:hover:bg-neutral-800'
  }
  if (kind === 'originality') {
    return 'border-amber-200 text-amber-900 hover:bg-amber-50 dark:border-amber-900 dark:text-amber-200 dark:hover:bg-amber-950/40'
  }
  if (kind === 'scoreAggregator') {
    return 'border-emerald-200 text-emerald-900 hover:bg-emerald-50 dark:border-emerald-900 dark:text-emerald-200 dark:hover:bg-emerald-950/40'
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

export function NodePalette({ disabled, codeExecutionEnabled = false, onAddNode }: NodePaletteProps) {
  const { t } = useTranslation('common')

  return (
    <>
      <p className="mb-3 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-400">
        {t('gradingAgent.canvas.palette.title')}
      </p>
      <div className={`flex flex-col gap-4${disabled ? ' pointer-events-none opacity-50' : ''}`}>
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
          <PaletteItem
            type="reference"
            label={t('gradingAgent.canvas.palette.reference')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
          <PaletteItem
            type="rubric"
            label={t('gradingAgent.canvas.palette.rubric')}
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
          <PaletteItem
            type="criterionGrader"
            label={t('gradingAgent.canvas.palette.criterionGrader')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
          {codeExecutionEnabled ? (
            <PaletteItem
              type="codeTestRunner"
              label={t('gradingAgent.canvas.palette.codeTests')}
              disabled={disabled}
              onAddNode={onAddNode}
            />
          ) : (
            <div
              className="rounded-lg border border-dashed border-slate-200 px-3 py-2 text-sm text-slate-400 dark:border-neutral-700 dark:text-neutral-500"
              title={t('gradingAgent.canvas.palette.codeTestsDisabledTooltip')}
            >
              {t('gradingAgent.canvas.palette.codeTests')}
            </div>
          )}
          <PaletteItem
            type="conditionalRouter"
            label={t('gradingAgent.canvas.palette.router')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
          <PaletteItem
            type="scoreAggregator"
            label={t('gradingAgent.canvas.palette.aggregator')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
          <PaletteItem
            type="humanReviewGate"
            label={t('gradingAgent.canvas.palette.reviewGate')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
          <PaletteItem
            type="originality"
            label={t('gradingAgent.canvas.palette.originality')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
        </PaletteGroup>
        <PaletteGroup title={t('gradingAgent.canvas.palette.groupOutput')}>
          <div
            className="rounded-lg border border-dashed border-emerald-200 px-3 py-2 text-sm text-emerald-800 dark:border-emerald-900 dark:text-emerald-200"
            title={t('gradingAgent.canvas.palette.studentGradeFixedTooltip')}
          >
            {t('gradingAgent.canvas.palette.studentGrade')}
          </div>
          <PaletteItem
            type="flagForReview"
            label={t('gradingAgent.canvas.palette.flagForReview')}
            disabled={disabled}
            onAddNode={onAddNode}
          />
        </PaletteGroup>
      </div>
    </>
  )
}