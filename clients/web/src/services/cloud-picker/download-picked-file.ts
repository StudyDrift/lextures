import type { PickedFile } from './types'

const GOOGLE_APPS_EXPORT: Record<string, { mimeType: string; extension: string }> = {
  'application/vnd.google-apps.document': {
    mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    extension: '.docx',
  },
  'application/vnd.google-apps.spreadsheet': {
    mimeType: 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
    extension: '.xlsx',
  },
  'application/vnd.google-apps.presentation': {
    mimeType: 'application/vnd.openxmlformats-officedocument.presentationml.presentation',
    extension: '.pptx',
  },
  'application/vnd.google-apps.drawing': {
    mimeType: 'image/png',
    extension: '.png',
  },
}

function ensureExtension(name: string, extension: string): string {
  if (name.toLowerCase().endsWith(extension)) return name
  return `${name}${extension}`
}

function filenameFromResponse(res: Response, fallback: string): string {
  const cd = res.headers.get('content-disposition')
  if (!cd) return fallback
  const match = /filename\*?=(?:UTF-8''|")?([^";]+)/i.exec(cd)
  if (!match?.[1]) return fallback
  try {
    return decodeURIComponent(match[1].replace(/"/g, ''))
  } catch {
    return match[1]
  }
}

async function downloadGoogleDriveFile(file: PickedFile): Promise<File> {
  if (!file.accessToken) {
    throw new Error('Google Drive import requires an access token.')
  }
  const googleApp = file.mimeType?.startsWith('application/vnd.google-apps.')
  let url: string
  let filename = file.name
  let mimeType = file.mimeType ?? 'application/octet-stream'

  if (googleApp && file.mimeType) {
    const exportSpec = GOOGLE_APPS_EXPORT[file.mimeType] ?? {
      mimeType: 'application/pdf',
      extension: '.pdf',
    }
    url = `https://www.googleapis.com/drive/v3/files/${encodeURIComponent(file.externalId)}/export?mimeType=${encodeURIComponent(exportSpec.mimeType)}`
    filename = ensureExtension(filename, exportSpec.extension)
    mimeType = exportSpec.mimeType
  } else {
    url = `https://www.googleapis.com/drive/v3/files/${encodeURIComponent(file.externalId)}?alt=media`
  }

  const res = await fetch(url, {
    headers: { Authorization: `Bearer ${file.accessToken}` },
  })
  if (!res.ok) {
    throw new Error(`Could not download from Google Drive (HTTP ${res.status}).`)
  }
  const blob = await res.blob()
  return new File([blob], filenameFromResponse(res, filename), {
    type: blob.type || mimeType,
  })
}

async function downloadOneDriveFile(file: PickedFile): Promise<File> {
  const url = file.downloadUrl ?? file.viewUrl
  if (!url) throw new Error('OneDrive import requires a download URL.')
  const res = await fetch(url)
  if (!res.ok) {
    throw new Error(`Could not download from OneDrive (HTTP ${res.status}).`)
  }
  const blob = await res.blob()
  return new File([blob], filenameFromResponse(res, file.name), {
    type: blob.type || 'application/octet-stream',
  })
}

async function downloadDropboxFile(file: PickedFile): Promise<File> {
  const url = file.downloadUrl ?? file.viewUrl
  if (!url) throw new Error('Dropbox import requires a download URL.')
  const res = await fetch(url.replace('?dl=0', '?dl=1'))
  if (!res.ok) {
    throw new Error(`Could not download from Dropbox (HTTP ${res.status}).`)
  }
  const blob = await res.blob()
  return new File([blob], filenameFromResponse(res, file.name), {
    type: blob.type || 'application/octet-stream',
  })
}

export async function downloadPickedFile(file: PickedFile): Promise<File> {
  switch (file.provider) {
    case 'google_drive':
      return downloadGoogleDriveFile(file)
    case 'onedrive':
      return downloadOneDriveFile(file)
    case 'dropbox':
      return downloadDropboxFile(file)
    default: {
      const _exhaustive: never = file.provider
      throw new Error(`Unknown cloud provider: ${_exhaustive}`)
    }
  }
}
