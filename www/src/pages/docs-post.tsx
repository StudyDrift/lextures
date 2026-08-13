import { ArrowLeft } from 'lucide-react'
import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { Byline } from '../components/byline'
import { MarkdownBody } from '../components/markdown-body'
import { articlePath, formatDate, getCategorizedArticle } from '../utils/docs'
import { getHelpCategory } from '../docs/_categories'
import { RelatedContent } from '../components/related-content'
import { ContextualLinks } from '../components/contextual-links'
import { useSsrData } from '../lib/ssr-context'

export function DocsPost({ category: categoryId, slug }: { category: string; slug: string }) {
  const ssr = useSsrData()
  // Prefer the normalized content-source article (string author, html, dates).
  // Raw SSR payloads embed API author objects and bodyMd only.
  const article = getCategorizedArticle(categoryId, slug) ?? (ssr.article?.kind === 'doc' ? ssr.article : undefined)
  const category = getHelpCategory(categoryId)

  if (!article) {
    return (
      <div className="relative min-h-screen bg-white text-slate-900">
        <Header />
        <main className="mx-auto max-w-3xl px-4 py-24 sm:px-6 lg:px-8">
          <p className="text-slate-500">Article not found.</p>
          <a href="/docs" className="btn-secondary mt-6 inline-flex gap-2">
            <ArrowLeft className="h-4 w-4" aria-hidden />
            Back to documentation
          </a>
        </main>
      </div>
    )
  }

  return (
    <div className="relative min-h-screen overflow-x-hidden bg-white text-slate-900">
      <Header />

      <main>
        <div className="border-b border-slate-200 bg-white py-12 sm:py-16">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <a
              href={category ? `/docs/${category.id}` : '/docs'}
              className="inline-flex items-center gap-1.5 text-sm font-medium text-slate-500 no-underline transition-colors hover:text-slate-900"
            >
              <ArrowLeft className="h-3.5 w-3.5" aria-hidden />
              {category?.title || 'Help center'}
            </a>
            <time
              dateTime={article.date}
              className="mt-6 block text-xs font-medium uppercase tracking-widest text-slate-400"
            >
              {formatDate(article.date)}
            </time>
            <h1 className="font-display mt-3 text-3xl font-normal leading-tight tracking-tight text-slate-900 sm:text-4xl lg:text-[2.5rem]">
              {article.title}
            </h1>
            <p className="mt-4 text-lg leading-relaxed text-slate-600">{article.description}</p>
            <dl className="mt-6 flex flex-wrap gap-x-6 gap-y-2 text-sm text-slate-600">
              <div><dt className="inline font-semibold">Roles: </dt><dd className="inline">{article.roles.join(', ')}</dd></div>
              <div><dt className="inline font-semibold">Segments: </dt><dd className="inline">{article.segments.join(', ')}</dd></div>
              <div><dt className="inline font-semibold">Verified: </dt><dd className="inline">{article.verifiedAgainst}</dd></div>
            </dl>
          </div>
        </div>

        <div className="py-12 sm:py-16">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <article>
              <MarkdownBody html={article.html} className="prose-content" />
              <ContextualLinks kind="docs" />
              <Byline
                authorSlug={article.author}
                datePublished={article.date}
                dateModified={article.updated || article.date}
                reviewedBySlug={article.reviewedBy}
                className="mt-12"
              />
              <RelatedContent path={article.path || articlePath(article)} />
            </article>

            <div className="mt-16 border-t border-slate-200/80 pt-10">
              <a href={`/docs/${article.category}`} className="btn-secondary inline-flex gap-2">
                <ArrowLeft className="h-4 w-4" aria-hidden />
                Back to {category?.title || 'help center'}
              </a>
            </div>
          </div>
        </div>
      </main>

      <SiteFooter />
    </div>
  )
}
