import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ExternalLink, FileText, FolderOpen } from 'lucide-react'
import { getPublicPortfolio, type PublicPortfolio } from '../../lib/eportfolio-api'
import { EmptyState } from '../../components/ui/empty-state'


export default function PublicPortfolioPage() {
  const { slug = '' } = useParams<{ slug: string }>()
  const [portfolio, setPortfolio] = useState<PublicPortfolio | null>(null)
  const [status, setStatus] = useState<'loading' | 'ready' | 'notfound' | 'error'>('loading')

  useEffect(() => {
    let active = true
    setStatus('loading')
    getPublicPortfolio(slug)
      .then((p) => {
        if (!active) return
        if (!p) {
          setStatus('notfound')
          return
        }
        setPortfolio(p)
        setStatus('ready')
        document.title = `${p.title} — ePortfolio`
      })
      .catch(() => active && setStatus('error'))
    return () => {
      active = false
    }
  }, [slug])

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
        <h1 className="text-2xl font-semibold">Portfolio not found</h1>
        <p className="mt-2 text-muted-foreground">
          This portfolio link is invalid or is no longer shared publicly.
        </p>
      </main>
    )
  }

  if (status === 'error' || !portfolio) {
    return (
      <main className="mx-auto max-w-3xl px-4 py-16 text-center">
        <h1 className="text-2xl font-semibold">Something went wrong</h1>
        <p className="mt-2 text-muted-foreground">Please try again later.</p>
      </main>
    )
  }

  return (
    <div className="min-h-screen bg-slate-50/50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <a
        href="#portfolio-artifacts"
        className="sr-only focus:not-sr-only focus:absolute focus:start-2 focus:top-2 focus:z-10 focus:rounded focus:bg-primary focus:px-3 focus:py-1.5 focus:text-sm focus:text-primary-foreground"
      >
        Skip to artifacts
      </a>
      <header className="border-b border-slate-100 bg-white dark:border-neutral-800 dark:bg-neutral-900/50">
        <div className="mx-auto max-w-3xl px-4 py-12">
          <h1 className="text-3xl font-bold tracking-tight text-slate-900 dark:text-neutral-50">{portfolio.title}</h1>
          {portfolio.ownerName && (
            <p className="mt-2 text-sm font-semibold text-indigo-650 dark:text-indigo-400">{portfolio.ownerName}</p>
          )}
          {portfolio.introText && (
            <p className="mt-5 max-w-2xl text-base leading-relaxed text-slate-600 dark:text-neutral-350">{portfolio.introText}</p>
          )}
        </div>
      </header>

      <main id="portfolio-artifacts" className="mx-auto max-w-3xl px-4 py-10">
        {portfolio.artifacts.length === 0 ? (
          <EmptyState
            icon={FolderOpen}
            title="No public artifacts"
            body="There are no public artifacts in this portfolio yet."
          />
        ) : (
          <div className="grid gap-6 sm:grid-cols-2">
            {portfolio.artifacts.map((a) => (
              <article
                key={a.id}
                role="article"
                className="flex flex-col rounded-2xl border border-slate-200/80 bg-white p-5 shadow-sm transition hover:-translate-y-0.5 hover:border-slate-300 hover:shadow-md dark:border-neutral-800 dark:bg-neutral-900 dark:hover:border-neutral-700"
              >
                <h2 className="flex items-center gap-2 text-base font-semibold text-slate-900 dark:text-neutral-100">
                  <FileText className="h-4 w-4 text-indigo-500" aria-hidden />
                  {a.title}
                </h2>
                {a.description && (
                  <p className="mt-2 text-sm text-slate-600 dark:text-neutral-400">{a.description}</p>
                )}
                {a.artifactType === 'text_page' && a.textContent && (
                  <p className="mt-3 flex-1 whitespace-pre-wrap text-sm leading-relaxed text-slate-700 dark:text-neutral-300">{a.textContent}</p>
                )}
                {a.artifactType === 'url' && a.externalUrl && (
                  <div className="mt-4 flex-1">
                    <a
                      href={a.externalUrl}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-sm font-semibold text-indigo-600 hover:text-indigo-505 dark:text-indigo-400 dark:hover:text-indigo-300"
                    >
                      <ExternalLink className="h-3.5 w-3.5" aria-hidden /> View resource
                    </a>
                  </div>
                )}
                {a.fileName && (
                  <div className="mt-3 rounded-lg bg-slate-50 px-2.5 py-1.5 text-xs text-slate-500 dark:bg-neutral-950 dark:text-neutral-400">
                    Attachment: <span className="font-medium">{a.fileName}</span>
                  </div>
                )}
                {a.outcomeIds.length > 0 && (
                  <ul className="mt-4 flex flex-wrap gap-1.5" aria-label="Aligned outcomes">
                    {a.outcomeIds.map((oid) => (
                      <li
                        key={oid}
                        className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-semibold text-slate-600 dark:bg-neutral-800 dark:text-neutral-400"
                      >
                        Outcome
                      </li>
                    ))}
                  </ul>
                )}
              </article>
            ))}
          </div>
        )}
      </main>

      <footer className="border-t border-slate-150 bg-slate-50/50 dark:border-neutral-800 dark:bg-neutral-950">
        <div className="mx-auto max-w-3xl px-4 py-8 text-xs text-slate-500 dark:text-neutral-500">
          <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:justify-between">
            <p>Conforms to WCAG 2.1 AA accessibility standards.</p>
            <p className="font-semibold">{portfolio.viewCount} view{portfolio.viewCount === 1 ? '' : 's'}</p>
          </div>
        </div>
      </footer>
    </div>
  )
}
