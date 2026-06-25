import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { RunAgentPopover } from '../run-agent-popover'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

const defaultProps = {
  disabled: false,
  tooltip: null,
  dryRunDisabled: false,
  dryRunTooltip: null,
  dryRunning: false,
  batchRunning: false,
  runScope: 'ungraded' as const,
  setRunScope: vi.fn(),
  confirmOverwrite: false,
  setConfirmOverwrite: vi.fn(),
  runProgress: null,
  autoGradeNew: true,
  postPolicy: 'draft' as const,
  suggestModeEnabled: true,
  runMode: 'suggest' as const,
  setRunMode: vi.fn(),
  saving: false,
  onDryRun: vi.fn(),
  onToggleAutoGrade: vi.fn(),
  onTogglePostPolicy: vi.fn(),
  onSetConfidenceFloor: vi.fn(),
  onRun: vi.fn(),
}

describe('RunAgentPopover', () => {
  it('opens run settings in a popover when the trigger is clicked', () => {
    render(<RunAgentPopover {...defaultProps} />)

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.run.start' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('gradingAgent.run.scopeLabel')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.run.execute' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'gradingAgent.dryRun' })).toBeInTheDocument()
  })

  it('calls onDryRun from the dry run button', () => {
    const onDryRun = vi.fn()
    render(<RunAgentPopover {...defaultProps} onDryRun={onDryRun} />)

    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.run.start' }))
    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.dryRun' }))
    expect(onDryRun).toHaveBeenCalledTimes(1)
  })

  it('calls onRun from the inner run button', () => {
    const onRun = vi.fn()
    render(<RunAgentPopover {...defaultProps} onRun={onRun} />)

    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.run.start' }))
    fireEvent.click(screen.getByRole('button', { name: 'gradingAgent.run.execute' }))
    expect(onRun).toHaveBeenCalledTimes(1)
  })
})
