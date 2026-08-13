import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import { BuildArticleWithAiModal } from '../build-article-with-ai-modal'

describe('BuildArticleWithAiModal', () => {
  it('generates a draft from the prompt and closes', async () => {
    const user = userEvent.setup()
    const onBuild = vi.fn().mockResolvedValue({
      title: 'Why grades belong in one place',
      description: 'A short summary.',
      bodyMd: ':::key-takeaways\n- One\n:::',
      primaryQuestion: 'Why keep grades in one place?',
      cluster: 'Grading',
      pillar: 'Product',
      keywords: ['grades'],
    })
    const onBuilt = vi.fn()
    const onClose = vi.fn()

    render(
      <BuildArticleWithAiModal
        open
        kind="blog"
        existingTitle=""
        existingBodyMd=""
        onClose={onClose}
        onBuild={onBuild}
        onBuilt={onBuilt}
      />,
    )

    expect(screen.getByRole('dialog', { name: /build with ai/i })).toBeInTheDocument()
    const generate = screen.getByRole('button', { name: 'Generate' })
    expect(generate).toBeDisabled()

    await user.type(screen.getByLabelText(/what should this article cover/i), 'Write about grades')
    await user.click(generate)

    expect(onBuild).toHaveBeenCalledWith('Write about grades')
    expect(onBuilt).toHaveBeenCalledWith(expect.objectContaining({ title: 'Why grades belong in one place' }))
    expect(onClose).toHaveBeenCalled()
  })

  it('shows an error when generation fails', async () => {
    const user = userEvent.setup()
    render(
      <BuildArticleWithAiModal
        open
        kind="blog"
        existingTitle=""
        existingBodyMd=""
        onClose={vi.fn()}
        onBuild={vi.fn().mockRejectedValue(new Error('AI is not configured. Configure AI under Settings → Intelligence.'))}
        onBuilt={vi.fn()}
      />,
    )

    await user.type(screen.getByLabelText(/what should this article cover/i), 'A topic')
    await user.click(screen.getByRole('button', { name: 'Generate' }))

    expect(await screen.findByText(/AI is not configured/i)).toBeInTheDocument()
  })
})
