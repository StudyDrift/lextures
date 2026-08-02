import { describe, expect, it, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { MutableRefObject } from 'react'
import { ToolConfigPanel } from '../tool-config-panel'
import type { ContentToolInstance, ContentToolManifest } from '../../../../lib/courses-api'

const patchContentToolInstance = vi.fn()
const fetchContentToolManifest = vi.fn()

vi.mock('../../../../lib/courses-api', async () => {
  const actual = await vi.importActual<typeof import('../../../../lib/courses-api')>(
    '../../../../lib/courses-api',
  )
  return {
    ...actual,
    patchContentToolInstance: (...args: unknown[]) => patchContentToolInstance(...args),
    fetchContentToolManifest: (...args: unknown[]) => fetchContentToolManifest(...args),
  }
})

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string, opts?: { defaultValue?: string; name?: string }) =>
      opts?.name ? `${opts.defaultValue ?? key} ${opts.name}` : (opts?.defaultValue ?? key),
  }),
}))

vi.mock('../editors/registry', () => ({
  resolveCustomEditor: () => null,
}))

const manifest = {
  id: 'inline_questions',
  version: '1',
  name: 'Inline questions',
  category: 'check',
  capabilities: [],
  configSchema: {
    type: 'object',
    properties: {
      title: { type: 'string', title: 'Title' },
    },
    required: ['title'],
  },
  stateSchema: { type: 'object' },
  ui: {
    renderer: 'inline',
    icon: 'help',
    group: 'check',
  },
} as ContentToolManifest

const instance = {
  id: 'inst-1',
  toolId: 'inline_questions',
  toolVersion: '1',
  title: 'Check',
  config: { title: 'Original' },
  status: 'active',
  hostKind: 'content_page',
  sectionKey: null,
  updatedAt: '2026-01-01T00:00:00Z',
} as ContentToolInstance

describe('ToolConfigPanel flush', () => {
  beforeEach(() => {
    patchContentToolInstance.mockReset()
    fetchContentToolManifest.mockReset()
    fetchContentToolManifest.mockResolvedValue(manifest)
    patchContentToolInstance.mockImplementation(
      async (_course: string, id: string, body: { config: Record<string, unknown> }) => ({
        ...instance,
        id,
        config: body.config,
      }),
    )
  })

  it('flushRef persists dirty config when host document save runs', async () => {
    const user = userEvent.setup()
    const flushRef: MutableRefObject<(() => Promise<void>) | null> = { current: null }
    const onDraftChange = vi.fn()
    const onSaved = vi.fn()

    render(
      <ToolConfigPanel
        courseCode="CS101"
        instance={instance}
        manifestCache={manifest}
        onDraftChange={onDraftChange}
        onSaved={onSaved}
        flushRef={flushRef}
      />,
    )

    const titleInput = await screen.findByLabelText(/title/i)
    await user.clear(titleInput)
    await user.type(titleInput, 'Updated title')

    expect(flushRef.current).toBeTypeOf('function')
    await flushRef.current!()

    await waitFor(() => {
      expect(patchContentToolInstance).toHaveBeenCalledWith('CS101', 'inst-1', {
        config: expect.objectContaining({ title: 'Updated title' }),
      })
    })
    expect(onSaved).toHaveBeenCalled()
  })

  it('flushRef is a no-op when config is unchanged', async () => {
    const flushRef: MutableRefObject<(() => Promise<void>) | null> = { current: null }

    render(
      <ToolConfigPanel
        courseCode="CS101"
        instance={instance}
        manifestCache={manifest}
        flushRef={flushRef}
      />,
    )

    await screen.findByLabelText(/title/i)
    await flushRef.current!()
    expect(patchContentToolInstance).not.toHaveBeenCalled()
  })

  it('paste JSON config saves valid config and updates the form', async () => {
    const user = userEvent.setup()
    const onSaved = vi.fn()
    const onDraftChange = vi.fn()

    render(
      <ToolConfigPanel
        courseCode="CS101"
        instance={instance}
        manifestCache={manifest}
        onSaved={onSaved}
        onDraftChange={onDraftChange}
      />,
    )

    await screen.findByLabelText(/title/i)
    await user.click(screen.getByRole('button', { name: /pasteJson$/i }))

    const dialog = await screen.findByRole('dialog')
    const textarea = dialog.querySelector('textarea')
    expect(textarea).toBeTruthy()
    await user.clear(textarea!)
    await user.click(textarea!)
    await user.paste('{"title":"From paste"}')

    await user.click(screen.getByRole('button', { name: /pasteJsonApply/i }))

    await waitFor(() => {
      expect(patchContentToolInstance).toHaveBeenCalledWith('CS101', 'inst-1', {
        config: { title: 'From paste' },
      })
    })
    expect(onSaved).toHaveBeenCalled()
    await waitFor(() => {
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
    expect(await screen.findByLabelText(/title/i)).toHaveValue('From paste')
  })

  it('paste JSON config shows client validation errors without saving', async () => {
    const user = userEvent.setup()

    render(
      <ToolConfigPanel
        courseCode="CS101"
        instance={instance}
        manifestCache={manifest}
      />,
    )

    await screen.findByLabelText(/title/i)
    await user.click(screen.getByRole('button', { name: /pasteJson$/i }))

    const dialog = await screen.findByRole('dialog')
    const textarea = dialog.querySelector('textarea')
    await user.clear(textarea!)
    await user.click(textarea!)
    await user.paste('{}')

    await user.click(screen.getByRole('button', { name: /pasteJsonApply/i }))

    expect(await screen.findByText(/this field is required/i)).toBeInTheDocument()
    expect(patchContentToolInstance).not.toHaveBeenCalled()
    expect(screen.getByRole('dialog')).toBeInTheDocument()
  })
})
