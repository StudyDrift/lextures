import type { CloudPickerProvider, PickedFile } from './types'

declare global {
  interface Window {
    OneDrive?: {
      open(options: OneDriveOptions): void
    }
  }
}

interface OneDriveOptions {
  clientId: string
  action: string
  multiSelect: boolean
  openInNewWindow: boolean
  advanced: { redirectUri: string }
  success(result: { value: OneDriveItem[] }): void
  cancel(): void
  error(e: unknown): void
}

interface OneDriveItem {
  id: string
  name: string
  webUrl: string
  thumbnails?: Array<{ large?: { url: string } }>
}

function loadScript(src: string): Promise<void> {
  return new Promise((resolve, reject) => {
    if (document.querySelector(`script[src="${src}"]`)) {
      resolve()
      return
    }
    const s = document.createElement('script')
    s.src = src
    s.onload = () => resolve()
    s.onerror = () => reject(new Error(`Failed to load ${src}`))
    document.head.appendChild(s)
  })
}

export class OneDrivePicker implements CloudPickerProvider {
  readonly provider = 'onedrive' as const

  constructor(private readonly clientId: string) {}

  async pick(): Promise<PickedFile | null> {
    await loadScript('https://js.live.net/v7.2/OneDrive.js')
    if (!window.OneDrive) throw new Error('OneDrive SDK not loaded')

    return new Promise((resolve, reject) => {
      window.OneDrive!.open({
        clientId: this.clientId,
        action: 'share',
        multiSelect: false,
        openInNewWindow: true,
        advanced: { redirectUri: window.location.origin },
        success(result) {
          const item = result.value[0]
          if (!item) { resolve(null); return }
          resolve({
            provider: 'onedrive',
            externalId: item.id,
            name: item.name,
            viewUrl: item.webUrl,
            iconUrl: item.thumbnails?.[0]?.large?.url ?? 'https://res-1.cdn.office.net/files/fabric-cdn-prod_20221209.001/assets/item-types/16/docx.svg',
          })
        },
        cancel() { resolve(null) },
        error(e) { reject(e instanceof Error ? e : new Error(String(e))) },
      })
    })
  }
}
