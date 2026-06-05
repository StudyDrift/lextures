import type { CloudPickerProvider, PickedFile } from './types'

declare global {
  interface Window {
    Dropbox?: {
      choose(options: DropboxOptions): void
      isBrowserSupported(): boolean
    }
  }
}

interface DropboxOptions {
  success(files: DropboxFile[]): void
  cancel(): void
  linkType: 'preview' | 'direct'
  multiselect: boolean
  extensions?: string[]
  folderselect: boolean
}

interface DropboxFile {
  id: string
  name: string
  link: string
  icon: string
  isDir: boolean
}

function loadScript(src: string, appKey?: string): Promise<void> {
  return new Promise((resolve, reject) => {
    const selector = appKey
      ? `script[src="${src}"][data-app-key="${appKey}"]`
      : `script[src="${src}"]`
    if (document.querySelector(selector)) {
      resolve()
      return
    }
    const s = document.createElement('script')
    s.src = src
    s.async = true
    if (appKey) {
      s.setAttribute('data-app-key', appKey)
    }
    s.onload = () => resolve()
    s.onerror = () => reject(new Error(`Failed to load ${src}`))
    document.head.appendChild(s)
  })
}

export class DropboxPicker implements CloudPickerProvider {
  readonly provider = 'dropbox' as const
  private readonly appKey: string
  private readonly linkType: 'preview' | 'direct'

  constructor(appKey = '', linkType: 'preview' | 'direct' = 'preview') {
    this.appKey = appKey
    this.linkType = linkType
  }

  async pick(): Promise<PickedFile | null> {
    await loadScript('https://www.dropbox.com/static/api/2/dropins.js', this.appKey || undefined)
    if (!window.Dropbox) throw new Error('Dropbox SDK not loaded')

    const linkType = this.linkType
    return new Promise((resolve) => {
      window.Dropbox!.choose({
        success(files) {
          const f = files[0]
          if (!f) { resolve(null); return }
          resolve({
            provider: 'dropbox',
            externalId: f.id || f.link,
            name: f.name,
            viewUrl: f.link,
            downloadUrl: f.link,
            iconUrl: f.icon,
          })
        },
        cancel() { resolve(null) },
        linkType,
        multiselect: false,
        folderselect: false,
      })
    })
  }
}
