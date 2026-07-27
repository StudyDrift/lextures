import { useEffect, useState } from 'react'
import { latexAccessibleLabel, loadKatex, renderKatexSafe } from '../../../../lib/math'

type MathPreviewProps = {
  source: string
  label: string
}

/** Live KaTeX preview for plain-text math input (CT.18 FR-11). */
export function MathPreview({ source, label }: MathPreviewProps) {
  const [html, setHtml] = useState<string>('')
  const [failed, setFailed] = useState(false)
  const trimmed = source.trim()

  useEffect(() => {
    let cancelled = false
    if (!trimmed) {
      setHtml('')
      setFailed(false)
      return
    }
    // Convert plain-text math (^, *) into something KaTeX can render lightly.
    const latex = toLightLatex(trimmed)
    void loadKatex().then((katex) => {
      if (cancelled) return
      const out = renderKatexSafe(katex, latex, false)
      setHtml(out.html)
      setFailed(out.failed)
    })
    return () => {
      cancelled = true
    }
  }, [trimmed])

  if (!trimmed) {
    return (
      <p className="text-sm text-[color:var(--lex-muted)]" aria-live="polite">
        {label}
      </p>
    )
  }

  return (
    <div
      className="rounded-md border border-[color:var(--lex-border)] bg-[color:var(--lex-surface-2,#f8fafc)] px-3 py-2 text-sm"
      aria-live="polite"
      aria-label={latexAccessibleLabel(trimmed, false)}
      data-testid="worked-example-math-preview"
    >
      {failed ? (
        <code className="text-[color:var(--lex-fg)]">{trimmed}</code>
      ) : (
        <span dangerouslySetInnerHTML={{ __html: html }} />
      )}
      <span className="sr-only">{trimmed}</span>
    </div>
  )
}

/** Minimal plain-text → LaTeX for preview (not a CAS). */
function toLightLatex(s: string): string {
  return s
    .replace(/\*/g, '\\cdot ')
    .replace(/\^(\d+)/g, '^{$1}')
    .replace(/\^([a-zA-Z])/g, '^{$1}')
}
