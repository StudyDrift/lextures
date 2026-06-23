/* eslint-disable react-refresh/only-export-components -- component file exports context menu builders */
import { useEffect, useRef } from 'react'
import { createPortal } from 'react-dom'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import type { GraderWorkflowGraph } from './types'
import { workflowNodeDisplayLabel } from './workflow-node-label'

export type WorkflowCanvasContextMenuState =
  | { kind: 'node'; nodeId: string; top: number; left: number }
  | { kind: 'edge'; edgeId: string; sourceId: string; targetId: string; top: number; left: number }
  | null

export type WorkflowCanvasContextMenuItem = {
  kind: 'rename' | 'deleteNode' | 'selectSource' | 'selectTarget' | 'deleteEdge'
  label: string
  destructive?: boolean
}

function menuItemClass(destructive?: boolean) {
  if (destructive) {
    return 'block w-full px-3 py-2 text-start text-sm text-rose-700 hover:bg-rose-50 dark:text-rose-300 dark:hover:bg-rose-950/30'
  }
  return 'block w-full px-3 py-2 text-start text-sm text-slate-800 hover:bg-slate-100 dark:text-neutral-100 dark:hover:bg-neutral-800'
}

function defaultLabelForNodeType(type: string, t: TFunction): string {
  switch (type) {
    case 'studentSubmission':
      return t('gradingAgent.canvas.nodes.studentSubmission.title')
    case 'activity':
      return t('gradingAgent.canvas.nodes.activity.title')
    case 'ai':
      return t('gradingAgent.canvas.nodes.ai.title')
    case 'codeTestRunner':
      return t('gradingAgent.canvas.nodes.codeTests.title')
    case 'grader':
      return t('gradingAgent.canvas.nodes.grader.title')
    case 'output':
      return t('gradingAgent.canvas.nodes.output.title')
    default:
      return type
  }
}

export function nodeDisplayLabel(graph: GraderWorkflowGraph, nodeId: string, t: TFunction): string {
  const node = graph.nodes.find((entry) => entry.id === nodeId)
  if (!node) return nodeId
  return workflowNodeDisplayLabel(node.data, defaultLabelForNodeType(node.type, t))
}

export function buildNodeContextMenuItems(
  nodeId: string,
  readOnly: boolean,
  t: TFunction,
): WorkflowCanvasContextMenuItem[] {
  if (readOnly) return []
  const items: WorkflowCanvasContextMenuItem[] = [
    { kind: 'rename', label: t('gradingAgent.canvas.contextMenu.rename') },
  ]
  if (nodeId !== 'output') {
    items.push({
      kind: 'deleteNode',
      label: t('gradingAgent.canvas.contextMenu.deleteNode'),
      destructive: true,
    })
  }
  return items
}

export function buildEdgeContextMenuItems(
  graph: GraderWorkflowGraph,
  sourceId: string,
  targetId: string,
  readOnly: boolean,
  t: TFunction,
): WorkflowCanvasContextMenuItem[] {
  const sourceLabel = nodeDisplayLabel(graph, sourceId, t)
  const targetLabel = nodeDisplayLabel(graph, targetId, t)
  const items: WorkflowCanvasContextMenuItem[] = [
    {
      kind: 'selectSource',
      label: t('gradingAgent.canvas.contextMenu.selectSource', { label: sourceLabel }),
    },
    {
      kind: 'selectTarget',
      label: t('gradingAgent.canvas.contextMenu.selectTarget', { label: targetLabel }),
    },
  ]
  if (!readOnly) {
    items.push({
      kind: 'deleteEdge',
      label: t('gradingAgent.canvas.contextMenu.deleteEdge'),
      destructive: true,
    })
  }
  return items
}

type WorkflowCanvasContextMenuProps = {
  menu: Exclude<WorkflowCanvasContextMenuState, null>
  graph: GraderWorkflowGraph
  readOnly: boolean
  onClose: () => void
  onSelect: (item: WorkflowCanvasContextMenuItem) => void
}

export function WorkflowCanvasContextMenu({
  menu,
  graph,
  readOnly,
  onClose,
  onSelect,
}: WorkflowCanvasContextMenuProps) {
  const { t } = useTranslation('common')
  const panelRef = useRef<HTMLDivElement>(null)

  const items =
    menu.kind === 'node'
      ? buildNodeContextMenuItems(menu.nodeId, readOnly, t)
      : buildEdgeContextMenuItems(graph, menu.sourceId, menu.targetId, readOnly, t)

  useEffect(() => {
    function onDocMouseDown(event: MouseEvent) {
      if (!(event.target instanceof Node)) return
      if (panelRef.current?.contains(event.target)) return
      onClose()
    }
    function onKey(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        event.stopPropagation()
        onClose()
      }
    }
    document.addEventListener('mousedown', onDocMouseDown)
    document.addEventListener('keydown', onKey)
    return () => {
      document.removeEventListener('mousedown', onDocMouseDown)
      document.removeEventListener('keydown', onKey)
    }
  }, [onClose])

  useEffect(() => {
    function onScroll() {
      onClose()
    }
    window.addEventListener('scroll', onScroll, true)
    window.addEventListener('resize', onScroll)
    return () => {
      window.removeEventListener('scroll', onScroll, true)
      window.removeEventListener('resize', onScroll)
    }
  }, [onClose])

  if (items.length === 0) return null

  const ariaLabel =
    menu.kind === 'node'
      ? t('gradingAgent.canvas.contextMenu.node')
      : t('gradingAgent.canvas.contextMenu.edge')

  const regularItems = items.filter((item) => !item.destructive)
  const destructiveItems = items.filter((item) => item.destructive)

  return createPortal(
    <div
      ref={panelRef}
      role="menu"
      aria-label={ariaLabel}
      className="fixed z-[560] min-w-[12rem] overflow-hidden rounded-xl border border-slate-200 bg-white py-1 shadow-lg shadow-slate-900/10 dark:border-neutral-600 dark:bg-neutral-900"
      style={{ top: menu.top, left: menu.left }}
    >
      {regularItems.map((item) => (
        <button
          key={item.kind}
          type="button"
          role="menuitem"
          className={menuItemClass()}
          onClick={() => onSelect(item)}
        >
          {item.label}
        </button>
      ))}
      {regularItems.length > 0 && destructiveItems.length > 0 ? (
        <div className="my-1 border-t border-slate-100 dark:border-neutral-800" role="separator" aria-hidden />
      ) : null}
      {destructiveItems.map((item) => (
        <button
          key={item.kind}
          type="button"
          role="menuitem"
          className={menuItemClass(true)}
          onClick={() => onSelect(item)}
        >
          {item.label}
        </button>
      ))}
    </div>,
    document.body,
  )
}