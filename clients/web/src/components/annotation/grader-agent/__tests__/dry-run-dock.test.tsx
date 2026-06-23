import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { DryRunDock } from '../dry-run-dock'
import type { GraderAgentWorkflowState } from '../use-grader-agent-workflow'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

const baseWorkflow = {
  dryRunResult: {
    suggestedPoints: 12,
    comment: 'Nice work',
    confidence: 0.8,
  },
  setDryRunResult: vi.fn(),
  saving: false,
  dryRunning: false,
  handleApply: vi.fn(),
  handleDryRun: vi.fn(),
} as unknown as GraderAgentWorkflowState

describe('DryRunDock', () => {
  it('lays out console and preview side by side above the status bar', () => {
    const { container } = render(
      <DryRunDock
        workflow={baseWorkflow}
        rubric={null}
        maxPoints={100}
        submissionId="sub-1"
        consoleOpen
        logs={[{ message: 'Starting dry run…', level: 'info' }]}
        running={false}
      />,
    )
    expect(screen.getByRole('toolbar')).toBeInTheDocument()
    expect(screen.getByText('gradingAgent.dryRun.console.title')).toBeInTheDocument()
    expect(screen.getByText('gradingAgent.result.title')).toBeInTheDocument()
    expect(container.querySelector('[role="separator"]')).toBeInTheDocument()
    expect(container.querySelector('.lg\\:grid-cols-2')).toBeInTheDocument()
  })

  it('places the resize handle above the expanded panels', () => {
    const { container } = render(
      <DryRunDock
        workflow={baseWorkflow}
        rubric={null}
        maxPoints={100}
        submissionId="sub-1"
        consoleOpen
        logs={[{ message: 'Starting dry run…', level: 'info' }]}
        running={false}
      />,
    )
    const separator = container.querySelector('[role="separator"]')
    const toolbar = screen.getByRole('toolbar')
    const grid = container.querySelector('.lg\\:grid-cols-2')
    expect(separator).not.toBeNull()
    expect(grid).not.toBeNull()
    expect(separator!.compareDocumentPosition(toolbar!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
    expect(separator!.compareDocumentPosition(grid!) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy()
  })

  it('collapses panels into the bottom status bar', () => {
    render(
      <DryRunDock
        workflow={baseWorkflow}
        rubric={null}
        maxPoints={100}
        submissionId="sub-1"
        consoleOpen
        logs={[{ message: 'Hidden after collapse', level: 'info' }]}
        running={false}
      />,
    )
    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.dryRun.console.collapse' }))
    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.result.collapse' }))
    expect(screen.queryByRole('log')).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'gradingAgent.apply' })).not.toBeInTheDocument()
    expect(screen.getByRole('toolbar')).toHaveTextContent('Hidden after collapse')
    expect(screen.getByRole('toolbar')).toHaveTextContent('12 / 100')
  })
})