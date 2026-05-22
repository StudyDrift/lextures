import type { CloudPickerProvider, PickedFile } from './types'

declare global {
  interface Window {
    gapi?: {
      load(lib: string, cb: () => void): void
      auth2?: {
        init(cfg: { client_id: string; scope: string }): Promise<unknown>
        getAuthInstance(): { signIn(): Promise<{ getAuthResponse(): { access_token: string } }> }
      }
      client?: { init(cfg: { apiKey: string; discoveryDocs: string[] }): Promise<void> }
    }
    google?: {
      picker?: {
        PickerBuilder: new () => GooglePickerBuilder
        Action: { PICKED: string; CANCEL: string }
        ViewId: { DOCS: string }
        Feature: { MULTISELECT_ENABLED: string }
      }
    }
  }
}

interface GooglePickerBuilder {
  addView(viewId: string): GooglePickerBuilder
  setOAuthToken(token: string): GooglePickerBuilder
  setDeveloperKey(key: string): GooglePickerBuilder
  setCallback(cb: (data: GooglePickerResult) => void): GooglePickerBuilder
  build(): { setVisible(v: boolean): void }
}

interface GooglePickerResult {
  action: string
  docs?: Array<{
    id: string
    name: string
    url: string
    iconUrl: string
    mimeType: string
  }>
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

export class GoogleDrivePicker implements CloudPickerProvider {
  readonly provider = 'google_drive' as const
  private readonly clientId: string
  private readonly apiKey: string

  constructor(clientId: string, apiKey: string) {
    this.clientId = clientId
    this.apiKey = apiKey
  }

  async pick(): Promise<PickedFile | null> {
    await loadScript('https://apis.google.com/js/api.js')

    const gapi = window.gapi
    if (!gapi) throw new Error('Google API not loaded')

    await new Promise<void>((resolve) => gapi.load('auth2:client:picker', resolve))
    await gapi.client!.init({ apiKey: this.apiKey, discoveryDocs: [] })
    await gapi.auth2!.init({ client_id: this.clientId, scope: 'https://www.googleapis.com/auth/drive.file' })

    const authInstance = gapi.auth2!.getAuthInstance()
    const user = await authInstance.signIn()
    const token = user.getAuthResponse().access_token

    return new Promise((resolve) => {
      const pickerBuilder = new window.google!.picker!.PickerBuilder()
      const picker = pickerBuilder
        .addView(window.google!.picker!.ViewId.DOCS)
        .setOAuthToken(token)
        .setDeveloperKey(this.apiKey)
        .setCallback((data: GooglePickerResult) => {
          if (data.action === window.google!.picker!.Action.PICKED && data.docs?.[0]) {
            const doc = data.docs[0]
            resolve({
              provider: 'google_drive',
              externalId: doc.id,
              name: doc.name,
              viewUrl: doc.url,
              iconUrl: doc.iconUrl,
              mimeType: doc.mimeType,
            })
          } else if (data.action === window.google!.picker!.Action.CANCEL) {
            resolve(null)
          }
        })
        .build()
      picker.setVisible(true)
    })
  }
}
