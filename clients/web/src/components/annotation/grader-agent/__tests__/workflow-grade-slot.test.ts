import { describe, expect, it } from 'vitest'
import { WORKFLOW_VERSION, type GraderWorkflowGraph } from '../types'
import { workflowHasAttachedRubric } from '../workflow-grade-slot'

describe('workflow grade slot label', () => {
  it('defaults to score when no rubric is wired', () => {
    const graph: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'act', type: 'activity', position: { x: -320, y: 0 }, data: {} },
      ],
      edges: [{ id: 'e1', source: 'act', sourceHandle: 'content', target: 'g1', targetHandle: 'content' }],
    }
    expect(workflowHasAttachedRubric(graph)).toBe(false)
  })

  it('detects an attached activity rubric output', () => {
    const graph: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'act', type: 'activity', position: { x: -320, y: 0 }, data: {} },
        { id: 'g1', type: 'grader', position: { x: -160, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [{ id: 'e1', source: 'act', sourceHandle: 'rubric', target: 'g1', targetHandle: 'rubric' }],
    }
    expect(workflowHasAttachedRubric(graph)).toBe(true)
  })

  it('detects an attached rubric node output', () => {
    const graph: GraderWorkflowGraph = {
      version: WORKFLOW_VERSION,
      nodes: [
        { id: 'output', type: 'output', position: { x: 0, y: 0 }, data: {} },
        { id: 'rub1', type: 'rubric', position: { x: -320, y: 0 }, data: { source: 'assignment' } },
        { id: 'ai1', type: 'ai', position: { x: -160, y: 0 }, data: { prompt: 'Grade' } },
      ],
      edges: [{ id: 'e1', source: 'rub1', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' }],
    }
    expect(workflowHasAttachedRubric(graph)).toBe(true)
  })
})