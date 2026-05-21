import { describe, expect, it, vi, beforeEach } from 'vitest'
import { DropboxPicker } from './dropbox'
import { GoogleDrivePicker } from './google-drive'
import { OneDrivePicker } from './onedrive'
import type { PickedFile } from './types'

// Shared script loader mock — prevents DOM manipulation in tests
vi.mock('./dropbox', async (importOriginal) => {
  const original = await importOriginal<typeof import('./dropbox')>()
  return original
})

describe('DropboxPicker', () => {
  beforeEach(() => {
    // Reset window.Dropbox before each test
    ;(window as unknown as Record<string, unknown>).Dropbox = undefined
  })

  it('has provider = dropbox', () => {
    const p = new DropboxPicker()
    expect(p.provider).toBe('dropbox')
  })

  it('resolves with PickedFile when Dropbox.choose calls success', async () => {
    const mockFile = {
      id: 'dbx-id-123',
      name: 'lecture-notes.docx',
      link: 'https://www.dropbox.com/s/abc/lecture-notes.docx',
      icon: 'https://www.dropbox.com/icon.png',
      isDir: false,
    }
    ;(window as unknown as Record<string, unknown>).Dropbox = {
      choose: (opts: { success: (files: typeof mockFile[]) => void }) => {
        opts.success([mockFile])
      },
    }

    const picker = new DropboxPicker()
    // loadScript is called internally; mock the script injection
    const origCreate = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      if (tag === 'script') {
        const el = origCreate(tag)
        // Trigger onload synchronously
        setTimeout(() => (el as HTMLScriptElement).onload?.(new Event('load')), 0)
        return el
      }
      return origCreate(tag)
    })
    vi.spyOn(document.head, 'appendChild').mockImplementation(() => document.head)

    const result: PickedFile | null = await picker.pick()
    expect(result).not.toBeNull()
    expect(result?.provider).toBe('dropbox')
    expect(result?.name).toBe('lecture-notes.docx')
    expect(result?.viewUrl).toBe(mockFile.link)
  })

  it('resolves with null when Dropbox.choose calls cancel', async () => {
    ;(window as unknown as Record<string, unknown>).Dropbox = {
      choose: (opts: { cancel: () => void }) => {
        opts.cancel()
      },
    }

    const picker = new DropboxPicker()
    const origCreate = document.createElement.bind(document)
    vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
      if (tag === 'script') {
        const el = origCreate(tag)
        setTimeout(() => (el as HTMLScriptElement).onload?.(new Event('load')), 0)
        return el
      }
      return origCreate(tag)
    })
    vi.spyOn(document.head, 'appendChild').mockImplementation(() => document.head)

    const result = await picker.pick()
    expect(result).toBeNull()
  })
})

describe('GoogleDrivePicker', () => {
  it('has provider = google_drive', () => {
    const p = new GoogleDrivePicker('client-id', 'api-key')
    expect(p.provider).toBe('google_drive')
  })
})

describe('OneDrivePicker', () => {
  it('has provider = onedrive', () => {
    const p = new OneDrivePicker('client-id')
    expect(p.provider).toBe('onedrive')
  })
})

describe('PickedFile shape', () => {
  it('has all required fields', () => {
    const f: PickedFile = {
      provider: 'google_drive',
      externalId: 'abc123',
      name: 'slides.pptx',
      viewUrl: 'https://docs.google.com/presentation/d/abc123/edit',
      iconUrl: 'https://drive.google.com/icon.png',
      mimeType: 'application/vnd.google-apps.presentation',
    }
    expect(f.provider).toBe('google_drive')
    expect(f.externalId).toBe('abc123')
    expect(f.viewUrl).toContain('docs.google.com')
  })
})
