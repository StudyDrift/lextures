import { parseContentPath } from './public-content-core'

/** Load live catalog data only on content surfaces that need it. */
export function initLiveContent(): void {
  if (typeof document === 'undefined') return
  if (document.querySelector('[data-blog-index]')) {
    void import('./live-blog-index').then(mod => mod.initLiveBlogIndex())
  }
  if (document.querySelector('[data-docs-category]')) {
    void import('./live-docs-category').then(mod => mod.initLiveDocsCategory())
  }
  const alreadyRendered = document.querySelector('[data-blog-post-page], [data-docs-post-page]')
  if (!alreadyRendered && parseContentPath(window.location.pathname)) {
    void import('./live-article').then(mod => mod.initLiveArticle())
  }
}
