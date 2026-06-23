import { describe, expect, it } from 'vitest'
import { aiOutputFormatForNode, buildAiSystemPrompt } from '../ai-output-system-prompt'
import type { GraderWorkflowGraph } from '../types'

function graphWithRubricInput(): GraderWorkflowGraph {
  return {
    version: 1,
    nodes: [
      { id: 'ai1', type: 'ai', position: { x: 0, y: 0 }, data: {} },
      { id: 'act1', type: 'activity', position: { x: -200, y: 0 }, data: {} },
    ],
    edges: [{ id: 'e1', source: 'act1', sourceHandle: 'rubric', target: 'ai1', targetHandle: 'input' }],
  }
}

describe('ai output system prompt', () => {
  it('uses rubric format when rubric input is wired', () => {
    expect(aiOutputFormatForNode(graphWithRubricInput(), 'ai1')).toBe('rubric')
  })

  it('includes criterion ids in rubric system prompt', () => {
    const prompt = buildAiSystemPrompt('rubric', {
      criteria: [
        {
          id: 'a1b2c3d4-e5f6-7890-abcd-ef1234567890',
          title: 'Thesis',
          levels: [
            { label: 'Weak', points: 0 },
            { label: 'Strong', points: 4 },
          ],
        },
      ],
    }, 10)
    expect(prompt).toContain('a1b2c3d4-e5f6-7890-abcd-ef1234567890')
    expect(prompt).toContain('"total": 8')
  })
})