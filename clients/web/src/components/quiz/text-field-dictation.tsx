import { useRef, useState } from 'react'
import { DictationButton } from '../editor/block-editor/dictation-button'
import { commitTextAtSelection, insertTextAtSelection } from '../../lib/stt/text-insert'

export type TextFieldDictationProps = {
  getInput: () => HTMLTextAreaElement | HTMLInputElement | null
  value: string
  onChange: (next: string) => void
  disabled?: boolean
  language?: string
  accommodationTooltip?: string
}

/** Dictation controls for plain textarea/input quiz short-answer fields. */
export function TextFieldDictation({
  getInput,
  value,
  onChange,
  disabled,
  language,
  accommodationTooltip,
}: TextFieldDictationProps) {
  const interimRef = useRef<{ start: number | null; length: number }>({ start: null, length: 0 })
  const [displayInterim, setDisplayInterim] = useState('')

  return (
    <div className="mt-2 flex flex-col gap-1">
      <DictationButton
        disabled={disabled}
        language={language}
        accommodationTooltip={accommodationTooltip}
        onInterimResult={(text) => {
          setDisplayInterim(text)
          const el = getInput()
          if (!el) return
          const { value: next, caret, interimStart, interimLength } = insertTextAtSelection(
            el,
            text,
            interimRef.current.start,
            interimRef.current.length,
          )
          interimRef.current = { start: interimStart, length: interimLength }
          onChange(next)
          requestAnimationFrame(() => {
            el.focus()
            el.setSelectionRange(caret, caret)
          })
        }}
        onFinalResult={(text) => {
          setDisplayInterim('')
          const el = getInput()
          if (!el) {
            onChange(value + text)
            interimRef.current = { start: null, length: 0 }
            return
          }
          const { value: next, caret } = commitTextAtSelection(
            el,
            text,
            interimRef.current.start,
            interimRef.current.length,
          )
          interimRef.current = { start: null, length: 0 }
          onChange(next)
          requestAnimationFrame(() => {
            el.focus()
            el.setSelectionRange(caret, caret)
          })
        }}
      />
      {displayInterim ? (
        <p className="text-sm italic text-slate-400 dark:text-neutral-500" aria-hidden>
          {displayInterim}
        </p>
      ) : null}
    </div>
  )
}
