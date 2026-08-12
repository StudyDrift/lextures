import { useMemo } from 'react'
import { Header } from '../components/header'
import { LegalNav } from '../components/legal-nav'
import { SiteFooter } from '../components/site-footer'
import { MarkdownBody } from '../components/markdown-body'
import {
  extractTocEntries,
  type LegalDocumentConfig,
} from '../lib/legal-documents'

type LegalDocumentPageProps = {
  document: LegalDocumentConfig
  showHistory?: boolean
}

export function LegalDocumentPage({ document: doc, showHistory }: LegalDocumentPageProps) {
  const markdown = showHistory ? doc.historyMarkdown : doc.bodyMarkdown
  const html = showHistory ? doc.historyHtml : doc.bodyHtml
  const toc = useMemo(() => extractTocEntries(markdown), [markdown])

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-white text-slate-900">
      <Header />

      <div className="mx-auto flex max-w-5xl flex-col gap-8 px-4 py-8 sm:px-6 lg:flex-row lg:py-10">
        {!showHistory && toc.length > 0 ? (
          <aside className="lg:w-56 lg:shrink-0">
            <nav
              aria-label="Table of contents"
              className="sticky top-24 rounded-xl border border-slate-200 bg-white p-4 text-sm shadow-sm"
            >
              <p className="mb-2 text-xs font-semibold uppercase tracking-wide text-slate-500">
                On this page
              </p>
              <ol className="space-y-1.5">
                {toc.map(entry => (
                  <li key={entry.id}>
                    <a
                      href={`#${entry.id}`}
                      className="text-slate-700 no-underline underline-offset-2 transition-colors hover:text-accent hover:underline"
                    >
                      {entry.title}
                    </a>
                  </li>
                ))}
              </ol>
              {doc.id === 'privacy_policy' ? (
                <p className="mt-4 border-t border-slate-100 pt-3">
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
          <header className="mb-6 border-b border-slate-200 pb-6">
            <LegalNav />
            <h1 className="font-display text-3xl font-normal tracking-tight text-slate-900 sm:text-4xl">
              {showHistory ? `${doc.title} — History of changes` : doc.title}
            </h1>
            {!showHistory ? (
              <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm text-slate-600">
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

          <MarkdownBody
            html={html}
            className="prose-content legal-prose [&_h2]:scroll-mt-28 [&_h2]:border-b [&_h2]:border-slate-200 [&_h2]:pb-2 [&_table]:w-full [&_table]:border-collapse [&_td]:border [&_td]:border-slate-200 [&_td]:p-2 [&_th]:border [&_th]:border-slate-200 [&_th]:bg-slate-50 [&_th]:p-2"
          />
        </article>
      </div>

      <SiteFooter />
    </div>
  )
}
