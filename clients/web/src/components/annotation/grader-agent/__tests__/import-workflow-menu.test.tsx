import { describe, expect, it, vi, beforeEach } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { ImportWorkflowMenu } from '../import-workflow-menu'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, vars?: Record<string, string>) =>
      vars ? `${key}:${JSON.stringify(vars)}` : key,
  }),
}))

vi.mock('../../../../lib/api', () => ({
  authorizedFetch: vi.fn(),
}))

vi.mock('../../../../lib/courses-api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../../../../lib/courses-api')>()
  return {
    ...actual,
    fetchCourseGradingAgentTemplates: vi.fn(),
    fetchCourseGradingAgents: vi.fn(),
    fetchGraderAgentTemplate: vi.fn(),
    fetchGraderAgentConfig: vi.fn(),
  }
})

describe('ImportWorkflowMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads templates for the current course in the import panel', async () => {
    const onImport = vi.fn()
    const { authorizedFetch } = await import('../../../../lib/api')
    const { fetchCourseGradingAgentTemplates } = await import('../../../../lib/courses-api')

    vi.mocked(authorizedFetch).mockResolvedValue({
      ok: true,
      json: async () => ({
        courses: [
          {
            id: 'course-1',
            courseCode: 'demo',
            title: 'Demo course',
            description: '',
            heroImageUrl: null,
            heroImageObjectPosition: null,
            startsAt: null,
            endsAt: null,
            visibleFrom: null,
            hiddenAt: null,
            published: true,
            archived: false,
            markdownThemePreset: 'default',
            markdownThemeCustom: null,
            gradingScale: 'letter',
          },
        ],
      }),
    } as Response)

    vi.mocked(fetchCourseGradingAgentTemplates).mockResolvedValue({
      templates: [{ id: 'template-1', name: 'Essay template', updatedAt: '2026-01-01T00:00:00.000Z' }],
    })

    render(
      <ImportWorkflowMenu
        courseCode="demo"
        itemId="assignment-1"
        itemKind="assignment"
        onImport={onImport}
      />,
    )

    fireEvent.click(screen.getByRole('button', { name: /gradingAgent.import.button/i }))
    expect(await screen.findByRole('dialog', { name: 'gradingAgent.import.panelLabel' })).toBeInTheDocument()

    await waitFor(() => {
      expect(fetchCourseGradingAgentTemplates).toHaveBeenCalledWith('demo')
    })

    const itemPicker = screen.getByRole('button', { name: /gradingAgent.import.templateLabel/i })
    fireEvent.click(itemPicker)
    expect(await screen.findByRole('menuitemradio', { name: 'Essay template' })).toBeInTheDocument()
  })
})