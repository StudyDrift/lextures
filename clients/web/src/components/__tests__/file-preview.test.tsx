import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/mocks/server'
import { FilePreview } from '../file-preview'

// Mock PdfViewer — it relies on PDF.js canvas APIs unavailable in jsdom
vi.mock('../pdf-viewer', () => ({
  PdfViewer: ({ filePath, filename }: { filePath: string; filename: string }) => (
    <div data-testid="pdf-viewer" data-filepath={filePath} data-filename={filename}>
      PDF Viewer Mock
    </div>
  ),
}))

vi.mock('../office-html-preview', () => ({
  OfficeHtmlPreview: ({ filename }: { filename: string }) => (
    <div data-testid="office-html-preview">{filename}</div>
  ),
}))

vi.mock('../text-file-preview', () => ({
  TextFilePreview: ({ filename }: { filename: string }) => (
    <div data-testid="text-file-preview">{filename}</div>
  ),
}))

const PNG_1X1 = new Uint8Array([
  0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0x00, 0x00, 0x00, 0x0d,
  0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
  0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53, 0xde, 0x00, 0x00, 0x00,
  0x0c, 0x49, 0x44, 0x41, 0x54, 0x08, 0xd7, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
  0x00, 0x00, 0x02, 0x00, 0x01, 0xe2, 0x21, 0xbc, 0x33, 0x00, 0x00, 0x00,
  0x00, 0x49, 0x45, 0x4e, 0x44, 0xae, 0x42, 0x60, 0x82,
])

const FILE_PATH = '/api/v1/courses/C-TEST01/course-files/00000000-0000-0000-0000-000000000001/content'

const defaultProps = {
  open: true,
  filePath: FILE_PATH,
  filename: 'test.pdf',
  mimeType: 'application/pdf',
  onClose: vi.fn(),
}

describe('FilePreview', () => {
  beforeEach(() => {
    defaultProps.onClose = vi.fn()
  })

  it('renders nothing when open=false', () => {
    render(<FilePreview {...defaultProps} open={false} />)
    expect(screen.queryByRole('dialog')).toBeNull()
  })

  it('renders modal dialog with aria attributes when open', () => {
    render(<FilePreview {...defaultProps} />)
    const dialog = screen.getByRole('dialog')
    expect(dialog).toBeInTheDocument()
    expect(dialog).toHaveAttribute('aria-modal', 'true')
    const titleId = dialog.getAttribute('aria-labelledby')
    expect(titleId).toBeTruthy()
    expect(document.getElementById(titleId!)).toHaveTextContent('test.pdf')
  })

  it('renders PdfViewer for PDF mime type', () => {
    render(<FilePreview {...defaultProps} mimeType="application/pdf" filename="test.pdf" />)
    expect(screen.getByTestId('pdf-viewer')).toBeInTheDocument()
    expect(screen.getByTestId('pdf-viewer')).toHaveAttribute('data-filepath', FILE_PATH)
  })

  it('renders PdfViewer for .pdf extension with no mime type', () => {
    render(<FilePreview {...defaultProps} mimeType={null} filename="report.pdf" />)
    expect(screen.getByTestId('pdf-viewer')).toBeInTheDocument()
  })

  it('renders server HTML office preview for DOCX files', () => {
    render(<FilePreview {...defaultProps} mimeType="application/vnd.openxmlformats-officedocument.wordprocessingml.document" filename="document.docx" />)
    expect(screen.queryByTestId('pdf-viewer')).toBeNull()
    expect(screen.getByTestId('office-html-preview')).toHaveTextContent('document.docx')
    expect(screen.queryByText(/cannot be previewed/i)).toBeNull()
  })

  it('renders text preview for .txt files', () => {
    render(<FilePreview {...defaultProps} mimeType="text/plain" filename="notes.txt" />)
    expect(screen.getByTestId('text-file-preview')).toHaveTextContent('notes.txt')
    expect(screen.queryByText(/cannot be previewed/i)).toBeNull()
  })

  it('renders text preview for .md files', () => {
    render(<FilePreview {...defaultProps} mimeType={null} filename="readme.md" />)
    expect(screen.getByTestId('text-file-preview')).toHaveTextContent('readme.md')
  })

  it('renders download button for truly unsupported file types', () => {
    render(<FilePreview {...defaultProps} mimeType="application/octet-stream" filename="archive.zip" />)
    expect(screen.queryByTestId('pdf-viewer')).toBeNull()
    expect(screen.getByRole('button', { name: /download to view/i })).toBeInTheDocument()
    expect(screen.getByText(/cannot be previewed/i)).toBeInTheDocument()
  })

  it('closes on Escape key press', () => {
    render(<FilePreview {...defaultProps} />)
    fireEvent.keyDown(window, { key: 'Escape' })
    expect(defaultProps.onClose).toHaveBeenCalledOnce()
  })

  it('closes when close button is clicked', () => {
    render(<FilePreview {...defaultProps} />)
    fireEvent.click(screen.getByRole('button', { name: /close preview/i }))
    expect(defaultProps.onClose).toHaveBeenCalledOnce()
  })

  it('closes when backdrop is clicked', () => {
    render(<FilePreview {...defaultProps} />)
    const backdrop = screen.getByRole('button', { name: /close file preview backdrop/i })
    fireEvent.click(backdrop)
    expect(defaultProps.onClose).toHaveBeenCalledOnce()
  })

  it('shows filename in dialog title', () => {
    render(<FilePreview {...defaultProps} filename="annual-report-2024.pdf" />)
    expect(screen.getByText('annual-report-2024.pdf')).toBeInTheDocument()
  })

  describe('image viewer', () => {
    beforeEach(() => {
      server.use(
        http.get('http://localhost:8080' + FILE_PATH, () =>
          new HttpResponse(PNG_1X1, {
            headers: { 'Content-Type': 'image/png' },
          }),
        ),
      )
    })

    it('renders image viewer controls for image mime type', async () => {
      render(
        <FilePreview
          {...defaultProps}
          mimeType="image/png"
          filename="photo.png"
        />,
      )

      // Loading state initially
      expect(screen.getByRole('status', { name: /loading image/i })).toBeInTheDocument()

      // After fetch, image controls appear
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /zoom in/i })).toBeInTheDocument()
      })
      expect(screen.getByRole('button', { name: /zoom out/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /reset zoom/i })).toBeInTheDocument()
      expect(screen.getByRole('button', { name: /download photo\.png/i })).toBeInTheDocument()
    })

    it('renders image viewer for .png extension with null mime type', async () => {
      render(
        <FilePreview
          {...defaultProps}
          mimeType={null}
          filename="photo.png"
        />,
      )
      await waitFor(() => {
        expect(screen.getByRole('button', { name: /zoom in/i })).toBeInTheDocument()
      })
    })

    it('shows error state when image fetch fails', async () => {
      server.use(
        http.get('http://localhost:8080' + FILE_PATH, () =>
          new HttpResponse(null, { status: 500 }),
        ),
      )
      render(
        <FilePreview
          {...defaultProps}
          mimeType="image/jpeg"
          filename="photo.jpg"
        />,
      )
      await waitFor(() => {
        expect(screen.getByRole('alert')).toHaveTextContent(/could not load/i)
      })
    })
  })
})
