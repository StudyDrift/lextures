export type FontFace = 'default' | 'open-dyslexic' | 'atkinson' | 'system'
export type LetterSpacing = 'normal' | 'wide' | 'wider'
export type WordSpacing = 'normal' | 'wide' | 'wider'
export type LineHeight = 'normal' | 'tall' | 'taller'
export type RulerColor = 'yellow' | 'grey'
export type UIMode = 'k2' | 'elementary' | 'standard'
/** UX.3 FR-9/10 — multiplies the type scale (1.0 default … 1.5). */
export type TextScale = 1 | 1.125 | 1.25 | 1.5

export interface ReadingPreferences {
  fontFace: FontFace
  letterSpacing: LetterSpacing
  wordSpacing: WordSpacing
  lineHeight: LineHeight
  /** UX.3 — multiplies all type role sizes via --lx-type-scale. */
  textScale: TextScale
  rulerEnabled: boolean
  rulerColor: RulerColor
  highContrastEnabled: boolean
  reducedMotionEnabled: boolean
  /** Admin-set override; null means derive from grade_level (plan 13.11). */
  uiModeOverride?: UIMode | null
  /** Effective UI mode derived server-side (grade_level or override). */
  effectiveUiMode?: UIMode
  updatedAt?: string
}

export const defaultReadingPreferences: ReadingPreferences = {
  fontFace: 'default',
  letterSpacing: 'normal',
  wordSpacing: 'normal',
  lineHeight: 'normal',
  textScale: 1,
  rulerEnabled: false,
  rulerColor: 'yellow',
  highContrastEnabled: false,
  reducedMotionEnabled: false,
}

export const TEXT_SCALE_OPTIONS: { value: TextScale; label: string }[] = [
  { value: 1, label: '100%' },
  { value: 1.125, label: '112%' },
  { value: 1.25, label: '125%' },
  { value: 1.5, label: '150%' },
]

export function normalizeTextScale(raw: unknown): TextScale {
  const n = typeof raw === 'number' ? raw : typeof raw === 'string' ? Number(raw) : NaN
  if (n === 1.125 || n === 1.25 || n === 1.5) return n
  return 1
}

const fontFamilyMap: Record<FontFace, string> = {
  'default':       "'Lextures', system-ui, sans-serif",
  'open-dyslexic': "'OpenDyslexic', sans-serif",
  'atkinson':      "'Atkinson Hyperlegible', system-ui, sans-serif",
  'system':        "system-ui, -apple-system, sans-serif",
}

const letterSpacingMap: Record<LetterSpacing, string> = {
  normal: 'normal',
  wide:   '0.12em',
  wider:  '0.35em',
}

const wordSpacingMap: Record<WordSpacing, string> = {
  normal: 'normal',
  wide:   '0.16em',
  wider:  '0.35em',
}

const lineHeightMap: Record<LineHeight, string> = {
  normal: '1.5',
  tall:   '1.8',
  taller: '2.0',
}

export function applyReadingPreferences(prefs: ReadingPreferences): void {
  const root = document.documentElement
  const scale = normalizeTextScale(prefs.textScale)
  root.style.setProperty('--reading-font-family',    fontFamilyMap[prefs.fontFace])
  root.style.setProperty('--reading-letter-spacing', letterSpacingMap[prefs.letterSpacing])
  root.style.setProperty('--reading-word-spacing',   wordSpacingMap[prefs.wordSpacing])
  root.style.setProperty('--reading-line-height',    lineHeightMap[prefs.lineHeight])
  root.style.setProperty('--reading-text-scale',     String(scale))
  root.style.setProperty('--lx-type-scale',          String(scale))
  root.classList.toggle('high-contrast', prefs.highContrastEnabled)
  root.classList.toggle('reduced-motion', prefs.reducedMotionEnabled)

  const mode = prefs.effectiveUiMode ?? 'standard'
  root.classList.toggle('ui-mode-k2', mode === 'k2')
  root.classList.toggle('ui-mode-elementary', mode === 'elementary')

  try {
    localStorage.setItem('lextures.highContrast', prefs.highContrastEnabled ? '1' : '0')
    localStorage.setItem('lextures.reduceMotion', prefs.reducedMotionEnabled ? '1' : '0')
    localStorage.setItem('lextures.uiMode', mode === 'standard' ? '' : mode)
    localStorage.setItem('lextures.textScale', String(scale))
  } catch { /* ignore */ }
}
