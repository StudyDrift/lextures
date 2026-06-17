export type ClientPlatform = 'windows' | 'macos' | 'linux' | 'other'

function detectClientPlatform(): ClientPlatform {
  if (typeof navigator === 'undefined') return 'other'

  const uaData = (navigator as Navigator & { userAgentData?: { platform: string } }).userAgentData
  const platform = uaData?.platform ?? navigator.platform ?? ''
  const ua = navigator.userAgent

  if (/Win/i.test(platform) || /Windows/i.test(ua)) return 'windows'
  if (/Mac/i.test(platform) || /Macintosh|Mac OS X/i.test(ua)) return 'macos'
  if (/Linux/i.test(platform) || /Linux|CrOS/i.test(ua)) return 'linux'
  return 'other'
}

/** Sets `data-platform` on `<html>` for platform-scoped global CSS (e.g. Windows scrollbars). */
export function applyPlatformToDocument(): void {
  if (typeof document === 'undefined') return
  document.documentElement.dataset.platform = detectClientPlatform()
}