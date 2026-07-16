import { describe, expect, it } from 'vitest'
import {
  AI_SUBPROCESSOR_BYOK_NOTE,
  SUB_PROCESSORS,
  SUB_PROCESSORS_EFFECTIVE_DATE,
} from '../sub-processors'

describe('trust sub-processors (AP.7)', () => {
  it('has an effective date', () => {
    expect(SUB_PROCESSORS_EFFECTIVE_DATE).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('marks AI model vendors as when-configured, not always-on OpenRouter routing', () => {
    const openRouter = SUB_PROCESSORS.find((sp) => sp.name === 'OpenRouter')
    expect(openRouter?.aiProcessingMode).toBe('when_configured')
    expect(openRouter?.service).toMatch(/when configured/i)
    expect(openRouter?.service).toMatch(/BYOK-only/i)

    for (const name of ['Anthropic', 'OpenAI', 'OpenRouter'] as const) {
      const row = SUB_PROCESSORS.find((sp) => sp.name === name)
      expect(row?.aiProcessingMode).toBe('when_configured')
    }
  })

  it('explains platform vs customer BYOK subprocessor scope', () => {
    expect(AI_SUBPROCESSOR_BYOK_NOTE).toMatch(/bring-your-own-key/i)
    expect(AI_SUBPROCESSOR_BYOK_NOTE).toMatch(/not automatically a Lextures sub-processor/i)
    expect(AI_SUBPROCESSOR_BYOK_NOTE).toMatch(/when configured/i)
  })
})
