import { Mic, MicOff } from 'lucide-react'
import type { KeyboardEvent, MouseEvent } from 'react'
import { useEffect, useId, useRef } from 'react'
import { LiveRegion } from '../../a11y/live-region'
import { useSpeechToText, type SttMode } from '../../../lib/stt/useSpeechToText'

export type DictationButtonProps = {
  disabled?: boolean
  language?: string
  mode?: SttMode
  /** When set, shown as the button tooltip (e.g. accommodation message). */
  accommodationTooltip?: string
  onInterimResult?: (text: string) => void
  onFinalResult?: (text: string) => void
}

/**
 * Microphone dictation control for block editor toolbars and quiz text fields.
 * Hidden when the browser lacks SpeechRecognition (FR-6).
 */
export function DictationButton({
  disabled,
  language = 'en-US',
  mode = 'continuous',
  accommodationTooltip,
  onInterimResult,
  onFinalResult,
}: DictationButtonProps) {
  const errorRef = useRef<HTMLParagraphElement>(null)
  const errorId = useId()
  const {
    supported,
    statusLabel,
    liveAnnouncement,
    interimText,
    errorMessage,
    isListening,
    permissionDenied,
    toggle,
  } = useSpeechToText({
    language,
    mode,
    enabled: !disabled,
    onInterimResult,
    onFinalResult,
  })

  useEffect(() => {
    if (permissionDenied && errorRef.current) {
      errorRef.current.focus()
    }
  }, [permissionDenied])

  if (!supported) {
    return null
  }

  function preventBlur(e: MouseEvent) {
    e.preventDefault()
  }

  function onKeyDown(e: KeyboardEvent<HTMLButtonElement>) {
    if (e.key === 'Enter' || e.key === ' ') {
      e.preventDefault()
      void toggle()
    }
  }

  const label = isListening ? 'Stop dictation' : 'Start dictation'
  const title = accommodationTooltip ?? label

  return (
    <>
      <span className="mx-0.5 h-5 w-px shrink-0 bg-slate-200 dark:bg-neutral-600" aria-hidden />
      <button
        type="button"
        disabled={disabled}
        role="button"
        aria-pressed={isListening}
        aria-label={label}
        aria-describedby={errorMessage ? errorId : undefined}
        title={title}
        onMouseDown={preventBlur}
        onClick={() => void toggle()}
        onKeyDown={onKeyDown}
        className={`relative flex h-7 w-7 shrink-0 items-center justify-center rounded text-slate-600 hover:bg-slate-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-neutral-300 dark:hover:bg-neutral-700 ${
          isListening ? 'text-red-600 dark:text-red-400' : ''
        }`}
      >
        {isListening ? (
          <>
            <span
              className="absolute inset-0 rounded motion-safe:animate-ping bg-red-400/30 motion-reduce:animate-none"
              aria-hidden
            />
            <MicOff className="relative h-4 w-4" aria-hidden />
          </>
        ) : (
          <Mic className="h-4 w-4" aria-hidden />
        )}
      </button>
      <span className="sr-only" aria-live="polite" aria-atomic="true">
        {statusLabel}
      </span>
      <LiveRegion>{liveAnnouncement}</LiveRegion>
      {interimText ? (
        <span className="sr-only" aria-live="polite">
          {interimText}
        </span>
      ) : null}
      {errorMessage ? (
        <p
          id={errorId}
          ref={errorRef}
          tabIndex={-1}
          role="alert"
          className="sr-only"
        >
          {errorMessage}
        </p>
      ) : null}
    </>
  )
}
