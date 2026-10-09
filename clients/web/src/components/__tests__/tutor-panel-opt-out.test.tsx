import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { AiTutorMenu } from '../tutor-panel'

const { fetchTutorMenuBlocked } = vi.hoisted(() => ({
  fetchTutorMenuBlocked: vi.fn(),
}))

vi.mock('../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({ ffPersistentTutor: false }),
}))

vi.mock('../../lib/tutor-api', async () => {
  const actual = await vi.importActual<typeof import('../../lib/tutor-api')>('../../lib/tutor-api')
  return { ...actual, fetchTutorMenuBlocked }
})

describe('AiTutorMenu account opt-out', () => {
  it('hides the tutor button when the account opted out of AI processing or the tutor', async () => {
    fetchTutorMenuBlocked.mockResolvedValue(true)
    render(<AiTutorMenu courseCode="HEB101" />)
    await waitFor(() => expect(fetchTutorMenuBlocked).toHaveBeenCalled())
    expect(screen.queryByRole('button', { name: 'Open AI Tutor' })).toBeNull()
  })

  it('shows the tutor button when neither opt-out is set', async () => {
    fetchTutorMenuBlocked.mockResolvedValue(false)
    render(<AiTutorMenu courseCode="HEB101" />)
    expect(await screen.findByRole('button', { name: 'Open AI Tutor' })).toBeInTheDocument()
  })
})
