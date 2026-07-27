import { useId, type KeyboardEvent } from 'react'

type Props = {
  value: string
  onChange: (next: string) => void
  language: string
  readOnly?: boolean
  mode: 'plain' | 'rich'
  onEscapeBlur?: () => void
  ariaLabel: string
  describedBy?: string
}

/** Accessible code editor: plain textarea, with light rich helpers (indent / brackets). */
export function CodeEditor({
  value,
  onChange,
  language,
  readOnly,
  mode,
  onEscapeBlur,
  ariaLabel,
  describedBy,
}: Props) {
  const id = useId()

  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Escape') {
      onEscapeBlur?.()
      e.currentTarget.blur()
      return
    }
    if (readOnly || mode !== 'rich') return
    if (e.key === 'Tab') {
      e.preventDefault()
      const el = e.currentTarget
      const start = el.selectionStart
      const end = el.selectionEnd
      const next = value.slice(0, start) + '  ' + value.slice(end)
      onChange(next)
      requestAnimationFrame(() => {
        el.selectionStart = el.selectionEnd = start + 2
      })
      return
    }
    if (e.key === 'Enter') {
      const el = e.currentTarget
      const start = el.selectionStart
      const lineStart = value.lastIndexOf('\n', start - 1) + 1
      const line = value.slice(lineStart, start)
      const indent = line.match(/^\s*/)?.[0] ?? ''
      if (!indent) return
      e.preventDefault()
      const next = value.slice(0, start) + '\n' + indent + value.slice(el.selectionEnd)
      onChange(next)
      requestAnimationFrame(() => {
        el.selectionStart = el.selectionEnd = start + 1 + indent.length
      })
      return
    }
    const pairs: Record<string, string> = { '(': ')', '[': ']', '{': '}', '"': '"', "'": "'" }
    if (pairs[e.key]) {
      const el = e.currentTarget
      const start = el.selectionStart
      const end = el.selectionEnd
      if (start !== end) return
      e.preventDefault()
      const close = pairs[e.key]!
      const next = value.slice(0, start) + e.key + close + value.slice(end)
      onChange(next)
      requestAnimationFrame(() => {
        el.selectionStart = el.selectionEnd = start + 1
      })
    }
  }

  return (
    <textarea
      id={id}
      data-testid="code-sandbox-editor"
      data-editor-mode={mode}
      data-language={language}
      className="min-h-[10rem] w-full resize-y rounded border border-slate-300 bg-white px-3 py-2 font-mono text-sm leading-5 text-slate-900 outline-none focus-visible:ring-2 focus-visible:ring-sky-500 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100"
      spellCheck={false}
      autoCapitalize="off"
      autoCorrect="off"
      readOnly={readOnly}
      aria-label={ariaLabel}
      aria-describedby={describedBy}
      value={value}
      onChange={(e) => onChange(e.target.value)}
      onKeyDown={onKeyDown}
      dir="ltr"
    />
  )
}
