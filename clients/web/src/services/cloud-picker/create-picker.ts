import type { ConfiguredCloudProvider, CloudProviderId } from '../../lib/cloud-providers-api'
import { DropboxPicker } from './dropbox'
import { GoogleDrivePicker } from './google-drive'
import { OneDrivePicker } from './onedrive'
import type { CloudPickerProvider } from './types'

export function createCloudPicker(
  provider: CloudProviderId,
  config: ConfiguredCloudProvider,
  mode: 'link' | 'import' = 'link',
): CloudPickerProvider {
  switch (provider) {
    case 'google_drive':
      return new GoogleDrivePicker(config.clientId ?? '', config.apiKey ?? '')
    case 'onedrive':
      return new OneDrivePicker(config.clientId ?? '', mode === 'import' ? 'download' : 'share')
    case 'dropbox':
      return new DropboxPicker(config.appKey ?? '', mode === 'import' ? 'direct' : 'preview')
    default: {
      const _exhaustive: never = provider
      throw new Error(`Unknown cloud provider: ${_exhaustive}`)
    }
  }
}
