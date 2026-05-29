import { useCallback, useEffect, useRef, useState } from 'react'
import { authorizedFetch } from '../api'
import { extractReadableContent, type ReadableSentence } from './extractReadableContent'
import type { TTSSpeed } from './speed-options'

export type TTSStatus = 'idle' | 'playing' | 'paused' | 'unavailable'

export type UseTTSOptions = {
  lang?: string
  speed: TTSSpeed
  voiceName: string | null
  onSpeedChange?: (speed: TTSSpeed) => void
}

const HIGHLIGHT_CLASS = 'tts-sentence-highlight'

function speechSynthesisSupported(): boolean {
  return typeof window !== 'undefined' && 'speechSynthesis' in window && window.speechSynthesis != null
}

/** Heuristic: screen reader user agents often include NVDA/JAWS/VoiceOver tokens. */
export function screenReaderLikelyActive(): boolean {
  if (typeof navigator === 'undefined') return false
  const ua = navigator.userAgent.toLowerCase()
  return /nvda|jaws|voiceover|talkback|orca|window-eyes/.test(ua)
}

function pickVoice(name: string | null, lang: string): SpeechSynthesisVoice | null {
  if (!speechSynthesisSupported()) return null
  const voices = window.speechSynthesis.getVoices()
  if (name) {
    const exact = voices.find((v) => v.name === name)
    if (exact) return exact
  }
  const langPrefix = lang.split('-')[0]?.toLowerCase()
  return (
    voices.find((v) => v.lang.toLowerCase().startsWith(lang)) ??
    voices.find((v) => v.lang.toLowerCase().startsWith(langPrefix ?? 'en')) ??
    voices[0] ??
    null
  )
}

function clearHighlights() {
  document.querySelectorAll(`.${HIGHLIGHT_CLASS}`).forEach((el) => {
    el.classList.remove(HIGHLIGHT_CLASS)
  })
}

function highlightSentence(sentence: ReadableSentence | undefined) {
  clearHighlights()
  if (!sentence) return
  sentence.element.classList.add(HIGHLIGHT_CLASS)
  sentence.element.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
}

async function playServerFallback(
  text: string,
  lang: string,
  speed: number,
  signal: AbortSignal,
): Promise<void> {
  const res = await authorizedFetch('/api/v1/tts/synthesize', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ text, lang, speed }),
    signal,
  })
  if (!res.ok) {
    throw new Error('Server TTS unavailable')
  }
  const blob = await res.blob()
  const url = URL.createObjectURL(blob)
  try {
    await new Promise<void>((resolve, reject) => {
      const audio = new Audio(url)
      const onAbort = () => {
        audio.pause()
        reject(new DOMException('Aborted', 'AbortError'))
      }
      signal.addEventListener('abort', onAbort, { once: true })
      audio.onended = () => {
        signal.removeEventListener('abort', onAbort)
        resolve()
      }
      audio.onerror = () => {
        signal.removeEventListener('abort', onAbort)
        reject(new Error('Audio playback failed'))
      }
      void audio.play().catch(reject)
    })
  } finally {
    URL.revokeObjectURL(url)
  }
}

export function useTTS(options: UseTTSOptions) {
  const { lang = 'en-US', speed, voiceName, onSpeedChange } = options
  const [status, setStatus] = useState<TTSStatus>('idle')
  const [sentenceIndex, setSentenceIndex] = useState(0)
  const [sentences, setSentences] = useState<ReadableSentence[]>([])
  const [voices, setVoices] = useState<SpeechSynthesisVoice[]>([])
  const [unavailableMessage, setUnavailableMessage] = useState<string | null>(null)

  const abortRef = useRef<AbortController | null>(null)
  const pausedRef = useRef(false)
  const indexRef = useRef(0)
  const sentencesRef = useRef<ReadableSentence[]>([])
  const speedRef = useRef(speed)
  const voiceRef = useRef(voiceName)

  useEffect(() => {
    speedRef.current = speed
  }, [speed])
  useEffect(() => {
    voiceRef.current = voiceName
  }, [voiceName])
  useEffect(() => {
    sentencesRef.current = sentences
  }, [sentences])

  useEffect(() => {
    if (!speechSynthesisSupported()) return
    const load = () => setVoices(window.speechSynthesis.getVoices())
    load()
    window.speechSynthesis.addEventListener('voiceschanged', load)
    return () => window.speechSynthesis.removeEventListener('voiceschanged', load)
  }, [])

  const stopInternal = useCallback(() => {
    abortRef.current?.abort()
    abortRef.current = null
    if (speechSynthesisSupported()) {
      window.speechSynthesis.cancel()
    }
    clearHighlights()
    pausedRef.current = false
    setStatus('idle')
    setSentenceIndex(0)
    indexRef.current = 0
  }, [])

  const speakSentenceWeb = useCallback(
    (sentence: ReadableSentence, rate: number, voice: SpeechSynthesisVoice | null): Promise<void> =>
      new Promise((resolve, reject) => {
        if (!speechSynthesisSupported()) {
          reject(new Error('Speech synthesis unavailable'))
          return
        }
        const utter = new SpeechSynthesisUtterance(sentence.text)
        utter.rate = rate
        utter.lang = lang
        if (voice) utter.voice = voice
        utter.onstart = () => highlightSentence(sentence)
        utter.onend = () => resolve()
        utter.onerror = (e) => {
          if (e.error === 'interrupted' || e.error === 'canceled') {
            resolve()
            return
          }
          reject(new Error(e.error))
        }
        window.speechSynthesis.speak(utter)
      }),
    [lang],
  )

  const runLoop = useCallback(async () => {
    const useWeb = speechSynthesisSupported()
    const controller = new AbortController()
    abortRef.current = controller

    try {
      for (let i = indexRef.current; i < sentencesRef.current.length; i++) {
        if (controller.signal.aborted || pausedRef.current) break
        indexRef.current = i
        setSentenceIndex(i)
        const sentence = sentencesRef.current[i]!
        highlightSentence(sentence)

        if (useWeb) {
          const voice = pickVoice(voiceRef.current, lang)
          await speakSentenceWeb(sentence, speedRef.current, voice)
        } else {
          await playServerFallback(sentence.text, lang, speedRef.current, controller.signal)
        }
      }
      if (!pausedRef.current && !controller.signal.aborted) {
        stopInternal()
      }
    } catch (err) {
      if (err instanceof DOMException && err.name === 'AbortError') {
        return
      }
      setUnavailableMessage('Read aloud is not available on this browser. Try Chrome or Edge.')
      setStatus('unavailable')
      clearHighlights()
    }
  }, [lang, speakSentenceWeb, stopInternal])

  const refreshSentences = useCallback(() => {
    const next = extractReadableContent(document)
    setSentences(next)
    sentencesRef.current = next
    return next
  }, [])

  const play = useCallback(() => {
    setUnavailableMessage(null)
    if (screenReaderLikelyActive()) {
      setUnavailableMessage(
        'Read aloud is paused because a screen reader may be active. You can still enable it from the toolbar.',
      )
    }
    const list = refreshSentences()
    if (list.length === 0) {
      setUnavailableMessage('No readable content found on this page.')
      setStatus('unavailable')
      return
    }
    pausedRef.current = false
    setStatus('playing')
    void runLoop()
  }, [refreshSentences, runLoop])

  const pause = useCallback(() => {
    pausedRef.current = true
    abortRef.current?.abort()
    if (speechSynthesisSupported()) {
      window.speechSynthesis.cancel()
    }
    setStatus('paused')
  }, [])

  const resume = useCallback(() => {
    if (sentencesRef.current.length === 0) {
      play()
      return
    }
    pausedRef.current = false
    setStatus('playing')
    void runLoop()
  }, [play, runLoop])

  const toggle = useCallback(() => {
    if (status === 'playing') {
      pause()
    } else if (status === 'paused') {
      resume()
    } else {
      indexRef.current = 0
      play()
    }
  }, [pause, play, resume, status])

  const restart = useCallback(() => {
    stopInternal()
    indexRef.current = 0
    play()
  }, [play, stopInternal])

  const setSpeed = useCallback(
    (next: TTSSpeed) => {
      speedRef.current = next
      onSpeedChange?.(next)
      if (status === 'playing') {
        pause()
        indexRef.current = sentenceIndex
        pausedRef.current = false
        setStatus('playing')
        void runLoop()
      }
    },
    [onSpeedChange, pause, runLoop, sentenceIndex, status],
  )

  const setVoice = useCallback((name: string | null) => {
    voiceRef.current = name
  }, [])

  useEffect(() => {
    return () => {
      stopInternal()
    }
  }, [stopInternal])

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.altKey && e.key.toLowerCase() === 'p') {
        e.preventDefault()
        toggle()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [toggle])

  useEffect(() => {
    const pauseOnInteraction = (e: Event) => {
      if (status !== 'playing') return
      const target = e.target
      if (!(target instanceof HTMLElement)) return
      if (target.closest('[data-read-aloud-controls]')) return
      if (e.type === 'keydown' && (e as KeyboardEvent).key === 'Tab') {
        pause()
      }
      if (e.type === 'click' && target.closest('a, button, input, select, textarea, [tabindex]')) {
        pause()
      }
    }
    document.addEventListener('keydown', pauseOnInteraction)
    document.addEventListener('click', pauseOnInteraction, true)
    return () => {
      document.removeEventListener('keydown', pauseOnInteraction)
      document.removeEventListener('click', pauseOnInteraction, true)
    }
  }, [pause, status])

  return {
    status,
    sentenceIndex,
    sentenceCount: sentences.length,
    voices,
    unavailableMessage,
    play,
    pause,
    resume,
    toggle,
    restart,
    stop: stopInternal,
    setSpeed,
    setVoice,
    refreshSentences,
  }
}
