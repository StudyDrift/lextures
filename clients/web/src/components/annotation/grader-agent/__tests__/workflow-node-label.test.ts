import { describe, expect, it } from 'vitest'
import {
  patchWorkflowNodeLabel,
  workflowNodeDisplayLabel,
  workflowNodeLabel,
} from '../workflow-node-label'

describe('workflow node label', () => {
  it('falls back to the default label', () => {
    expect(workflowNodeDisplayLabel({}, 'AI')).toBe('AI')
    expect(workflowNodeLabel({ label: '   ' })).toBeNull()
  })

  it('reads and writes custom labels', () => {
    const data = { prompt: 'x', label: ' Summarizer ' }
    expect(workflowNodeLabel(data)).toBe('Summarizer')
    expect(workflowNodeDisplayLabel(data, 'AI')).toBe('Summarizer')
    expect(patchWorkflowNodeLabel(data, 'Grader pass')).toEqual({ prompt: 'x', label: 'Grader pass' })
    expect(patchWorkflowNodeLabel(data, null)).toEqual({ prompt: 'x' })
  })
})