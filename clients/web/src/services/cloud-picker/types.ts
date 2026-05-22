export type CloudProvider = 'google_drive' | 'onedrive' | 'dropbox'

export type PickedFile = {
  provider: CloudProvider
  externalId: string
  name: string
  viewUrl: string
  iconUrl: string
  mimeType?: string
}

export interface CloudPickerProvider {
  provider: CloudProvider
  pick(): Promise<PickedFile | null>
}
