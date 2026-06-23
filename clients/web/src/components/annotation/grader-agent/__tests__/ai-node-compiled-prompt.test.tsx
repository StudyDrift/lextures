import { describe, expect, it, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { AiNodeCompiledPrompt } from '../ai-node-compiled-prompt'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

describe('AiNodeCompiledPrompt', () => {
  it('renders compiled prompt sections after dry run', () => {
    render(
      <AiNodeCompiledPrompt
        detail={{
          compiledSystemPrompt: 'Respond with ONLY valid JSON...',
          compiledPrompt: 'Grade this submission fairly.',
          compiledInput: '## Student Submission\nEssay text',
          compiledOutput: '{"total":8,"comment":"Good work","confidence":0.8}',
        }}
      />,
    )
    expect(screen.getByText('Grade this submission fairly.')).toBeInTheDocument()
    expect(screen.getByText(/Essay text/)).toBeInTheDocument()
    expect(screen.getByText(/Respond with ONLY valid JSON/)).toBeInTheDocument()
    expect(screen.getByText(/"total":8/)).toBeInTheDocument()
  })

  it('renders nothing when detail is empty', () => {
    const { container } = render(<AiNodeCompiledPrompt detail={{}} />)
    expect(container.firstChild).toBeNull()
  })
})