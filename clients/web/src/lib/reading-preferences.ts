export type FontFace = 'default' | 'open-dyslexic' | 'atkinson' | 'system'
export type LetterSpacing = 'normal' | 'wide' | 'wider'
export type WordSpacing = 'normal' | 'wide' | 'wider'
export type LineHeight = 'normal' | 'tall' | 'taller'
export type RulerColor = 'yellow' | 'grey'

export interface ReadingPreferences {
  fontFace: FontFace
  letterSpacing: LetterSpacing
  wordSpacing: WordSpacing
  lineHeight: LineHeight
  rulerEnabled: boolean
  rulerColor: RulerColor
  updatedAt?: string
}

export const defaultReadingPreferences: ReadingPreferences = {
  fontFace: 'default',
  letterSpacing: 'normal',
  wordSpacing: 'normal',
  lineHeight: 'normal',
  rulerEnabled: false,
  rulerColor: 'yellow',
}

const fontFamilyMap: Record<FontFace, string> = {
  'default':       "'Plus Jakarta Sans', system-ui, sans-serif",
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
  root.style.setProperty('--reading-font-family',    fontFamilyMap[prefs.fontFace])
  root.style.setProperty('--reading-letter-spacing', letterSpacingMap[prefs.letterSpacing])
  root.style.setProperty('--reading-word-spacing',   wordSpacingMap[prefs.wordSpacing])
  root.style.setProperty('--reading-line-height',    lineHeightMap[prefs.lineHeight])
}

export function clearReadingPreferences(): void {
  const root = document.documentElement
  root.style.removeProperty('--reading-font-family')
  root.style.removeProperty('--reading-letter-spacing')
  root.style.removeProperty('--reading-word-spacing')
  root.style.removeProperty('--reading-line-height')
}
