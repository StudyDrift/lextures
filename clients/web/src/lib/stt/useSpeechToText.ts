import { useCallback, useEffect, useRef, useState } from 'react'
import {
  createSpeechRecognition,
  isSpeechRecognitionSupported,
  type SpeechRecognitionInstance,
} from './speech-recognition'

export type SttMode = 'continuous' | 'single'
export type SttStatus = 'idle' | 'listening' | 'processing' | 'stopped' | 'error'

export type UseSpeechToTextOptions = {
  language?: string
  mode?: SttMode
  enabled?: boolean
  onInterimResult?: (text: string) => void
  onFinalResult?: (text: string) => void
  onError?: (message: string) => void
}

export type UseSpeechToTextReturn = {
  supported: boolean
  status: SttStatus
  statusLabel: string
  liveAnnouncement: string
  interimText: string
  errorMessage: string | null
  isListening: boolean
  permissionDenied: boolean
  start: () => Promise<void>
  stop: () => void
  toggle: () => Promise<void>
}

function statusToLabel(status: SttStatus): string {
  switch (status) {
    case 'listening':
      return 'Listening…'
    case 'processing':
      return 'Processing…'
    case 'stopped':
      return 'Stopped'
    case 'error':
      return 'Error'
    case 'idle':
    default:
      return 'Stopped'
  }
}

function statusToAnnouncement(status: SttStatus, extra?: string): string {
  switch (status) {
    case 'listening':
      return 'Dictation started. Listening.'
    case 'processing':
      return 'Processing dictation.'
    case 'stopped':
      return 'Dictation stopped.'
    case 'error':
      return extra ?? 'Dictation error.'
    case 'idle':
    default:
      return ''
  }
}

export function useSpeechToText(options: UseSpeechToTextOptions = {}): UseSpeechToTextReturn {
  const {
    language = 'en-US',
    mode = 'continuous',
    enabled = true,
    onInterimResult,
    onFinalResult,
    onError,
  } = options

  const supported = isSpeechRecognitionSupported()
  const [status, setStatus] = useState<SttStatus>('idle')
  const [interimText, setInterimText] = useState('')
  const [errorMessage, setErrorMessage] = useState<string | null>(null)
  const [permissionDenied, setPermissionDenied] = useState(false)
  const [liveAnnouncement, setLiveAnnouncement] = useState('')
  const recognitionRef = useRef<SpeechRecognitionInstance | null>(null)
  const permissionRequestedRef = useRef(false)

  const stop = useCallback(() => {
    recognitionRef.current?.stop()
    recognitionRef.current = null
    setInterimText('')
    setStatus('stopped')
    setLiveAnnouncement(statusToAnnouncement('stopped'))
  }, [])

  useEffect(() => {
    return () => {
      recognitionRef.current?.abort()
      recognitionRef.current = null
    }
  }, [])

  const start = useCallback(async () => {
    if (!enabled || !supported) return
    setErrorMessage(null)
    setPermissionDenied(false)

    if (!permissionRequestedRef.current && navigator.mediaDevices?.getUserMedia) {
      permissionRequestedRef.current = true
      try {
        const stream = await navigator.mediaDevices.getUserMedia({ audio: true })
        for (const track of stream.getTracks()) track.stop()
      } catch {
        const msg = 'Microphone permission denied. Check your browser settings.'
        setPermissionDenied(true)
        setErrorMessage(msg)
        setStatus('error')
        setLiveAnnouncement(msg)
        onError?.(msg)
        return
      }
    }

    const recognition = createSpeechRecognition()
    if (!recognition) return

    recognition.continuous = mode === 'continuous'
    recognition.interimResults = true
    recognition.lang = language
    recognition.maxAlternatives = 1

    recognition.onstart = () => {
      setStatus('listening')
      setLiveAnnouncement(statusToAnnouncement('listening'))
    }

    recognition.onresult = (event) => {
      let interim = ''
      let finalChunk = ''
      for (let i = event.resultIndex; i < event.results.length; i++) {
        const result = event.results[i]
        const transcript = result[0]?.transcript ?? ''
        if (result.isFinal) {
          finalChunk += transcript
        } else {
          interim += transcript
        }
      }
      if (interim) {
        setInterimText(interim)
        onInterimResult?.(interim)
      }
      if (finalChunk.trim()) {
        setInterimText('')
        setStatus('processing')
        setLiveAnnouncement(statusToAnnouncement('processing'))
        onFinalResult?.(finalChunk.trim())
        setStatus('listening')
        setLiveAnnouncement(statusToAnnouncement('listening'))
      }
    }

    recognition.onerror = (event) => {
      if (event.error === 'aborted') return
      let msg = 'Dictation stopped.'
      if (event.error === 'not-allowed' || event.error === 'service-not-allowed') {
        msg = 'Microphone permission denied. Check your browser settings.'
        setPermissionDenied(true)
      } else if (event.error === 'network') {
        msg = 'Dictation stopped: connection lost'
      } else if (event.error === 'no-speech') {
        msg = 'No speech detected.'
      }
      setErrorMessage(msg)
      setStatus('error')
      setLiveAnnouncement(msg)
      onError?.(msg)
      recognitionRef.current = null
    }

    recognition.onend = () => {
      if (recognitionRef.current === recognition) {
        recognitionRef.current = null
        setInterimText('')
        setStatus((prev) => {
          if (prev === 'error') return prev
          setLiveAnnouncement(statusToAnnouncement('stopped'))
          return 'stopped'
        })
      }
    }

    recognitionRef.current = recognition
    try {
      recognition.start()
    } catch {
      const msg = 'Could not start dictation.'
      setErrorMessage(msg)
      setStatus('error')
      setLiveAnnouncement(msg)
      onError?.(msg)
    }
  }, [enabled, supported, language, mode, onError, onFinalResult, onInterimResult])

  const toggle = useCallback(async () => {
    if (status === 'listening' || status === 'processing') {
      stop()
    } else {
      await start()
    }
  }, [start, status, stop])

  return {
    supported,
    status,
    statusLabel: statusToLabel(status),
    liveAnnouncement,
    interimText,
    errorMessage,
    isListening: status === 'listening' || status === 'processing',
    permissionDenied,
    start,
    stop,
    toggle,
  }
}
