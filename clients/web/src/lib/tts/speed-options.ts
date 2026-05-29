export const TTS_SPEED_OPTIONS = [0.75, 1, 1.25, 1.5, 2] as const

export type TTSSpeed = (typeof TTS_SPEED_OPTIONS)[number]

export function formatTTSSpeedLabel(speed: number): string {
  if (speed === 1) return '1×'
  return `${speed}×`
}

export function normalizeTTSSpeed(value: number): TTSSpeed {
  const match = TTS_SPEED_OPTIONS.find((s) => Math.abs(s - value) < 0.001)
  return match ?? 1
}
