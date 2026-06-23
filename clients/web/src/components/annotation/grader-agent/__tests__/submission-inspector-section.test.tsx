import { describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen } from '@testing-library/react'
import { SubmissionInspectorSection } from '../submission-inspector-section'

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('../../../file-preview', () => ({
  FilePreviewBody: ({ filename }: { filename: string }) => (
    <div data-testid="file-preview-body">{filename}</div>
  ),
  FilePreview: ({ open, filename }: { open: boolean; filename: string }) =>
    open ? <div data-testid="file-preview-modal">{filename}</div> : null,
}))

describe('SubmissionInspectorSection', () => {
  it('prompts to choose a student when no submission is selected', () => {
    render(<SubmissionInspectorSection submission={null} />)
    expect(screen.getByText('gradingAgent.canvas.inspector.submissionNoStudent')).toBeInTheDocument()
  })

  it('lists submission files and previews the selected file', () => {
    render(
      <SubmissionInspectorSection
        submission={{
          id: 'sub-1',
          attachmentFileId: 'file-a',
          attachments: [
            {
              fileId: 'file-a',
              filename: 'essay.pdf',
              mimeType: 'application/pdf',
              contentPath: '/api/v1/files/a',
            },
            {
              fileId: 'file-b',
              filename: 'notes.txt',
              mimeType: 'text/plain',
              contentPath: '/api/v1/files/b',
            },
          ],
        }}
      />,
    )

    expect(screen.getByRole('button', { name: 'essay.pdf' })).toHaveAttribute('aria-current', 'true')
    expect(screen.getByTestId('file-preview-body')).toHaveTextContent('essay.pdf')

    fireEvent.click(screen.getByRole('button', { name: 'notes.txt' }))
    expect(screen.getByRole('button', { name: 'notes.txt' })).toHaveAttribute('aria-current', 'true')
    expect(screen.getByTestId('file-preview-body')).toHaveTextContent('notes.txt')
  })

  it('opens a full preview modal from the expand control', () => {
    render(
      <SubmissionInspectorSection
        submission={{
          id: 'sub-1',
          attachmentFileId: 'file-a',
          attachments: [
            {
              fileId: 'file-a',
              filename: 'essay.pdf',
              mimeType: 'application/pdf',
              contentPath: '/api/v1/files/a',
            },
          ],
        }}
      />,
    )

    fireEvent.click(
      screen.getByRole('button', { name: 'gradingAgent.canvas.inspector.submissionExpandPreview' }),
    )
    expect(screen.getByTestId('file-preview-modal')).toHaveTextContent('essay.pdf')
  })
})