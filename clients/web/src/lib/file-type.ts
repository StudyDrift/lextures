const IMAGE_MIME_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'image/svg+xml',
])

const AUDIO_MIME_TYPES = new Set([
  'audio/mpeg',
  'audio/mp4',
  'audio/ogg',
  'audio/wav',
  'audio/webm',
  'audio/aac',
  'audio/flac',
  'audio/x-flac',
  'audio/opus',
])

const OFFICE_MIME_TYPES = new Set([
  'application/msword',
  'application/vnd.ms-powerpoint',
  'application/vnd.ms-excel',
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
])

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg'])
const PDF_EXTS = new Set(['.pdf'])
const AUDIO_EXTS = new Set(['.mp3', '.m4a', '.ogg', '.wav', '.flac', '.aac', '.opus'])
const OFFICE_EXTS = new Set(['.doc', '.docx', '.ppt', '.pptx', '.xls', '.xlsx'])

export type PreviewType = 'pdf' | 'image' | 'video' | 'audio' | 'office' | 'none'

function fileExt(filename: string): string {
  const i = filename.lastIndexOf('.')
  return i >= 0 ? filename.slice(i).toLowerCase() : ''
}

/**
 * Determine preview type from MIME type and/or filename extension.
 * MIME type is primary; extension is fallback for unknown or missing MIME types.
 */
export function detectPreviewType(
  mimeType: string | null | undefined,
  filename: string | null | undefined,
): PreviewType {
  const mt = (mimeType ?? '').toLowerCase().trim()
  const ext = filename ? fileExt(filename) : ''
  if (mt === 'application/pdf' || PDF_EXTS.has(ext)) return 'pdf'
  if (IMAGE_MIME_TYPES.has(mt) || IMAGE_EXTS.has(ext)) return 'image'
  if (mt.startsWith('video/') || ext === '.mp4' || ext === '.webm' || ext === '.mov') return 'video'
  if (AUDIO_MIME_TYPES.has(mt) || mt.startsWith('audio/') || AUDIO_EXTS.has(ext)) return 'audio'
  if (OFFICE_MIME_TYPES.has(mt) || OFFICE_EXTS.has(ext)) return 'office'
  return 'none'
}
