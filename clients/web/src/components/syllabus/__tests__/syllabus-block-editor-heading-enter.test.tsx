import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { describe, expect, it, vi } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'
import type { SyllabusSection } from '../../../lib/courses-api'
import { SyllabusBlockEditor } from '../syllabus-block-editor'

vi.mock('../../../hooks/use-speech-to-text-availability', () => ({
  useSpeechToTextAvailability: () => ({
    enabled: false,
    language: 'en-US',
    loading: false,
  }),
}))

vi.mock('../../../context/course-nav-features-context', () => ({
  useCourseNavFeatures: () => ({
    visualBoardsEnabled: false,
  }),
}))

vi.mock('../../../lib/platform-features', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../lib/platform-features')>()
  return {
    ...actual,
    altTextEnforcementFeatureEnabled: () => false,
    altTextHardBlockEnabled: () => false,
    speechToTextFeatureEnabled: () => false,
  }
})

function EditorHarness() {
  const [sections, setSections] = useState<SyllabusSection[]>([
    { id: 'sec-1', heading: 'Intro', markdown: 'Body text here.' },
  ])
  return (
    <MemoryRouter>
      <I18nProvider>
        <SyllabusBlockEditor sections={sections} onChange={setSections} />
      </I18nProvider>
    </MemoryRouter>
  )
}

describe('SyllabusBlockEditor — heading Enter focuses content', () => {
  it('moves focus from the section heading into the body editor on Enter', async () => {
    const user = userEvent.setup()
    render(<EditorHarness />)

    const heading = await screen.findByRole('textbox', { name: /section heading/i })
    await user.click(heading)
    expect(heading).toHaveFocus()

    await user.keyboard('{Enter}')

    await waitFor(() => {
      const body = document.querySelector('#canvas-md-sec-1 [contenteditable="true"]')
      expect(body).toBeTruthy()
      expect(body).toHaveFocus()
    })
  })
})
