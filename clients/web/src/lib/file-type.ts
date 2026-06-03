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
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
  'application/vnd.openxmlformats-officedocument.presentationml.presentation',
  'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet',
])

const IMAGE_EXTS = new Set(['.jpg', '.jpeg', '.png', '.gif', '.webp', '.svg'])
const PDF_EXTS = new Set(['.pdf'])
const AUDIO_EXTS = new Set(['.mp3', '.m4a', '.ogg', '.wav', '.flac', '.aac', '.opus'])
/** Open XML only; legacy .doc/.xls/.ppt are not server-previewable. */
const OFFICE_EXTS = new Set(['.docx', '.pptx', '.xlsx'])

const TEXT_MIME_TYPES = new Set([
  'text/plain',
  'text/markdown',
  'text/x-markdown',
])

const TEXT_EXTS = new Set(['.txt', '.md', '.markdown'])
const MARKDOWN_EXTS = new Set(['.md', '.markdown'])
const MARKDOWN_MIME_TYPES = new Set(['text/markdown', 'text/x-markdown'])

const CODE_EXTS = new Set([
  '.js', '.mjs', '.cjs', '.jsx',
  '.ts', '.mts', '.cts', '.tsx',
  '.cs', '.java', '.kt', '.kts',
  '.jl', '.sql', '.py', '.pyw',
  '.rb', '.go', '.rs', '.c', '.cpp', '.cxx', '.cc', '.h', '.hpp', '.hxx',
  '.sh', '.bash', '.zsh', '.fish',
  '.yaml', '.yml', '.json', '.jsonc',
  '.xml', '.html', '.htm', '.css', '.scss', '.less',
  '.php', '.swift', '.dart', '.r',
  '.scala', '.lua', '.pl', '.pm',
  '.hs', '.ex', '.exs', '.clj', '.groovy',
  '.ps1', '.psm1', '.tf', '.toml',
  '.graphql', '.gql', '.vim',
  '.proto', '.svelte', '.vue',
])

export type PreviewType = 'pdf' | 'image' | 'video' | 'audio' | 'office' | 'text' | 'code' | 'none'

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
  if (TEXT_MIME_TYPES.has(mt) || TEXT_EXTS.has(ext)) return 'text'
  if (CODE_EXTS.has(ext)) return 'code'
  return 'none'
}

/** True when the file should offer a rendered markdown preview tab. */
export function isMarkdownFilename(
  filename: string,
  mimeType?: string | null,
): boolean {
  const mt = (mimeType ?? '').toLowerCase().trim()
  if (MARKDOWN_MIME_TYPES.has(mt)) return true
  return MARKDOWN_EXTS.has(fileExt(filename))
}
