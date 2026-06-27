import { describe, expect, it, vi } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AiBuildPanel } from '../ai-build-panel'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

describe('AiBuildPanel', () => {
  it('expands and submits the instruction, then collapses on success', async () => {
    const user = userEvent.setup()
    const onBuild = vi.fn().mockResolvedValue(true)
    render(<AiBuildPanel building={false} onBuild={onBuild} />)

    await user.click(screen.getByText('gradingAgent.aiBuilder.open'))
    const textarea = screen.getByPlaceholderText('gradingAgent.aiBuilder.placeholder')
    await user.type(textarea, 'Give full points over 10')
    await user.click(screen.getByText('gradingAgent.aiBuilder.generate'))

    expect(onBuild).toHaveBeenCalledWith('Give full points over 10')
    // Collapses back to the open button after a successful build.
    await waitFor(() => expect(screen.getByText('gradingAgent.aiBuilder.open')).toBeInTheDocument())
  })

  it('keeps the instruction when the build fails', async () => {
    const user = userEvent.setup()
    const onBuild = vi.fn().mockResolvedValue(false)
    render(<AiBuildPanel building={false} onBuild={onBuild} />)

    await user.click(screen.getByText('gradingAgent.aiBuilder.open'))
    const textarea = screen.getByPlaceholderText('gradingAgent.aiBuilder.placeholder')
    await user.type(textarea, 'broken rule')
    await user.click(screen.getByText('gradingAgent.aiBuilder.generate'))

    await waitFor(() => expect(onBuild).toHaveBeenCalled())
    expect(screen.getByPlaceholderText('gradingAgent.aiBuilder.placeholder')).toHaveValue('broken rule')
  })

  it('does not submit an empty instruction', async () => {
    const user = userEvent.setup()
    const onBuild = vi.fn().mockResolvedValue(true)
    render(<AiBuildPanel building={false} onBuild={onBuild} />)

    await user.click(screen.getByText('gradingAgent.aiBuilder.open'))
    expect(screen.getByText('gradingAgent.aiBuilder.generate').closest('button')).toBeDisabled()
    expect(onBuild).not.toHaveBeenCalled()
  })
})
