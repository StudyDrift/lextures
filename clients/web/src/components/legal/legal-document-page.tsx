import { useEffect, useMemo } from 'react'
import { Link } from 'react-router-dom'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { BrandLogo } from '../brand-logo'
import {
  extractTocEntries,
  slugifyHeading,
  type LegalDocumentConfig,
} from '../../lib/legal-documents'

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

export function LegalDocumentPage({ document: doc, showHistory }: LegalDocumentPageProps) {
  const markdown = showHistory ? doc.historyMarkdown : doc.bodyMarkdown
  const toc = useMemo(() => extractTocEntries(markdown), [markdown])

  useEffect(() => {
    window.document.title = showHistory ? `${doc.title} — History` : doc.title
  }, [doc.title, showHistory])

  return (
    <div className="min-h-dvh bg-slate-50 text-slate-900 dark:bg-neutral-950 dark:text-neutral-100">
      <LegalJsonLd legalDoc={doc} />
      <header className="border-b border-slate-200 bg-white px-4 py-4 dark:border-neutral-800 dark:bg-neutral-900 sm:px-6">
        <div className="mx-auto flex max-w-5xl flex-wrap items-center justify-between gap-3">
          <Link to="/login" className="inline-flex items-center gap-2 text-sm font-medium text-slate-700 dark:text-neutral-200">
            <BrandLogo className="h-7 w-auto" />
            <span className="sr-only">Lextures home</span>
          </Link>
          <nav aria-label="Legal" className="flex flex-wrap gap-3 text-sm">
            <Link to="/privacy" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">
              Privacy
            </Link>
            <Link to="/terms" className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300">
              Terms
            </Link>
            <Link to="/login" className="text-slate-600 hover:text-slate-900 dark:text-neutral-400 dark:hover:text-neutral-100">
              Sign in
            </Link>
          </nav>
        </div>
      </header>

      <div className="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-8 sm:px-6 lg:flex-row lg:py-10">
        {!showHistory && toc.length > 0 ? (
          <aside className="lg:w-56 lg:shrink-0">
            <nav
              aria-label="Table of contents"
              className="sticky top-6 rounded-lg border border-slate-200 bg-white p-4 text-sm dark:border-neutral-800 dark:bg-neutral-900"
            >
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500 dark:text-neutral-500">
                On this page
              </p>
              <ol className="space-y-1.5">
                {toc.map((entry) => (
                  <li key={entry.id}>
                    <a
                      href={`#${entry.id}`}
                      className="text-slate-700 underline-offset-2 hover:text-indigo-700 hover:underline dark:text-neutral-300 dark:hover:text-indigo-300"
                    >
                      {entry.title}
                    </a>
                  </li>
                ))}
              </ol>
              {doc.id === 'privacy_policy' ? (
                <p className="mt-4 border-t border-slate-100 pt-3 dark:border-neutral-800">
                  <a
                    href="#your-rights-under-gdpr"
                    className="font-medium text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300"
                  >
                    Jump to rights
                  </a>
                </p>
              ) : null}
            </nav>
          </aside>
        ) : null}

        <article className="min-w-0 flex-1 print:max-w-none">
          <header className="mb-6 border-b border-slate-200 pb-6 dark:border-neutral-800">
            <h1 className="text-3xl font-semibold tracking-tight text-slate-900 dark:text-neutral-50">
              {showHistory ? `${doc.title} — History of changes` : doc.title}
            </h1>
            {!showHistory ? (
              <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-600 dark:text-neutral-400">
                <div>
                  <dt className="inline font-medium text-slate-800 dark:text-neutral-300">Effective date: </dt>
                  <dd className="inline">{doc.effectiveDateLabel}</dd>
                </div>
                <div>
                  <dt className="inline font-medium text-slate-800 dark:text-neutral-300">Version: </dt>
                  <dd className="inline font-mono text-xs">{doc.version}</dd>
                </div>
              </dl>
            ) : null}
            <p className="mt-3 text-sm">
              <Link
                to={showHistory ? doc.path : doc.historyPath}
                className="text-indigo-700 underline-offset-2 hover:underline dark:text-indigo-300"
              >
                {showHistory ? `Back to ${doc.title}` : 'History of changes'}
              </Link>
            </p>
          </header>

          <div className="legal-prose max-w-none text-base leading-relaxed text-slate-800 dark:text-neutral-200 [&_a]:text-indigo-700 [&_a]:underline [&_h2]:scroll-mt-24 [&_h2]:border-b [&_h2]:border-slate-100 [&_h2]:pb-2 [&_h2]:pt-8 [&_h2]:text-xl [&_h2]:font-semibold [&_h3]:mt-4 [&_h3]:text-lg [&_h3]:font-semibold [&_li]:my-1 [&_ol]:list-decimal [&_ol]:ps-6 [&_p]:my-3 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-slate-200 [&_td]:p-2 [&_th]:border [&_th]:border-slate-200 [&_th]:bg-slate-50 [&_th]:p-2 [&_ul]:list-disc [&_ul]:ps-6 dark:[&_a]:text-indigo-300 dark:[&_h2]:border-neutral-800 dark:[&_td]:border-neutral-700 dark:[&_th]:border-neutral-700 dark:[&_th]:bg-neutral-900">
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
                a: ({ href, children }) => {
                  if (href?.startsWith('/')) {
                    return (
                      <Link to={href} className="text-indigo-700 underline dark:text-indigo-300">
                        {children}
                      </Link>
                    )
                  }
                  return (
                    <a href={href} className="text-indigo-700 underline dark:text-indigo-300">
                      {children}
                    </a>
                  )
                },
              }}
            >
              {markdown}
            </ReactMarkdown>
          </div>
        </article>
      </div>
    </div>
  )
}
