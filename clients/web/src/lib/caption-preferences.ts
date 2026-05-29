import type { CSSProperties } from 'react'

/** Session-persisted caption display preferences (plan 12.4). */

export type CaptionFontSize = 'small' | 'medium' | 'large'
export type CaptionPosition = 'top' | 'bottom'

export type CaptionColorPreset = 'default' | 'high-contrast' | 'yellow-on-black'

export type CaptionPreferences = {
  enabled: boolean
  fontSize: CaptionFontSize
  colorPreset: CaptionColorPreset
  position: CaptionPosition
}

const STORAGE_KEY = 'lextures_caption_prefs'

const defaults: CaptionPreferences = {
  enabled: true,
  fontSize: 'medium',
  colorPreset: 'default',
  position: 'bottom',
}

export function loadCaptionPreferences(): CaptionPreferences {
  try {
    const raw = sessionStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaults }
    const parsed = JSON.parse(raw) as Partial<CaptionPreferences>
    return { ...defaults, ...parsed }
  } catch {
    return { ...defaults }
  }
}

export function saveCaptionPreferences(prefs: CaptionPreferences): void {
  sessionStorage.setItem(STORAGE_KEY, JSON.stringify(prefs))
}

export function captionStyleVars(prefs: CaptionPreferences): CSSProperties {
  const size =
    prefs.fontSize === 'small' ? '0.85rem' : prefs.fontSize === 'large' ? '1.35rem' : '1.05rem'
  let color = '#fff'
  let bg = 'rgba(0,0,0,0.75)'
  if (prefs.colorPreset === 'high-contrast') {
    color = '#fff'
    bg = '#000'
  } else if (prefs.colorPreset === 'yellow-on-black') {
    color = '#ffff00'
    bg = '#000'
  }
  return {
    ['--caption-font-size' as string]: size,
    ['--caption-color' as string]: color,
    ['--caption-bg' as string]: bg,
    ['--caption-position' as string]: prefs.position === 'top' ? 'flex-start' : 'flex-end',
  }
}
