/** Web Speech API types and browser support helpers (plan 12.9). */

export type SpeechRecognitionCtor = new () => SpeechRecognitionInstance

export type SpeechRecognitionInstance = {
  continuous: boolean
  interimResults: boolean
  lang: string
  maxAlternatives: number
  onstart: (() => void) | null
  onend: (() => void) | null
  onerror: ((event: SpeechRecognitionErrorEvent) => void) | null
  onresult: ((event: SpeechRecognitionResultEvent) => void) | null
  start: () => void
  stop: () => void
  abort: () => void
}

export type SpeechRecognitionErrorEvent = {
  error: 'no-speech' | 'aborted' | 'audio-capture' | 'network' | 'not-allowed' | 'service-not-allowed' | string
  message?: string
}

export type SpeechRecognitionResultEvent = {
  resultIndex: number
  results: SpeechRecognitionResultList
}

export type SpeechRecognitionResultList = {
  length: number
  item: (index: number) => SpeechRecognitionResult
  [index: number]: SpeechRecognitionResult
}

export type SpeechRecognitionResult = {
  isFinal: boolean
  length: number
  item: (index: number) => SpeechRecognitionAlternative
  [index: number]: SpeechRecognitionAlternative
}

export type SpeechRecognitionAlternative = {
  transcript: string
  confidence: number
}

declare global {
  interface Window {
    SpeechRecognition?: SpeechRecognitionCtor
    webkitSpeechRecognition?: SpeechRecognitionCtor
  }
}

export function getSpeechRecognitionCtor(): SpeechRecognitionCtor | null {
  if (typeof window === 'undefined') return null
  return window.SpeechRecognition ?? window.webkitSpeechRecognition ?? null
}

export function isSpeechRecognitionSupported(): boolean {
  return getSpeechRecognitionCtor() != null
}

export function createSpeechRecognition(): SpeechRecognitionInstance | null {
  const Ctor = getSpeechRecognitionCtor()
  if (!Ctor) return null
  return new Ctor()
}
