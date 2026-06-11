import { useEffect, useMemo, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { MarkdownArticleView } from '../../components/syllabus/syllabus-markdown-view'
import { getPublicPortfolioContentPage } from '../../lib/eportfolio-api'
import {
  type ResolvedMarkdownTheme,
  resolveMarkdownTheme,
} from '../../lib/markdown-theme'
import { useLmsDarkMode } from '../../hooks/use-lms-dark-mode'

export default function PublicPortfolioContentPage() {
  const { slug = '', aid = '' } = useParams<{ slug: string; aid: string }>()
  const [title, setTitle] = useState('')
  const [portfolioTitle, setPortfolioTitle] = useState('')
  const [ownerName, setOwnerName] = useState('')
  const [markdown, setMarkdown] = useState('')
  const [status, setStatus] = useState<'loading' | 'ready' | 'notfound' | 'error'>('loading')

  const lmsUiDark = useLmsDarkMode()
  const mdTheme = useMemo(
    (): ResolvedMarkdownTheme => resolveMarkdownTheme('classic', null, { lmsUiDark }),
    [lmsUiDark],
  )

  useEffect(() => {
    let active = true
    setStatus('loading')
    getPublicPortfolioContentPage(slug, aid)
      .then((result) => {
        if (!active) return
        if (!result) {
          setStatus('notfound')
          return
        }
        setTitle(result.artifact.title)
        setMarkdown(result.artifact.textContent)
        setPortfolioTitle(result.portfolio.title)
        setOwnerName(result.portfolio.ownerName)
        setStatus('ready')
        document.title = `${result.artifact.title} — ${result.portfolio.title}`
      })
      .catch(() => active && setStatus('error'))
    return () => {
      active = false
    }
  }, [slug, aid])

  if (status === 'loading') {
    return (
      <main className="mx-auto max-w-3xl px-4 py-12">
        <div className="h-32 motion-safe:animate-pulse rounded-lg border bg-card" aria-hidden />
      </main>
    )
  }

  if (status === 'notfound') {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-2xl font-semibold">Page not found</h1>
        <p className="mt-2 text-muted-foreground">
          This content is not available or is no longer shared publicly.
        </p>
        <Link
          to={`/p/${encodeURIComponent(slug)}`}
          className="mt-4 inline-block text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400"
        >
          Back to portfolio
        </Link>
      </main>
    )
  }

  if (status === 'error') {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-2xl font-semibold">Something went wrong</h1>
        <p className="mt-2 text-muted-foreground">Please try again later.</p>
      </main>
    )
  }

  const backTo = `/p/${encodeURIComponent(slug)}`

  return (
    <div className="min-h-screen bg-slate-50/50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <header className="border-b border-slate-100 bg-white dark:border-neutral-800 dark:bg-neutral-900/50">
        <div className="mx-auto max-w-3xl px-4 py-10">
          <Link
            to={backTo}
            className="text-sm font-medium text-indigo-600 hover:text-indigo-500 dark:text-indigo-400 dark:hover:text-indigo-300"
          >
            ← Back to {portfolioTitle || 'portfolio'}
          </Link>
          <h1 className="mt-4 text-3xl font-bold tracking-tight text-slate-900 dark:text-neutral-50">{title}</h1>
          {ownerName ? (
            <p className="mt-2 text-sm font-semibold text-indigo-650 dark:text-indigo-400">{ownerName}</p>
          ) : null}
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-4 py-10">
        <div className="mx-auto w-full max-w-[72ch] min-w-0 text-[1.0625rem] leading-relaxed">
          <MarkdownArticleView
            markdown={markdown}
            theme={mdTheme}
            emptyMessage="This page has no content yet."
          />
        </div>
      </main>
    </div>
  )
}