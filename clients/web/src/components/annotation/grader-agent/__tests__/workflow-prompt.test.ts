import { describe, expect, it } from 'vitest'
import { workflowPromptIsPresent, workflowPromptText } from '../workflow-prompt'

describe('workflow prompt validation', () => {
  it('treats blank prompts as missing', () => {
    expect(workflowPromptText({})).toBe('')
    expect(workflowPromptIsPresent({ prompt: '   ' })).toBe(false)
  })

  it('rejects punctuation-only prompts', () => {
    expect(workflowPromptIsPresent({ prompt: '$' })).toBe(false)
    expect(workflowPromptIsPresent({ prompt: '!!!' })).toBe(false)
  })

  it('accepts prompts with letters or digits', () => {
    expect(workflowPromptIsPresent({ prompt: 'Grade fairly' })).toBe(true)
    expect(workflowPromptIsPresent({ prompt: '$50 max' })).toBe(true)
  })
})