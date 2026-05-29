import { useEffect, useMemo, type ReactNode } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Header } from '../components/header'
import { LegalNav } from '../components/legal-nav'
import { SiteFooter } from '../components/site-footer'
import {
  extractTocEntries,
  slugifyHeading,
  type LegalDocumentConfig,
} from '../lib/legal-documents'

type LegalDocumentPageProps = {
  document: LegalDocumentConfig
  showHistory?: boolean
}

function LegalJsonLd({ legalDoc }: { legalDoc: LegalDocumentConfig }) {
  useEffect(() => {
    const scriptId = 'legal-json-ld'
    let el = window.document.getElementById(scriptId) as HTMLScriptElement | null
    if (!el) {
      el = window.document.createElement('script')
      el.id = scriptId
      el.type = 'application/ld+json'
      window.document.head.appendChild(el)
    }
    const payload = {
      '@context': 'https://schema.org',
      '@type': legalDoc.jsonLdType,
      name: legalDoc.title,
      url: `${window.location.origin}${legalDoc.path}`,
      dateModified: legalDoc.version,
      publisher: { '@type': 'Organization', name: 'Lextures, Inc.' },
    }
    el.textContent = JSON.stringify(payload)
    return () => {
      el?.remove()
    }
  }, [legalDoc])

  return null
}

function LegalMarkdownLink({ href, children }: { href?: string; children?: ReactNode }) {
  if (href?.startsWith('/')) {
    return (
      <a href={href} className="text-accent underline underline-offset-2 hover:text-accent-hover">
        {children}
      </a>
    )
  }
  return (
    <a
      href={href}
      className="text-accent underline underline-offset-2 hover:text-accent-hover"
      {...(href?.startsWith('http') ? { target: '_blank', rel: 'noopener noreferrer' } : {})}
    >
      {children}
    </a>
  )
}

export function LegalDocumentPage({ document: doc, showHistory }: LegalDocumentPageProps) {
  const markdown = showHistory ? doc.historyMarkdown : doc.bodyMarkdown
  const toc = useMemo(() => extractTocEntries(markdown), [markdown])

  useEffect(() => {
    window.document.title = showHistory ? `${doc.title} — History` : `${doc.title} — Lextures`
  }, [doc.title, showHistory])

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-stone-50 text-slate-700">
      <LegalJsonLd legalDoc={doc} />
      <Header />

      <div className="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-8 sm:px-6 lg:flex-row lg:py-10">
        {!showHistory && toc.length > 0 ? (
          <aside className="lg:w-56 lg:shrink-0">
            <nav
              aria-label="Table of contents"
              className="sticky top-24 rounded-xl border border-stone-200/90 bg-white p-4 text-sm shadow-sm"
            >
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-stone-500">
                On this page
              </p>
              <ol className="space-y-1.5">
                {toc.map((entry) => (
                  <li key={entry.id}>
                    <a
                      href={`#${entry.id}`}
                      className="text-stone-700 no-underline underline-offset-2 transition-colors hover:text-accent hover:underline"
                    >
                      {entry.title}
                    </a>
                  </li>
                ))}
              </ol>
              {doc.id === 'privacy_policy' ? (
                <p className="mt-4 border-t border-stone-100 pt-3">
                  <a
                    href="#your-rights-under-gdpr"
                    className="font-medium text-accent no-underline underline-offset-2 hover:underline"
                  >
                    Jump to rights
                  </a>
                </p>
              ) : null}
            </nav>
          </aside>
        ) : null}

        <article className="min-w-0 flex-1">
          <header className="mb-6 border-b border-stone-200/90 pb-6">
            <LegalNav />
            <h1 className="font-display text-3xl font-normal tracking-tight text-stone-900 sm:text-4xl">
              {showHistory ? `${doc.title} — History of changes` : doc.title}
            </h1>
            {!showHistory ? (
              <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-stone-600">
                <div>
                  <dt className="inline font-medium text-stone-800">Effective date: </dt>
                  <dd className="inline">{doc.effectiveDateLabel}</dd>
                </div>
                <div>
                  <dt className="inline font-medium text-stone-800">Version: </dt>
                  <dd className="inline font-mono text-xs">{doc.version}</dd>
                </div>
              </dl>
            ) : null}
            <p className="mt-3 text-sm">
              <a
                href={showHistory ? doc.path : doc.historyPath}
                className="font-medium text-accent no-underline underline-offset-2 hover:underline"
              >
                {showHistory ? `Back to ${doc.title}` : 'History of changes'}
              </a>
            </p>
          </header>

          <div className="prose-content legal-prose [&_h2]:scroll-mt-28 [&_h2]:border-b [&_h2]:border-stone-200/90 [&_h2]:pb-2 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-stone-200 [&_td]:p-2 [&_th]:border [&_th]:border-stone-200 [&_th]:bg-stone-50 [&_th]:p-2">
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                h2: ({ children }) => {
                  const text = String(children)
                  const id = slugifyHeading(text)
                  return (
                    <h2 id={id} tabIndex={-1}>
                      {children}
                    </h2>
                  )
                },
                a: ({ href, children }) => <LegalMarkdownLink href={href}>{children}</LegalMarkdownLink>,
              }}
            >
              {markdown}
            </ReactMarkdown>
          </div>
        </article>
      </div>

      <SiteFooter />
    </div>
  )
}
