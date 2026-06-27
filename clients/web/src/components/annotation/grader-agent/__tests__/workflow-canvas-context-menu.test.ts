import { describe, expect, it, vi } from 'vitest'
import type { GraderWorkflowGraph } from '../types'
import {
  buildEdgeContextMenuItems,
  buildNodeContextMenuItems,
  nodeDisplayLabel,
} from '../workflow-canvas-context-menu'

const t = vi.fn((key: string, options?: { label?: string }) => {
  if (key === 'gradingAgent.canvas.contextMenu.selectSource' && options?.label) {
    return `Select ${options.label}`
  }
  if (key === 'gradingAgent.canvas.contextMenu.selectTarget' && options?.label) {
    return `Select ${options.label}`
  }
  return key
}) as unknown as import('i18next').TFunction

const graph: GraderWorkflowGraph = {
  version: 1,
  nodes: [
    { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
    { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { label: 'Reviewer' } },
    { id: 'sub1', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
  ],
  edges: [
    { id: 'e1', source: 'sub1', sourceHandle: 'submission', target: 'ai1', targetHandle: 'input' },
  ],
}

describe('workflow canvas context menu', () => {
  it('builds editable node menu items and omits delete for output', () => {
    expect(buildNodeContextMenuItems('ai1', 'ai', false, t).map((item) => item.kind)).toEqual([
      'rename',
      'deleteNode',
    ])
    expect(buildNodeContextMenuItems('output', 'output', false, t).map((item) => item.kind)).toEqual([
      'rename',
    ])
    expect(buildNodeContextMenuItems('ai1', 'ai', true, t)).toEqual([])
  })

  it('adds open/ungroup actions for group nodes', () => {
    expect(buildNodeContextMenuItems('grp1', 'group', false, t).map((item) => item.kind)).toEqual([
      'openGroup',
      'rename',
      'ungroup',
      'deleteNode',
    ])
  })

  it('builds edge menu items with source and target labels', () => {
    const items = buildEdgeContextMenuItems(graph, 'sub1', 'ai1', false, t)
    expect(items.map((item) => item.kind)).toEqual([
      'selectSource',
      'selectTarget',
      'deleteEdge',
    ])
    expect(items[0]?.label).toBe('Select gradingAgent.canvas.nodes.studentSubmission.title')
    expect(items[1]?.label).toBe('Select Reviewer')
  })

  it('omits delete edge in read-only mode', () => {
    expect(
      buildEdgeContextMenuItems(graph, 'sub1', 'ai1', true, t).map((item) => item.kind),
    ).toEqual(['selectSource', 'selectTarget'])
  })

  it('resolves custom node labels for edge menu text', () => {
    expect(nodeDisplayLabel(graph, 'ai1', t)).toBe('Reviewer')
    expect(nodeDisplayLabel(graph, 'sub1', t)).toBe('gradingAgent.canvas.nodes.studentSubmission.title')
  })
})