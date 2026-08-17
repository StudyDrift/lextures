import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'
import type { MarketingFinding } from '../../../../lib/marketing-content-api'
import { ArticleFindingsBar } from '../article-findings-bar'

vi.mock('../../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffMotionControls: false }),
}))

const findings: MarketingFinding[] = [
  { rule: 'fm.cluster', severity: 'error', message: 'Required metadata field is missing.', path: 'cluster' },
  { rule: 'struct.faq-count', severity: 'warning', message: 'FAQ must contain 3–6 questions; found 0.', line: 1 },
]

describe('ArticleFindingsBar', () => {
  it('renders finding messages and jumps to the selected finding', async () => {
    const user = userEvent.setup()
    const onSelectFinding = vi.fn()
    const onInsertTemplate = vi.fn()
    render(
      <ArticleFindingsBar
        findings={findings}
        score={1}
        validating={false}
        open
        onOpenChange={vi.fn()}
        bodyMd="# Title"
        onSelectFinding={onSelectFinding}
        onInsertTemplate={onInsertTemplate}
      />,
    )

    expect(screen.getByText('Required metadata field is missing.')).toBeInTheDocument()
    expect(screen.getByText('FAQ must contain 3–6 questions; found 0.')).toBeInTheDocument()
    expect(screen.queryByText('5 blocking')).not.toBeInTheDocument()
    expect(screen.getByText('1 blocking')).toBeInTheDocument()
    expect(screen.getByText('1 suggestion')).toBeInTheDocument()
    expect(screen.getByText('/ 8.0 floor')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /Required metadata field is missing/i }))
    expect(onSelectFinding).toHaveBeenCalledWith(findings[0])

    await user.click(screen.getByRole('button', { name: 'Insert block' }))
    expect(onInsertTemplate).toHaveBeenCalledWith(expect.stringContaining(':::faq'))
  })

  it('solves every finding, including warnings, when Solve with AI is clicked', async () => {
    const user = userEvent.setup()
    const onSolveWithAI = vi.fn()
    render(
      <ArticleFindingsBar
        findings={findings}
        score={1}
        validating={false}
        open
        onOpenChange={vi.fn()}
        bodyMd="# Title"
        onSelectFinding={vi.fn()}
        onInsertTemplate={vi.fn()}
        canSolve
        onSolveWithAI={onSolveWithAI}
      />,
    )

    await user.click(screen.getByRole('button', { name: 'Solve with AI' }))
    expect(onSolveWithAI).toHaveBeenCalledTimes(1)
  })

  it('shows progress on the finding currently being solved', () => {
    render(
      <ArticleFindingsBar
        findings={findings}
        score={1}
        validating={false}
        open
        onOpenChange={vi.fn()}
        bodyMd="# Title"
        onSelectFinding={vi.fn()}
        onInsertTemplate={vi.fn()}
        canSolve
        solving
        solvingFindingKey="struct.faq-count-1"
        solveProgress="Solving 2 of 2: FAQ must contain 3–6 questions; found 0."
        onSolveWithAI={vi.fn()}
      />,
    )
    expect(screen.getByRole('status')).toHaveTextContent('Solving 2 of 2')
    expect(screen.getByText('Solving')).toBeInTheDocument()
  })

  it('hides Solve with AI when the author cannot repair findings', () => {
    render(
      <ArticleFindingsBar
        findings={findings}
        score={1}
        validating={false}
        open
        onOpenChange={vi.fn()}
        bodyMd="# Title"
        onSelectFinding={vi.fn()}
        onInsertTemplate={vi.fn()}
      />,
    )
    expect(screen.queryByRole('button', { name: 'Solve with AI' })).not.toBeInTheDocument()
  })
})
