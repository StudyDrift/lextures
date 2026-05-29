import { describe, expect, it, beforeEach } from 'vitest'
import {
  captionStyleVars,
  loadCaptionPreferences,
  saveCaptionPreferences,
} from '../caption-preferences'

describe('caption-preferences', () => {
  beforeEach(() => {
    sessionStorage.clear()
  })

  it('persists preferences in sessionStorage', () => {
    saveCaptionPreferences({
      enabled: false,
      fontSize: 'large',
      colorPreset: 'high-contrast',
      position: 'top',
    })
    const loaded = loadCaptionPreferences()
    expect(loaded.enabled).toBe(false)
    expect(loaded.fontSize).toBe('large')
    expect(loaded.colorPreset).toBe('high-contrast')
    expect(loaded.position).toBe('top')
  })

  it('high-contrast preset uses 7:1-friendly colors', () => {
    const vars = captionStyleVars({
      enabled: true,
      fontSize: 'medium',
      colorPreset: 'high-contrast',
      position: 'bottom',
    })
    const custom = vars as Record<string, string>
    expect(custom['--caption-color']).toBe('#fff')
    expect(custom['--caption-bg']).toBe('#000')
  })
})
