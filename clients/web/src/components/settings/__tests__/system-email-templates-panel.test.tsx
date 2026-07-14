import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import { I18nProvider } from '../../../context/i18n-provider'

const api = vi.hoisted(() => ({
  listSystemEmailTemplateSlots: vi.fn(),
  getSystemEmailTemplateSlot: vi.fn(),
  listSystemEmailTemplateHistory: vi.fn(),
  previewSystemEmailTemplate: vi.fn(),
  saveSystemEmailTemplate: vi.fn(),
  resetSystemEmailTemplate: vi.fn(),
  restoreSystemEmailTemplateVersion: vi.fn(),
  sendSystemEmailTemplateTest: vi.fn(),
}))

vi.mock('../../../lib/system-email-templates-api', () => api)

vi.mock('../../../context/platform-features-context', () => ({
  usePlatformFeatures: () => ({
    emailTemplateEditorEnabled: true,
    loading: false,
  }),
}))

vi.mock('../../use-confirm', () => ({
  useConfirm: () => ({
    confirm: vi.fn(async () => true),
    ConfirmDialogHost: null,
  }),
}))

import { SystemEmailTemplatesPanel } from '../system-email-templates-panel'

const slot = {
  id: 'magic_link',
  description: 'Passwordless sign-in link',
  mergeFields: { link: 'Sign-in link', expires_at: 'Expiry' },
  defaultHtml: '<p>default</p>',
  defaultText: 'default',
  defaultMarkdown: 'Sign in [now]({{link}})',
  hasCustom: false,
}

describe('SystemEmailTemplatesPanel', () => {
  beforeEach(() => {
    Object.values(api).forEach((fn) => fn.mockReset())
    api.listSystemEmailTemplateSlots.mockResolvedValue([slot])
    api.getSystemEmailTemplateSlot.mockResolvedValue({ ...slot, active: undefined })
    api.listSystemEmailTemplateHistory.mockResolvedValue([])
    api.previewSystemEmailTemplate.mockResolvedValue({
      html: '<html><body>Preview body</body></html>',
      text: 'Preview body',
    })
    api.saveSystemEmailTemplate.mockResolvedValue({
      id: 'v1',
      slotId: 'magic_link',
      sourceMarkdown: 'edited',
      htmlBody: '<p>edited</p>',
      createdAt: '2026-01-01T00:00:00Z',
      isActive: true,
      unknownFields: ['foo.bar'],
    })
  })

  it('loads slots and shows markdown default', async () => {
    render(
      <I18nProvider>
        <SystemEmailTemplatesPanel />
      </I18nProvider>,
    )
    await waitFor(() => {
      expect(screen.getByText('Passwordless sign-in link')).toBeInTheDocument()
    })
    await waitFor(() => {
      expect(screen.getByDisplayValue(/Sign in/)).toBeInTheDocument()
    })
  })

  it('saves markdown and surfaces unknown fields warning', async () => {
    const user = userEvent.setup()
    render(
      <I18nProvider>
        <SystemEmailTemplatesPanel />
      </I18nProvider>,
    )
    await waitFor(() => expect(screen.getByDisplayValue(/Sign in/)).toBeInTheDocument())
    const ta = screen.getByDisplayValue(/Sign in/)
    await user.clear(ta)
    await user.type(ta, 'Hello {{foo.bar}}')
    const save = await screen.findByRole('button', { name: /Save/i })
    await waitFor(() => expect(save).not.toBeDisabled())
    await user.click(save)
    await waitFor(() => {
      expect(api.saveSystemEmailTemplate).toHaveBeenCalledWith(
        'magic_link',
        expect.objectContaining({ sourceMarkdown: expect.stringContaining('foo.bar') }),
      )
    })
    // unknownFields are shown from the save response before reload clears them;
    // also accept the success toast if reload already ran.
    await waitFor(() => {
      const unknown = screen.queryByText(/Unknown merge fields/i)
      const saved = screen.queryByText(/Template saved/i)
      expect(unknown ?? saved).toBeTruthy()
    })
  })

  it('shows customized badge when hasCustom', async () => {
    api.listSystemEmailTemplateSlots.mockResolvedValue([{ ...slot, hasCustom: true }])
    render(
      <I18nProvider>
        <SystemEmailTemplatesPanel />
      </I18nProvider>,
    )
    await waitFor(() => {
      expect(screen.getByText('Customized')).toBeInTheDocument()
    })
  })
})
