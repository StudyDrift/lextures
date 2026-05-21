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

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) {
      resolve()
      return
    }
    const s = document.createElement('script')
    s.src = src
    s.async = true
    s.onload = () => resolve()
    s.onerror = () => reject(new Error(`Failed to load ${src}`))
    document.head.appendChild(s)
  })
}

export class DropboxPicker implements CloudPickerProvider {
  readonly provider = 'dropbox' as const

  async pick(): Promise<PickedFile | null> {
    await loadScript('https://www.dropbox.com/static/api/2/dropins.js')
    if (!window.Dropbox) throw new Error('Dropbox SDK not loaded')

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
            iconUrl: f.icon,
          })
        },
        cancel() { resolve(null) },
        linkType: 'preview',
        multiselect: false,
        folderselect: false,
      })
    })
  }
}
