import { createElement, StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BlogPost } from '../pages/blog-post'
import { DocsPost } from '../pages/docs-post'
import { canonicalUrl } from './site-origin'
import { SsrDataProvider } from './ssr-context'
import { fetchPublishedArticle, parseContentPath, previewTokenFromSearch, type ContentPath } from './public-content'
import type { ContentArticle } from './content-source'

function applyArticleHead(article: ContentArticle): void {
  document.title = `${article.title} — Lextures`
  document.documentElement.lang = article.locale || 'en'
  const description = document.querySelector('meta[name="description"]')
  if (description) description.setAttribute('content', article.description || article.title)
  const robots = document.querySelector('meta[name="robots"]')
  if (robots) {
    robots.setAttribute('content', article.noindex ? 'noindex,follow' : 'index,follow')
  }
  const canonical = document.querySelector('link[rel="canonical"]')
  if (canonical) canonical.setAttribute('href', article.canonicalOverride || canonicalUrl(article.path || `/blog/${article.slug}`))
}

function showLoading(): void {
  const heading = document.querySelector('main h1')
  if (heading && /page not found/i.test(heading.textContent || '')) {
    heading.textContent = 'Loading article'
    const copy = heading.parentElement?.querySelector('p')
    if (copy) copy.textContent = 'This published article is loading from the live catalog.'
  }
}

export async function renderLiveArticle(ref: ContentPath, root: HTMLElement): Promise<boolean> {
  const token = typeof window === 'undefined' ? undefined : previewTokenFromSearch(window.location.search)
  const article = await fetchPublishedArticle(ref, token)
  if (!article) return false
  applyArticleHead(article)
  const page =
    ref.kind === 'blog'
      ? createElement(BlogPost, { slug: article.slug })
      : createElement(DocsPost, { category: article.category || ref.category || '', slug: article.slug })
  createRoot(root).render(
    createElement(
      StrictMode,
      null,
      createElement(SsrDataProvider, { data: { article, path: article.path }, children: page }),
    ),
  )
  return true
}

export async function initLiveArticle(): Promise<void> {
  if (typeof document === 'undefined') return
  if (document.querySelector('[data-blog-post-page], [data-docs-post-page]')) return
  const ref = parseContentPath(window.location.pathname)
  if (!ref) return
  const root = document.getElementById('root')
  if (!root) return
  showLoading()
  await renderLiveArticle(ref, root)
}
