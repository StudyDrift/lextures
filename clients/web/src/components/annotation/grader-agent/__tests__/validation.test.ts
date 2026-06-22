import { describe, expect, it } from 'vitest'
import { synthesizeDefaultGraph, effectiveWorkflowGraph } from '../default-graph'
import { connectionIsValid, isWorkflowRunnable, validateWorkflowGraph } from '../validation'

describe('grader agent workflow validation', () => {
  it('accepts the default graph with a prompt', () => {
    const g = synthesizeDefaultGraph('Grade fairly', true, true)
    expect(isWorkflowRunnable(g)).toBe(true)
  })

  it('rejects cross-type grade to comments connection', () => {
    const g = synthesizeDefaultGraph('Grade fairly', true, true)
    g.edges[1] = { id: 'e2', source: 'g1', sourceHandle: 'grade', target: 'output', targetHandle: 'comments' }
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field.includes('output'))).toBe(true)
  })

  it('rejects unconnected grade slot', () => {
    const g = synthesizeDefaultGraph('Grade fairly', true, true)
    g.edges = g.edges.filter((e) => e.targetHandle !== 'grade')
    expect(isWorkflowRunnable(g)).toBe(false)
    expect(validateWorkflowGraph(g).some((i) => i.field === 'output.grade')).toBe(true)
  })

  it('synthesizes legacy config into default graph', () => {
    const g = effectiveWorkflowGraph(null, 'Legacy prompt', false, true)
    const grader = g.nodes.find((n) => n.type === 'grader')
    expect(grader?.data.prompt).toBe('Legacy prompt')
  })

  it('validates connection types', () => {
    const g = synthesizeDefaultGraph('x', true, true)
    g.edges = []
    expect(connectionIsValid(g, 'g1', 'grade', 'output', 'grade')).toBe(true)
    expect(connectionIsValid(g, 'g1', 'grade', 'output', 'comments')).toBe(false)
  })
})
