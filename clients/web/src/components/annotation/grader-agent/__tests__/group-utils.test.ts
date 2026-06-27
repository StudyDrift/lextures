import { describe, expect, it } from 'vitest'
import {
  createGroupFromSelection,
  flattenWorkflowGraph,
  graphContainsGroup,
  groupNodeData,
  ungroupNode,
} from '../group-utils'
import { isWorkflowRunnable, validateWorkflowGraph } from '../validation'
import { WORKFLOW_VERSION } from '../types'
import type { GraderWorkflowGraph } from '../types'

/** sub -> router(then->ss12, else->ss0); both set scores -> output.grade. */
function tieredGraph(): GraderWorkflowGraph {
  return {
    version: WORKFLOW_VERSION,
    nodes: [
      { id: 'sub', type: 'studentSubmission', position: { x: -640, y: 0 }, data: {} },
      {
        id: 'rtr',
        type: 'conditionalRouter',
        position: { x: -320, y: 0 },
        data: { condition: { field: 'submissionText', operator: 'contains', value: 'x' } },
      },
      { id: 'ss12', type: 'setScore', position: { x: -120, y: -80 }, data: { score: 12 } },
      { id: 'ss0', type: 'setScore', position: { x: -120, y: 80 }, data: { score: 0 } },
      { id: 'output', type: 'output', position: { x: 160, y: 0 }, data: {} },
    ],
    edges: [
      { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'rtr', targetHandle: 'input' },
      { id: 'e2', source: 'rtr', sourceHandle: 'then', target: 'ss12', targetHandle: 'grade' },
      { id: 'e3', source: 'rtr', sourceHandle: 'else', target: 'ss0', targetHandle: 'grade' },
      { id: 'e4', source: 'ss12', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
      { id: 'e5', source: 'ss0', sourceHandle: 'grade', target: 'output', targetHandle: 'grade' },
    ],
  }
}

describe('createGroupFromSelection', () => {
  it('collapses selected nodes and auto-derives boundary ports', () => {
    const g = tieredGraph()
    const result = createGroupFromSelection(g, ['rtr', 'ss12', 'ss0'], 'Q1 scoring')
    expect(result).not.toBeNull()
    const { graph, groupId } = result!
    const groupNode = graph.nodes.find((n) => n.id === groupId)
    expect(groupNode?.type).toBe('group')
    const data = groupNodeData(groupNode!)
    // One input (sub -> router) and one output (set scores -> output, deduped by source/handle).
    expect(data.inputs).toHaveLength(1)
    expect(data.inputs[0].nodeId).toBe('rtr')
    expect(data.outputs).toHaveLength(2)
    // The router + both set scores moved inside; only sub, group, output remain at root.
    expect(graph.nodes.map((n) => n.type).sort()).toEqual(['group', 'output', 'studentSubmission'])
  })

  it('refuses to group the output node or a single node', () => {
    const g = tieredGraph()
    expect(createGroupFromSelection(g, ['rtr', 'output'])).toBeNull()
    expect(createGroupFromSelection(g, ['rtr'])).toBeNull()
  })
})

describe('flatten + validation round-trip', () => {
  it('a grouped graph flattens back to an equivalent runnable graph', () => {
    const g = tieredGraph()
    const grouped = createGroupFromSelection(g, ['rtr', 'ss12', 'ss0'], 'Q1')!.graph
    expect(graphContainsGroup(grouped)).toBe(true)

    const flat = flattenWorkflowGraph(grouped)
    expect(graphContainsGroup(flat)).toBe(false)
    // Validation flattens internally; the grouped graph stays runnable.
    expect(validateWorkflowGraph(g)).toEqual([])
    expect(validateWorkflowGraph(grouped)).toEqual([])
    expect(isWorkflowRunnable(grouped)).toBe(true)
  })

  it('ungroup restores the original member nodes', () => {
    const g = tieredGraph()
    const created = createGroupFromSelection(g, ['rtr', 'ss12', 'ss0'], 'Q1')!
    const restored = ungroupNode(created.graph, created.groupId)
    expect(graphContainsGroup(restored)).toBe(false)
    // Router + both set scores are back (with prefixed ids) wired to the output.
    expect(restored.nodes.filter((n) => n.type === 'setScore')).toHaveLength(2)
    expect(restored.nodes.some((n) => n.type === 'conditionalRouter')).toBe(true)
    expect(isWorkflowRunnable(restored)).toBe(true)
  })
})
