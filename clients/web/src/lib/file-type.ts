const IMAGE_MIME_TYPES = new Set([
  'image/jpeg',
  'image/png',
  'image/gif',
  'image/webp',
  'image/svg+xml',
])

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg'])
const PDF_EXTS = new Set(['.pdf'])

export type PreviewType = 'pdf' | 'image' | 'none'

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
  return 'none'
}
