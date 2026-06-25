import { describe, expect, it } from 'vitest'
import { normalizeLegacyWorkflowGraph } from '../workflow-normalize'
import { WORKFLOW_VERSION } from '../types'

describe('normalizeLegacyWorkflowGraph', () => {
  it('rewrites submission alias', () => {
    const { graph, changes } = normalizeLegacyWorkflowGraph({
      version: WORKFLOW_VERSION,
      nodes: [{ id: 'sub', type: 'submission', position: { x: 0, y: 0 }, data: {} }],
      edges: [],
    })
    expect(changes).toBeGreaterThan(0)
    expect(graph.nodes[0]?.type).toBe('studentSubmission')
  })

  it('expands assignmentContext context handle', () => {
    const { graph } = normalizeLegacyWorkflowGraph({
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        {
          id: 'ctx',
          type: 'assignmentContext',
          position: { x: -640, y: 0 },
          data: { includeContent: true, includeRubric: false },
        },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [
        { id: 'e-grade', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
        { id: 'e-context', source: 'ctx', target: 'ai1', targetHandle: 'context' },
      ],
    })
    expect(graph.nodes.find((node) => node.id === 'ctx')?.type).toBe('activity')
    expect(graph.edges.some((edge) => edge.source === 'ctx' && edge.targetHandle === 'input')).toBe(true)
    expect(graph.edges.some((edge) => edge.targetHandle === 'context')).toBe(false)
  })
})
