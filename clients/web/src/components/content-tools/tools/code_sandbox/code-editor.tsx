import { javascript } from '@codemirror/lang-javascript'
import { python } from '@codemirror/lang-python'
import { EditorView, keymap } from '@codemirror/view'
import CodeMirror from '@uiw/react-codemirror'
import { useId, useMemo, type KeyboardEvent } from 'react'
import { useLmsDarkMode } from '../../../../hooks/use-lms-dark-mode'

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

function languageExtension(language: string) {
  const lang = language.trim().toLowerCase()
  if (lang === 'javascript' || lang === 'js' || lang === 'typescript' || lang === 'ts') {
    return javascript({ jsx: false, typescript: lang === 'typescript' || lang === 'ts' })
  }
  // Default: python (code sandbox only supports python | javascript today).
  return python()
}

/** Light theme aligned with LMS surfaces (slate/neutral + mono). */
const lightEditorTheme = EditorView.theme(
  {
    '&': {
      backgroundColor: 'transparent',
      fontSize: '13px',
      color: 'rgb(15 23 42)',
    },
    '.cm-content': {
      caretColor: 'rgb(79 70 229)',
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      padding: '10px 0',
      minHeight: '11rem',
    },
    '.cm-scroller': {
      overflow: 'auto',
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      lineHeight: '1.55',
    },
    '.cm-gutters': {
      backgroundColor: 'rgb(248 250 252)',
      color: 'rgb(100 116 139)',
      border: 'none',
      borderInlineEnd: '1px solid rgb(226 232 240)',
    },
    '.cm-activeLineGutter': {
      backgroundColor: 'rgb(241 245 249)',
    },
    '.cm-activeLine': {
      backgroundColor: 'rgb(241 245 249 / 0.65)',
    },
    '&.cm-focused .cm-selectionBackground, .cm-selectionBackground, .cm-content ::selection': {
      backgroundColor: 'rgb(199 210 254 / 0.55)',
    },
    '&.cm-focused': {
      outline: 'none',
    },
    '.cm-cursor, .cm-dropCursor': {
      borderLeftColor: 'rgb(79 70 229)',
    },
  },
  { dark: false },
)

/** Dark theme overrides so oneDark sits cleanly in the tool card. */
const darkEditorShell = EditorView.theme(
  {
    '&': {
      backgroundColor: 'transparent',
      fontSize: '13px',
    },
    '.cm-content': {
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      padding: '10px 0',
      minHeight: '11rem',
    },
    '.cm-scroller': {
      overflow: 'auto',
      fontFamily:
        'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      lineHeight: '1.55',
    },
    '.cm-gutters': {
      backgroundColor: 'rgb(10 10 10)',
      border: 'none',
      borderInlineEnd: '1px solid rgb(64 64 64)',
    },
    '&.cm-focused': {
      outline: 'none',
    },
  },
  { dark: true },
)

/**
 * Code Sandbox editor.
 * - **rich**: CodeMirror (syntax highlight, line numbers, brackets, indent).
 * - **plain**: accessible textarea for screen-reader / keyboard-first use.
 */
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
  const dark = useLmsDarkMode()

  const extensions = useMemo(() => {
    const esc = keymap.of([
      {
        key: 'Escape',
        run: (view) => {
          onEscapeBlur?.()
          view.contentDOM.blur()
          return true
        },
      },
    ])
    return [languageExtension(language), esc, dark ? darkEditorShell : lightEditorTheme]
  }, [language, dark, onEscapeBlur])

  if (mode === 'plain') {
    return (
      <PlainTextareaEditor
        id={id}
        value={value}
        onChange={onChange}
        language={language}
        readOnly={readOnly}
        onEscapeBlur={onEscapeBlur}
        ariaLabel={ariaLabel}
        describedBy={describedBy}
      />
    )
  }

  return (
    <div
      className="overflow-hidden rounded-xl border border-slate-200 bg-slate-50 shadow-sm focus-within:border-indigo-400 focus-within:ring-2 focus-within:ring-indigo-500/30 dark:border-neutral-600 dark:bg-neutral-950 dark:focus-within:border-indigo-500 dark:focus-within:ring-indigo-400/25"
      data-testid="code-sandbox-editor"
      data-editor-mode="rich"
      data-language={language}
    >
      <CodeMirror
        value={value}
        height="12rem"
        theme={dark ? 'dark' : 'light'}
        extensions={extensions}
        editable={!readOnly}
        readOnly={readOnly}
        basicSetup={{
          lineNumbers: true,
          highlightActiveLineGutter: true,
          highlightActiveLine: true,
          foldGutter: false,
          dropCursor: true,
          allowMultipleSelections: false,
          indentOnInput: true,
          bracketMatching: true,
          closeBrackets: true,
          autocompletion: true,
          rectangularSelection: false,
          crosshairCursor: false,
          highlightSelectionMatches: true,
          searchKeymap: true,
          tabSize: 2,
        }}
        onChange={(next) => onChange(next)}
        aria-label={ariaLabel}
        aria-describedby={describedBy}
        className="text-[13px] [&_.cm-editor]:bg-transparent [&_.cm-editor.cm-focused]:outline-none"
      />
    </div>
  )
}

function PlainTextareaEditor({
  id,
  value,
  onChange,
  language,
  readOnly,
  onEscapeBlur,
  ariaLabel,
  describedBy,
}: {
  id: string
  value: string
  onChange: (next: string) => void
  language: string
  readOnly?: boolean
  onEscapeBlur?: () => void
  ariaLabel: string
  describedBy?: string
}) {
  function onKeyDown(e: KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === 'Escape') {
      onEscapeBlur?.()
      e.currentTarget.blur()
      return
    }
    if (readOnly) return
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
    }
  }

  return (
    <textarea
      id={id}
      data-testid="code-sandbox-editor"
      data-editor-mode="plain"
      data-language={language}
      className="min-h-[12rem] w-full resize-y rounded-xl border border-slate-200 bg-slate-50 px-3.5 py-3 font-mono text-[13px] leading-6 text-slate-900 shadow-sm outline-none placeholder:text-slate-400 focus-visible:border-indigo-400 focus-visible:ring-2 focus-visible:ring-indigo-500/30 disabled:opacity-60 dark:border-neutral-600 dark:bg-neutral-950 dark:text-neutral-100 dark:placeholder:text-neutral-500 dark:focus-visible:border-indigo-500 dark:focus-visible:ring-indigo-400/25"
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
