/** Parses JSON error bodies returned by the StudyDrift API. */
export function readApiErrorMessage(raw: unknown): string {
  if (raw && typeof raw === 'object' && 'type' in raw) {
    const t = (raw as { type?: unknown }).type
    if (t === 'password_policy_violation') {
      const d = (raw as { detail?: unknown }).detail
      if (typeof d === 'string' && d.trim()) return d
    }
  }
  if (raw && typeof raw === 'object' && 'error' in raw) {
    const err = (raw as { error?: { code?: string; message?: string } }).error
    if (err?.code === 'SEAT_LIMIT_REACHED') {
      return 'Your organization has reached its licensed seat limit. Contact your administrator to request additional seats.'
    }
    if (err?.message) return err.message
  }
  if (raw && typeof raw === 'object' && 'message' in raw) {
    const m = (raw as { message?: unknown }).message
    if (typeof m === 'string') return m
  }
  return 'Request failed'
}
