import { afterEach, describe, expect, it, vi } from 'vitest'
import {
  assertVersionCoversSchemaDiff,
  defineTool,
  renderTool,
  resolveWithinMajor,
  ToolPrompt,
  ToolShell,
} from '@lextures/tool-sdk'

describe('@lextures/tool-sdk (CT.5)', () => {
  const mounts: Array<{ unmount: () => void }> = []
  afterEach(() => {
    while (mounts.length) mounts.pop()?.unmount()
  })

  it('renderTool harness mounts without a server (AC-1)', async () => {
    const tool = defineTool(
      { id: 'demo', version: '1.0.0' },
      ({ state, save, t }) => (
        <ToolShell>
          <ToolPrompt>{t('prompt')}</ToolPrompt>
          <button
            type="button"
            data-testid="bump"
            onClick={() => save({ n: Number(state.n ?? 0) + 1 })}
          >
            bump
          </button>
          <span data-testid="n">{String(state.n ?? 0)}</span>
        </ToolShell>
      ),
    )
    const handle = renderTool(tool, { state: { n: 0 } })
    mounts.push(handle)
    expect(handle.container.querySelector('[data-tool-shell]')).toBeTruthy()
    handle.container.querySelector<HTMLButtonElement>('[data-testid="bump"]')?.click()
    await vi.waitFor(() => {
      expect(handle.getState().n).toBe(1)
    })
  })

  it('fails schema-diff CI when a required field is removed with only a minor bump (AC-2)', () => {
    expect(() =>
      assertVersionCoversSchemaDiff(
        '1.0.0',
        '1.1.0',
        {
          type: 'object',
          required: ['prompt'],
          properties: { prompt: { type: 'string' }, answerKey: { type: 'string' } },
        },
        {
          type: 'object',
          required: ['prompt'],
          properties: { prompt: { type: 'string' } },
        },
      ),
    ).toThrow(/answerKey/)
  })

  it('resolves within major and never crosses major (AC-10)', () => {
    expect(resolveWithinMajor('1.4.0', ['1.2.0', '1.5.1', '2.0.0'])).toBe('1.5.1')
  })
})
