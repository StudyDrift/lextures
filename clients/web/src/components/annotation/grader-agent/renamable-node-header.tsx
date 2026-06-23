import { useEffect, useRef, useState, type KeyboardEvent, type MouseEvent } from 'react'
import { useTranslation } from 'react-i18next'
import { useWorkflowCanvas } from './workflow-canvas-context'
import { workflowNodeDisplayLabel, workflowNodeLabel } from './workflow-node-label'

type RenamableNodeHeaderProps = {
  nodeId: string
  data: Record<string, unknown>
  defaultLabel: string
  headerClassName: string
  dotClassName?: string
  titleClassName?: string
}

function useRenamableNodeLabel(nodeId: string, data: Record<string, unknown>, defaultLabel: string) {
  const { readOnly, onNodeLabelChange } = useWorkflowCanvas()
  const displayLabel = workflowNodeDisplayLabel(data, defaultLabel)
  const [editing, setEditing] = useState(false)
  const [draft, setDraft] = useState(displayLabel)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (!editing) setDraft(displayLabel)
  }, [displayLabel, editing])

  useEffect(() => {
    if (editing) {
      inputRef.current?.focus()
      inputRef.current?.select()
    }
  }, [editing])

  const startEditing = (event: MouseEvent) => {
    if (readOnly) return
    event.stopPropagation()
    event.preventDefault()
    setDraft(displayLabel)
    setEditing(true)
  }

  const commit = () => {
    const trimmed = draft.trim()
    const next = trimmed && trimmed !== defaultLabel ? trimmed : null
    const current = workflowNodeLabel(data)
    if (next !== current) onNodeLabelChange(nodeId, next)
    setEditing(false)
  }

  const cancel = () => {
    setDraft(displayLabel)
    setEditing(false)
  }

  const onKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    event.stopPropagation()
    if (event.key === 'Enter') {
      event.preventDefault()
      commit()
    } else if (event.key === 'Escape') {
      event.preventDefault()
      cancel()
    }
  }

  return {
    readOnly,
    editing,
    draft,
    setDraft,
    displayLabel,
    startEditing,
    commit,
    cancel,
    onKeyDown,
    inputRef,
  }
}

export function RenamableNodeHeader({
  nodeId,
  data,
  defaultLabel,
  headerClassName,
  dotClassName,
  titleClassName = 'text-sm font-semibold text-slate-800 dark:text-neutral-100',
}: RenamableNodeHeaderProps) {
  const { t } = useTranslation('common')
  const {
    readOnly,
    editing,
    draft,
    setDraft,
    displayLabel,
    startEditing,
    commit,
    onKeyDown,
    inputRef,
  } = useRenamableNodeLabel(nodeId, data, defaultLabel)

  return (
    <div
      className={`flex min-w-0 items-center gap-2 px-3 py-2.5 ${headerClassName}`}
      onDoubleClick={startEditing}
      title={readOnly ? undefined : t('gradingAgent.canvas.nodeLabel.renameHint')}
    >
      {dotClassName ? <span className={`size-2 shrink-0 rounded-full ${dotClassName}`} aria-hidden /> : null}
      {editing ? (
        <input
          ref={inputRef}
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onBlur={() => {
            commit()
          }}
          onKeyDown={onKeyDown}
          onClick={(event) => event.stopPropagation()}
          onPointerDown={(event) => event.stopPropagation()}
          className={`nodrag nopan min-w-0 flex-1 rounded border border-slate-300 bg-white px-1.5 py-0.5 text-sm font-semibold text-slate-800 outline-none ring-indigo-500/30 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 ${titleClassName}`}
          aria-label={t('gradingAgent.canvas.nodeLabel.renameInput')}
        />
      ) : (
        <p className={`min-w-0 truncate ${titleClassName}`}>{displayLabel}</p>
      )}
    </div>
  )
}

export function RenamableNodeTitle({
  nodeId,
  data,
  defaultLabel,
  className,
}: {
  nodeId: string
  data: Record<string, unknown>
  defaultLabel: string
  className: string
}) {
  const { t } = useTranslation('common')
  const {
    readOnly,
    editing,
    draft,
    setDraft,
    displayLabel,
    startEditing,
    commit,
    onKeyDown,
    inputRef,
  } = useRenamableNodeLabel(nodeId, data, defaultLabel)

  if (editing) {
    return (
      <input
        ref={inputRef}
        value={draft}
        onChange={(event) => setDraft(event.target.value)}
        onBlur={commit}
        onKeyDown={onKeyDown}
        onClick={(event) => event.stopPropagation()}
        onPointerDown={(event) => event.stopPropagation()}
        className={`nodrag nopan w-full rounded border border-slate-300 bg-white px-1.5 py-0.5 outline-none ring-indigo-500/30 focus:ring-2 dark:border-neutral-600 dark:bg-neutral-950 ${className}`}
        aria-label={t('gradingAgent.canvas.nodeLabel.renameInput')}
      />
    )
  }

  return (
    <p
      className={`truncate ${className}`}
      onDoubleClick={startEditing}
      title={readOnly ? undefined : t('gradingAgent.canvas.nodeLabel.renameHint')}
    >
      {displayLabel}
    </p>
  )
}