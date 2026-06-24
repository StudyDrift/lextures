import { describe, expect, it } from 'vitest'
import { isWorkflowRunnable, validateWorkflowGraph } from '../validation'
import { WORKFLOW_VERSION } from '../types'

describe('participation template graph', () => {
  it('is runnable with both router branches wired to grade output', () => {
    const g = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output' as const, position: { x: 0, y: 0 }, data: {} },
        { id: 'sub', type: 'studentSubmission' as const, position: { x: -640, y: 0 }, data: {} },
        {
          id: 'router',
          type: 'conditionalRouter' as const,
          position: { x: -320, y: 0 },
          data: { condition: { field: 'isEmpty', operator: 'isTrue', value: true } },
        },
      ],
      edges: [
        { id: 'e1', source: 'sub', sourceHandle: 'submission', target: 'router', targetHandle: 'input' },
        { id: 'e2', source: 'router', sourceHandle: 'then', target: 'output', targetHandle: 'grade' },
        { id: 'e3', source: 'router', sourceHandle: 'else', target: 'output', targetHandle: 'grade' },
      ],
    }
    const issues = validateWorkflowGraph(g)
    expect(issues).toEqual([])
    expect(isWorkflowRunnable(g)).toBe(true)
  })
})
