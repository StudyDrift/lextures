/**
 * Notification sounds (#775): an optional, device-local chime when new inbox mail arrives.
 * Off by default; kids can turn it on from their simplified Settings.
 */
const SOUNDS_KEY = 'studydrift_notification_sounds'

let memoryEnabled = false

export function notificationSoundsEnabled(): boolean {
  try {
    return localStorage.getItem(SOUNDS_KEY) === '1'
  } catch {
    return memoryEnabled
  }
}

export function setNotificationSoundsEnabled(enabled: boolean): void {
  memoryEnabled = enabled
  try {
    if (enabled) localStorage.setItem(SOUNDS_KEY, '1')
    else localStorage.removeItem(SOUNDS_KEY)
  } catch {
    /* private mode: keep the in-memory value */
  }
}

type AudioContextCtor = new () => AudioContext

function audioContextCtor(): AudioContextCtor | null {
  if (typeof window === 'undefined') return null
  const w = window as unknown as { AudioContext?: AudioContextCtor; webkitAudioContext?: AudioContextCtor }
  return w.AudioContext ?? w.webkitAudioContext ?? null
}

/** Plays a short, soft two-note chime. Silently does nothing when audio is unavailable. */
export function playNotificationChime(): void {
  const Ctor = audioContextCtor()
  if (!Ctor) return
  try {
    const ctx = new Ctor()
    const now = ctx.currentTime
    const notes = [880, 1320]
    notes.forEach((freq, i) => {
      const osc = ctx.createOscillator()
      const gain = ctx.createGain()
      osc.type = 'sine'
      osc.frequency.value = freq
      const start = now + i * 0.12
      gain.gain.setValueAtTime(0.0001, start)
      gain.gain.exponentialRampToValueAtTime(0.15, start + 0.02)
      gain.gain.exponentialRampToValueAtTime(0.0001, start + 0.25)
      osc.connect(gain)
      gain.connect(ctx.destination)
      osc.start(start)
      osc.stop(start + 0.3)
    })
    window.setTimeout(() => void ctx.close().catch(() => undefined), 800)
  } catch {
    /* autoplay policy or device without audio output */
  }
}

/** Chime when the unread count goes up (not on the first load). */
export function shouldChimeForUnread(previous: number | null, next: number): boolean {
  return previous !== null && next > previous && notificationSoundsEnabled()
}
