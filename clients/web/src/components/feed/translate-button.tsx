import { useState } from 'react'
import { Languages } from 'lucide-react'
import { translateContent, type ContentType } from '../../lib/translation-api'

type TranslateButtonProps = {
  contentType: ContentType
  contentId: string
  text: string
  targetLang?: string
}

const LANG_NAMES: Record<string, string> = {
  en: 'English', es: 'Spanish', fr: 'French', de: 'German', it: 'Italian',
  pt: 'Portuguese', zh: 'Chinese', ja: 'Japanese', ko: 'Korean', ar: 'Arabic',
  ru: 'Russian', hi: 'Hindi', nl: 'Dutch', pl: 'Polish', sv: 'Swedish',
  tr: 'Turkish', uk: 'Ukrainian', vi: 'Vietnamese', th: 'Thai', id: 'Indonesian',
}

function langName(code: string): string {
  return LANG_NAMES[code.toLowerCase()] ?? code.toUpperCase()
}

export function TranslateButton({ contentType, contentId, text, targetLang = 'en' }: TranslateButtonProps) {
  const [state, setState] = useState<'idle' | 'loading' | 'done' | 'error'>('idle')
  const [translated, setTranslated] = useState<string | null>(null)
  const [sourceLang, setSourceLang] = useState<string | null>(null)
  const [showOriginal, setShowOriginal] = useState(false)
  const [errorMsg, setErrorMsg] = useState<string | null>(null)

  async function handleTranslate() {
    if (state === 'loading') return
    if (state === 'done') {
      setShowOriginal((v) => !v)
      return
    }
    if (!text.trim()) return

    setState('loading')
    setErrorMsg(null)
    try {
      const result = await translateContent(contentType, contentId, targetLang, text)
      setTranslated(result.translated)
      setSourceLang(result.source_lang)
      setState('done')
      setShowOriginal(false)
    } catch {
      setErrorMsg('Translation temporarily unavailable.')
      setState('error')
    }
  }

  return (
    <div className="mt-1.5">
      {(state === 'idle' || state === 'error') && (
        <button
          type="button"
          aria-label="Translate this message"
          onClick={() => void handleTranslate()}
          className="inline-flex items-center gap-1 text-xs text-fg-subtle transition-colors hover:text-accent-fg dark:hover:text-indigo-400"
        >
          <Languages className="h-3.5 w-3.5" strokeWidth={1.75} />
          Translate
        </button>
      )}

      {state === 'loading' && (
        <span className="inline-flex items-center gap-1 text-xs text-fg-subtle">
          <Languages className="h-3.5 w-3.5 animate-pulse" strokeWidth={1.75} />
          Translating…
        </span>
      )}

      {state === 'done' && translated && !showOriginal && (
        <div>
          <p
            lang={targetLang}
            className="mt-1.5 rounded-md bg-surface-base px-3 py-2 text-[0.9375rem] leading-relaxed text-fg-muted ring-1 ring-inset ring-slate-200 dark:bg-surface-raised dark:text-fg-muted dark:ring-neutral-700"
          >
            {translated}
          </p>
          <p className="mt-1 text-xs text-fg-subtle">
            {sourceLang && sourceLang !== 'und' ? (
              <>Translated from {langName(sourceLang)} · </>
            ) : (
              <>Translated · </>
            )}
            <button
              type="button"
              className="underline hover:text-fg-muted dark:hover:text-fg-muted"
              onClick={() => setShowOriginal(true)}
            >
              Show original
            </button>
          </p>
        </div>
      )}

      {state === 'done' && showOriginal && (
        <p className="mt-1 text-xs text-fg-subtle">
          <button
            type="button"
            className="inline-flex items-center gap-1 underline hover:text-accent-fg dark:hover:text-indigo-400"
            onClick={() => setShowOriginal(false)}
          >
            Show translation
          </button>
        </p>
      )}

      {state === 'error' && errorMsg && (
        <p className="mt-1 text-xs text-rose-600 dark:text-rose-400">{errorMsg}</p>
      )}
    </div>
  )
}
