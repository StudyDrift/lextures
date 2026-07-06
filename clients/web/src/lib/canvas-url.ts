/** Canvas profile settings page where users create API access tokens. */
export function canvasAccessTokenSettingsUrl(raw: string): string | null {
  const trimmed = raw.trim()
  if (!trimmed) return null

  const candidate = /^https?:\/\//i.test(trimmed) ? trimmed : `https://${trimmed}`

  try {
    const url = new URL(candidate)
    if (!url.hostname) return null
    return `${url.origin}/profile/settings`
  } catch {
    return null
  }
}