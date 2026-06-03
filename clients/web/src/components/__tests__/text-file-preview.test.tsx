import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/mocks/server'
import { TextFilePreview } from '../text-file-preview'

const FILE_PATH = '/api/v1/courses/C-TEST01/course-files/00000000-0000-0000-0000-000000000001/content'

describe('TextFilePreview', () => {
  it('renders formatted preview by default for markdown files', async () => {
    server.use(
      http.get('http://localhost:8080' + FILE_PATH, () =>
        new HttpResponse('# Hello\n\nWorld', {
          headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
        }),
      ),
    )
    render(<TextFilePreview filePath={FILE_PATH} filename="readme.md" />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { level: 1, name: 'Hello' })).toBeInTheDocument()
    })
    expect(screen.getByText('World')).toBeInTheDocument()
    expect(screen.queryByText('# Hello')).toBeNull()
  })

  it('shows raw source when Source tab is selected', async () => {
    server.use(
      http.get('http://localhost:8080' + FILE_PATH, () =>
        new HttpResponse('# Hello\n\nWorld', {
          headers: { 'Content-Type': 'text/markdown; charset=utf-8' },
        }),
      ),
    )
    const user = userEvent.setup()
    render(<TextFilePreview filePath={FILE_PATH} filename="readme.md" />)
    await waitFor(() => {
      expect(screen.getByRole('heading', { name: 'Hello' })).toBeInTheDocument()
    })
    await user.click(screen.getByRole('tab', { name: 'Source' }))
    expect(screen.getByLabelText(/text preview of readme\.md/i)).toHaveTextContent('# Hello')
  })

  it('shows plain text only for .txt files (no tabs)', async () => {
    server.use(
      http.get('http://localhost:8080' + FILE_PATH, () =>
        new HttpResponse('plain line', {
          headers: { 'Content-Type': 'text/plain; charset=utf-8' },
        }),
      ),
    )
    render(<TextFilePreview filePath={FILE_PATH} filename="notes.txt" />)
    await waitFor(() => {
      expect(screen.getByLabelText(/text preview of notes\.txt/i)).toHaveTextContent('plain line')
    })
    expect(screen.queryByRole('tablist')).toBeNull()
  })

  it('shows error when fetch fails', async () => {
    server.use(
      http.get('http://localhost:8080' + FILE_PATH, () =>
        new HttpResponse(null, { status: 500 }),
      ),
    )
    render(<TextFilePreview filePath={FILE_PATH} filename="notes.txt" />)
    await waitFor(() => {
      expect(screen.getByRole('alert')).toHaveTextContent(/could not load/i)
    })
  })
})
