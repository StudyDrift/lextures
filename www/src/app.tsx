import { useEffect, useRef, useState, type ComponentType, type ReactElement } from 'react'
import {
  resolveRouteDescriptor,
  resolveDescription,
  resolveTitle,
  pagePropsFor,
  hreflangAlternatesForPath,
  type RenderContext,
} from './lib/route-manifest'
import { getPageLoader } from './lib/route-pages'
import { useDocumentHead } from './lib/use-document-head'
import { canonicalUrl, SITE_ORIGIN } from './lib/site-origin'
import { breadcrumbJsonLd } from './lib/information-architecture'
import { getLocale } from './lib/locales'

function useHashRoute(): string {
  const [hash, setHash] = useState(() =>
    typeof window !== 'undefined' ? window.location.hash : '',
  )
  useEffect(() => {
    const handler = () => setHash(window.location.hash)
    window.addEventListener('hashchange', handler)
    return () => window.removeEventListener('hashchange', handler)
  }, [])
  return hash
}

/** Resolve pathname + legacy hash routes (`#/docs/...`) to a clean path. */
export function resolveRoute(pathname: string, hash: string): string {
  if (hash.startsWith('#/')) {
    const hashRoute = hash.slice(1)
    if (pathname !== '/' && hashRoute.startsWith(`${pathname}/`)) {
      return hashRoute
    }
    if (pathname === '/') return hashRoute
  }
  if (pathname !== '/' && pathname.endsWith('/')) {
    return pathname.replace(/\/+$/, '') || '/'
  }
  return pathname !== '/' ? pathname : '/'
}

function liveRegionAnnounce(title: string): void {
  const el = document.getElementById('route-announcer')
  if (el) {
    el.textContent = title
  }
}

function focusMainHeading(): void {
  const h1 = document.querySelector('main h1, h1') as HTMLElement | null
  if (!h1) return
  if (!h1.hasAttribute('tabindex')) {
    h1.setAttribute('tabindex', '-1')
  }
  h1.focus({ preventScroll: false })
}

type AppProps = {
  /** Pathname for SSG / tests. Defaults to `window.location` in the browser. */
  url?: string
  /**
   * Preloaded page element for SSR (avoids async in renderToString).
   * When set, client still hydrates with the same tree on first paint.
   */
  ssrPage?: ReactElement | null
}

/**
 * Manifest-driven router (SEO.1 FR-2). Client navigations use full page loads
 * via real `<a href>` links; path still drives head sync after hydrate.
 * Pages are loaded via dynamic import() (SEO.4 FR-3).
 */
export default function App({ url, ssrPage }: AppProps = {}): ReactElement {
  const hash = useHashRoute()
  const initialPath =
    url ??
    (typeof window !== 'undefined'
      ? resolveRoute(window.location.pathname, hash)
      : '/')
  const [route, setRoute] = useState(initialPath)
  const prevRoute = useRef(route)
  const [Page, setPage] = useState<ComponentType<Record<string, unknown>> | null>(null)
  const [pageReady, setPageReady] = useState(Boolean(ssrPage))

  // Keep route in sync with browser navigation (back/forward, hash).
  useEffect(() => {
    if (url != null) return
    const sync = () => {
      setRoute(resolveRoute(window.location.pathname, window.location.hash))
    }
    window.addEventListener('popstate', sync)
    window.addEventListener('hashchange', sync)
    sync()
    return () => {
      window.removeEventListener('popstate', sync)
      window.removeEventListener('hashchange', sync)
    }
  }, [url, hash])

  // When url prop is provided (SSG), always use it.
  useEffect(() => {
    if (url != null) setRoute(resolveRoute(url, ''))
  }, [url])

  const resolved = resolveRouteDescriptor(route)
  const renderCtx: RenderContext = {
    path: route,
    origin: SITE_ORIGIN,
    params: resolved?.params ?? {},
  }

  // Lazy-load page module on client (and when SSR didn't pass ssrPage).
  useEffect(() => {
    if (ssrPage && route === initialPath) {
      // First paint hydrates from ssrPage; skip fetch until navigation.
      return
    }
    let cancelled = false
    const pattern = resolved?.descriptor.translationOf ?? resolved?.descriptor.path
    const loader = pattern ? getPageLoader(pattern) : getPageLoader('/404')
    setPageReady(false)
    void (loader ?? getPageLoader('/404'))!().then(C => {
      if (cancelled) return
      setPage(() => C)
      setPageReady(true)
    })
    return () => {
      cancelled = true
    }
  }, [route, resolved?.descriptor.path, resolved?.descriptor.translationOf, ssrPage, initialPath])

  const headTitle = resolved
    ? resolveTitle(resolved.descriptor, renderCtx)
    : 'Page not found — Lextures'
  const headDescription = resolved
    ? resolveDescription(resolved.descriptor, renderCtx)
    : 'The page you requested does not exist.'
  const headRobots = resolved?.descriptor.robots ?? (resolved ? 'index,follow' : 'noindex,follow')
  const markdownAlternate =
    (route.startsWith('/blog/') && route !== '/blog') ||
    (route.startsWith('/docs/') && route !== '/docs')
      ? `${SITE_ORIGIN}${route}.md`
      : null

  useDocumentHead({
    title: headTitle,
    description: headDescription,
    canonical: canonicalUrl(route === '/404' ? '/404' : route),
    robots: headRobots,
    image: resolved?.descriptor.ogImage,
    jsonLd: [
      ...(resolved?.descriptor.jsonLd?.(renderCtx) ?? []),
      ...(breadcrumbJsonLd(route) ? [breadcrumbJsonLd(route)!] : []),
    ],
    markdownAlternate,
    alternates: hreflangAlternatesForPath(route),
    locale: resolved?.descriptor.locale ?? 'en',
    dir: getLocale(resolved?.descriptor.locale).dir,
  })

  // Focus + live region after client-side path change (AC-8).
  useEffect(() => {
    if (prevRoute.current === route) return
    prevRoute.current = route
    if (typeof window === 'undefined' || typeof requestAnimationFrame !== 'function') return
    requestAnimationFrame(() => {
      liveRegionAnnounce(headTitle)
      focusMainHeading()
    })
  }, [route, headTitle])

  let body: ReactElement
  if (ssrPage && route === initialPath && !Page) {
    body = ssrPage
  } else if (Page) {
    const props = resolved
      ? pagePropsFor(resolved.descriptor.translationOf ?? resolved.descriptor.path, renderCtx)
      : {}
    body = <Page {...props} />
  } else if (!pageReady && !ssrPage) {
    // Client-only cold start without SSR markup (dev edge case).
    body = (
      <div className="mx-auto max-w-3xl px-4 py-24 text-center text-sm" style={{ color: 'var(--text-soft)' }}>
        Loading…
      </div>
    )
  } else {
    body = ssrPage ?? (
      <div className="mx-auto max-w-3xl px-4 py-24 text-center text-sm" style={{ color: 'var(--text-soft)' }}>
        Loading…
      </div>
    )
  }

  return (
    <>
      <div
        id="route-announcer"
        aria-live="polite"
        aria-atomic="true"
        className="sr-only"
      />
      {body}
    </>
  )
}
