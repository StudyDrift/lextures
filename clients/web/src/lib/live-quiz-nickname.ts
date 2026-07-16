/** Client-side nickname rules mirrored from server ValidateNickname (IQ.4). */
const NICKNAME_RE = /^[\p{L}\p{N} _.\-'!]+$/u
const NICKNAME_MAX = 24

export function normalizeNickname(raw: string): string {
  return raw.trim()
}

export function validateNickname(raw: string): { ok: true; nickname: string } | { ok: false; reason: 'empty' | 'too_long' | 'charset' } {
  const nickname = normalizeNickname(raw)
  if (!nickname) return { ok: false, reason: 'empty' }
  if ([...nickname].length > NICKNAME_MAX) return { ok: false, reason: 'too_long' }
  if (!NICKNAME_RE.test(nickname)) return { ok: false, reason: 'charset' }
  return { ok: true, nickname }
}
