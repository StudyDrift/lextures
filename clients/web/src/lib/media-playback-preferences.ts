/** Learner playback preferences for media tools (CT.19 FR-10). */

export type MediaPlaybackPreferences = {
  playbackRate: number
  volume: number
  muted: boolean
  captionsEnabled: boolean
}

const STORAGE_KEY = 'lextures_media_playback_prefs'
const RATES = [0.5, 0.75, 1, 1.25, 1.5, 1.75, 2] as const

const defaults: MediaPlaybackPreferences = {
  playbackRate: 1,
  volume: 1,
  muted: false,
  captionsEnabled: true,
}

export function supportedPlaybackRates(): readonly number[] {
  return RATES
}

export function loadMediaPlaybackPreferences(): MediaPlaybackPreferences {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return { ...defaults }
    const parsed = JSON.parse(raw) as Partial<MediaPlaybackPreferences>
    const rate = typeof parsed.playbackRate === 'number' ? parsed.playbackRate : defaults.playbackRate
    const volume = typeof parsed.volume === 'number' ? parsed.volume : defaults.volume
    return {
      playbackRate: RATES.includes(rate as (typeof RATES)[number]) ? rate : 1,
      volume: Math.min(1, Math.max(0, volume)),
      muted: parsed.muted === true,
      captionsEnabled: parsed.captionsEnabled !== false,
    }
  } catch {
    return { ...defaults }
  }
}

export function saveMediaPlaybackPreferences(prefs: MediaPlaybackPreferences): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(prefs))
  } catch {
    // Ignore quota / private-mode failures.
  }
}
