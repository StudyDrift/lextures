import { useEffect, useMemo, type RefObject } from 'react'
import { Link } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import researchMarkdown from '../../content/checklist/research.md?raw'
import { buildSourceIndex, sourceToAnchorId } from '../../lib/checklist-research-anchors'

const proseClassName = `space-y-4 text-sm leading-relaxed text-slate-700 dark:text-neutral-200
  [&_a]:font-medium [&_a]:text-indigo-700 [&_a]:underline-offset-2 hover:[&_a]:underline
  dark:[&_a]:text-indigo-300
  [&_h1]:text-2xl [&_h1]:font-semibold [&_h1]:text-slate-900 dark:[&_h1]:text-neutral-50
  [&_h2]:mt-8 [&_h2]:text-xl [&_h2]:font-semibold [&_h2]:text-slate-900 dark:[&_h2]:text-neutral-50
  [&_h3]:mt-6 [&_h3]:text-lg [&_h3]:font-semibold [&_h3]:text-slate-900 dark:[&_h3]:text-neutral-50
  [&_p]:my-3
  [&_ul]:my-3 [&_ul]:list-disc [&_ul]:pl-5
  [&_ol]:my-3 [&_ol]:list-decimal [&_ol]:pl-5
  [&_li]:my-1
  [&_blockquote]:border-l-4 [&_blockquote]:border-slate-200 [&_blockquote]:pl-3
  dark:[&_blockquote]:border-neutral-700
  [&_table]:my-4 [&_table]:w-full [&_table]:border-collapse [&_table]:text-left [&_table]:text-xs
  sm:[&_table]:text-sm
  [&_th]:border-b [&_th]:border-slate-200 [&_th]:bg-slate-100 [&_th]:px-2 [&_th]:py-2 [&_th]:font-semibold
  dark:[&_th]:border-neutral-700 dark:[&_th]:bg-neutral-900
  [&_td]:border-b [&_td]:border-slate-100 [&_td]:px-2 [&_td]:py-2 align-top
  dark:[&_td]:border-neutral-800
  [&_code]:rounded [&_code]:bg-slate-100 [&_code]:px-1 [&_code]:py-0.5 [&_code]:text-[0.9em]
  dark:[&_code]:bg-neutral-800
  [&_hr]:my-8 [&_hr]:border-slate-200 dark:[&_hr]:border-neutral-800`

type Props = {
  className?: string
  /** When set, scroll this source's anchor into view after mount (dialog / hash). */
  focusSource?: string | null
  /** Scroll container (dialog body). Defaults to the nearest scrollable ancestor / window. */
  scrollRootRef?: RefObject<HTMLElement | null>
}

/** Shared research body: narrative doc + per-source anchor index for chips. */
export function ChecklistResearchBody({ className = '', focusSource, scrollRootRef }: Props) {
  const markdown = useMemo(
    () => rewriteRepoRelativeLinks(stripLeadingH1(researchMarkdown)),
    [],
  )
  const sourceIndex = useMemo(() => buildSourceIndex(), [])
  const focusId = focusSource ? sourceToAnchorId(focusSource) : null

  useEffect(() => {
    if (!focusId) return
    let highlightTimer = 0
    const frame = requestAnimationFrame(() => {
      const el = document.getElementById(focusId)
      if (!el) return
      const root = scrollRootRef?.current
      if (root) {
        const elRect = el.getBoundingClientRect()
        const rootRect = root.getBoundingClientRect()
        const top = root.scrollTop + (elRect.top - rootRect.top) - 12
        root.scrollTo({ top: Math.max(0, top), behavior: 'smooth' })
      } else {
        el.scrollIntoView({ behavior: 'smooth', block: 'start' })
      }
      el.classList.add('ring-2', 'ring-amber-400/70', 'rounded-lg')
      highlightTimer = window.setTimeout(() => {
        el.classList.remove('ring-2', 'ring-amber-400/70', 'rounded-lg')
      }, 2200)
    })
    return () => {
      cancelAnimationFrame(frame)
      if (highlightTimer) window.clearTimeout(highlightTimer)
    }
  }, [focusId, scrollRootRef])

  return (
    <div className={`${proseClassName} ${className}`.trim()}>
      <article>
        <ReactMarkdown
          remarkPlugins={[remarkGfm]}
          components={{
            a: ({ href, children }) => {
              if (!href) return <span>{children}</span>
              if (href.startsWith('http://') || href.startsWith('https://')) {
                return (
                  <a href={href} target="_blank" rel="noopener noreferrer">
                    {children}
                  </a>
                )
              }
              if (href.startsWith('/')) {
                return <Link to={href}>{children}</Link>
              }
              if (href.startsWith('#')) {
                return <a href={href}>{children}</a>
              }
              return <span className="font-medium text-slate-800 dark:text-neutral-100">{children}</span>
            },
          }}
        >
          {markdown}
        </ReactMarkdown>
      </article>

      <section className="mt-10 border-t border-slate-200 pt-8 dark:border-neutral-800" aria-labelledby="standards-index-heading">
        <h2 id="standards-index-heading" className="!mt-0">
          Standards index
        </h2>
        <p className="text-sm text-slate-600 dark:text-neutral-400">
          Each source chip on the checklist links here. Use these anchors to share a specific standard.
        </p>
        <ul className="!list-none !pl-0 space-y-4">
          {sourceIndex.map((entry) => (
            <li
              key={entry.anchorId}
              id={entry.anchorId}
              className="scroll-mt-4 rounded-lg border border-slate-200 bg-slate-50/80 p-4 dark:border-neutral-700 dark:bg-neutral-900/60"
            >
              <h3 className="!mt-0 text-base font-semibold text-slate-900 dark:text-neutral-50">
                <a href={`#${entry.anchorId}`} className="!no-underline hover:!underline">
                  {entry.source}
                </a>
              </h3>
              <p className="!my-1 text-xs text-slate-500 dark:text-neutral-400">
                Checklist items that cite this source:
              </p>
              <ul className="!my-1 !list-disc !pl-5 text-sm">
                {entry.items.map((item) => (
                  <li key={item.title} className="!my-0.5">
                    {item.title}
                  </li>
                ))}
              </ul>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

function rewriteRepoRelativeLinks(src: string): string {
  return src.replace(/\[([^\]]+)\]\((?!https?:\/\/)([^)]+\.md(?:#[^)]*)?)\)/g, '**$1**')
}

function stripLeadingH1(src: string): string {
  return src.replace(/^#\s+[^\n]+\n+/, '')
}
