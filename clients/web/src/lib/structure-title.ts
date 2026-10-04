/** True when a structure title is a raw API error envelope, not a name someone chose. */
export function isApiErrorPayloadTitle(title: string): boolean {
  const trimmed = title.trim()
  if (!trimmed.startsWith('{')) return false
  try {
    const parsed = JSON.parse(trimmed) as { error?: { code?: unknown; message?: unknown } }
    const code = parsed?.error?.code
    const message = parsed?.error?.message
    return typeof code === 'string' && code.trim() !== '' && typeof message === 'string' && message.trim() !== ''
  } catch {
    return false
  }
}
