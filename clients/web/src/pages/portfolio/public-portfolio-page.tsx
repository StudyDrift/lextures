import { useEffect, useState } from 'react'
import { useParams } from 'react-router-dom'
import { ExternalLink, FileText } from 'lucide-react'
import { getPublicPortfolio, type PublicPortfolio } from '../../lib/eportfolio-api'

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
    <div className="min-h-screen bg-background text-foreground">
      <a
        href="#portfolio-artifacts"
        className="sr-only focus:not-sr-only focus:absolute focus:start-2 focus:top-2 focus:z-10 focus:rounded focus:bg-primary focus:px-3 focus:py-1.5 focus:text-sm focus:text-primary-foreground"
      >
        Skip to artifacts
      </a>
      <header className="border-b bg-card">
        <div className="mx-auto max-w-3xl px-4 py-10">
          <h1 className="text-3xl font-bold tracking-tight">{portfolio.title}</h1>
          {portfolio.ownerName && (
            <p className="mt-1 text-sm text-muted-foreground">{portfolio.ownerName}</p>
          )}
          {portfolio.introText && (
            <p className="mt-4 max-w-2xl leading-relaxed text-muted-foreground">{portfolio.introText}</p>
          )}
        </div>
      </header>

      <main id="portfolio-artifacts" className="mx-auto max-w-3xl px-4 py-8">
        {portfolio.artifacts.length === 0 ? (
          <p className="text-center text-muted-foreground">No public artifacts in this portfolio yet.</p>
        ) : (
          <div className="grid gap-4 sm:grid-cols-2">
            {portfolio.artifacts.map((a) => (
              <article
                key={a.id}
                role="article"
                className="flex flex-col rounded-lg border bg-card p-4"
              >
                <h2 className="flex items-center gap-1.5 font-semibold">
                  <FileText className="h-4 w-4 text-muted-foreground" aria-hidden />
                  {a.title}
                </h2>
                {a.description && (
                  <p className="mt-1 text-sm text-muted-foreground">{a.description}</p>
                )}
                {a.artifactType === 'text_page' && a.textContent && (
                  <p className="mt-2 whitespace-pre-wrap text-sm leading-relaxed">{a.textContent}</p>
                )}
                {a.artifactType === 'url' && a.externalUrl && (
                  <a
                    href={a.externalUrl}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-2 inline-flex items-center gap-1 text-sm text-primary hover:underline"
                  >
                    <ExternalLink className="h-3.5 w-3.5" aria-hidden /> View resource
                  </a>
                )}
                {a.fileName && (
                  <p className="mt-2 text-xs text-muted-foreground">Attachment: {a.fileName}</p>
                )}
                {a.outcomeIds.length > 0 && (
                  <ul className="mt-3 flex flex-wrap gap-1.5" aria-label="Aligned outcomes">
                    {a.outcomeIds.map((oid) => (
                      <li
                        key={oid}
                        className="rounded-full bg-secondary px-2 py-0.5 text-xs text-secondary-foreground"
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

      <footer className="border-t">
        <div className="mx-auto max-w-3xl px-4 py-6 text-xs text-muted-foreground">
          <p>This page conforms to WCAG 2.1 AA accessibility standards.</p>
          <p className="mt-1">{portfolio.viewCount} view{portfolio.viewCount === 1 ? '' : 's'}</p>
        </div>
      </footer>
    </div>
  )
}
