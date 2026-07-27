import { beforeEach, describe, expect, it } from 'vitest'
import {
  loadMediaPlaybackPreferences,
  saveMediaPlaybackPreferences,
  supportedPlaybackRates,
} from '../media-playback-preferences'

describe('media-playback-preferences', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('loads defaults and persists', () => {
    expect(loadMediaPlaybackPreferences().playbackRate).toBe(1)
    saveMediaPlaybackPreferences({
      playbackRate: 1.5,
      volume: 0.4,
      muted: false,
      captionsEnabled: false,
    })
    const next = loadMediaPlaybackPreferences()
    expect(next.playbackRate).toBe(1.5)
    expect(next.volume).toBe(0.4)
    expect(next.captionsEnabled).toBe(false)
  })

  it('exposes supported rates', () => {
    expect(supportedPlaybackRates()).toContain(1)
    expect(supportedPlaybackRates()).toContain(1.5)
  })
})
