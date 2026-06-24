import { describe, expect, it } from 'vitest'
import { connectionIsValid, isWorkflowRunnable } from '../validation'
import { outputSlotSourceIsValid } from '../workflow-output-slot'
import { WORKFLOW_VERSION, type GraderWorkflowGraph } from '../types'

describe('workflow output slot wiring', () => {
  it('accepts AI output into the grade slot', () => {
    expect(outputSlotSourceIsValid('ai', 'output', 'grade')).toBe(true)
    expect(outputSlotSourceIsValid('grader', 'grade', 'grade')).toBe(true)
    expect(outputSlotSourceIsValid('criterionGrader', 'grade', 'grade')).toBe(true)
    expect(outputSlotSourceIsValid('ai', 'output', 'comments')).toBe(false)
    expect(outputSlotSourceIsValid('scoreAggregator', 'grade', 'grade')).toBe(true)
    expect(outputSlotSourceIsValid('scoreAggregator', 'comments', 'comments')).toBe(true)
  })

  it('allows connecting AI output to Student Grade grade input', () => {
    const g: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'act', type: 'activity', position: { x: -640, y: 0 }, data: {} },
        { id: 'ai1', type: 'ai', position: { x: -320, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [
        { id: 'e1', source: 'act', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' },
      ],
    }
    expect(connectionIsValid(g, 'ai1', 'output', 'output', 'grade')).toBe(true)

    const wired: GraderWorkflowGraph = {
      ...g,
      edges: [
        ...g.edges,
        { id: 'e2', source: 'ai1', sourceHandle: 'output', target: 'output', targetHandle: 'grade' },
      ],
    }
    expect(isWorkflowRunnable(wired)).toBe(true)
  })
})