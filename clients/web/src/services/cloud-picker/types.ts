export type CloudProvider = 'google_drive' | 'onedrive' | 'dropbox'

export type PickedFile = {
  provider: CloudProvider
  externalId: string
  name: string
  viewUrl: string
  iconUrl: string
  mimeType?: string
  /** OAuth access token for downloading file bytes (Google Drive, OneDrive). */
  accessToken?: string
  /** Direct download URL when provided by the picker (OneDrive, Dropbox). */
  downloadUrl?: string
}

export interface CloudPickerProvider {
  provider: CloudProvider
  pick(): Promise<PickedFile | null>
}
