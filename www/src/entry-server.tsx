/**
 * Server/SSG entry: render a path to an HTML string (SEO.1 FR-4).
 * Page modules load via dynamic import for parity with the client graph (SEO.4 FR-3).
 */
import { renderToString } from 'react-dom/server'
import App from './app'
import { SsrDataProvider } from './lib/ssr-context'
import type { SsrData } from './lib/ssr-data'
import {
  resolveRouteDescriptor,
  resolveTitle,
  resolveDescription,
  loadRouteElement,
  type ConcreteRoute,
  enumerateConcreteRoutes,
  isInteractiveRoute,
  hreflangAlternatesForPath,
  validateLocaleManifest,
} from './lib/route-manifest'
import {
  buildPrerenderHeadTags,
  truncateMetaDescription,
  truncateTitle,
  DEFAULT_OG_IMAGE,
  type DocumentHeadOptions,
} from './lib/document-head'
import { canonicalUrl, SITE_ORIGIN } from './lib/site-origin'
import type { PublicMarketplaceCourse, PublicMarketplaceCourseDetail } from './lib/marketplace-api'
import { resolveApiAssetUrl } from './lib/api-base'
import { courseDetailGraph, coursesIndexGraph } from './lib/schema/page-graphs'
import { breadcrumbJsonLd } from './lib/information-architecture'
import { getLocale } from './lib/locales'

export type RenderResult = {
  bodyHtml: string
  headTags: string
  head: DocumentHeadOptions
  path: string
  interactive: boolean
}

export function buildHeadForRoute(
  route: Pick<ConcreteRoute, 'path' | 'title' | 'description' | 'robots' | 'canonical'> & {
    params?: Record<string, string>
  },
  opts?: {
    courseDetail?: PublicMarketplaceCourseDetail | null
    coursesIndex?: { courses: PublicMarketplaceCourse[]; total: number } | null
    image?: string
    jsonLd?: DocumentHeadOptions['jsonLd']
  },
): DocumentHeadOptions {
  const resolved = resolveRouteDescriptor(route.path)
  let title = route.title
  let description = route.description
  let image = opts?.image
  let jsonLd = opts?.jsonLd ?? null

  if (opts?.courseDetail?.course) {
    const c = opts.courseDetail.course
    const courseKind = [c.level, c.category].filter(Boolean).join(' ')
    title = truncateTitle(`${c.title} — ${courseKind ? `${courseKind} course` : 'online course'} | Lextures`)
    description = truncateMetaDescription(c.description || description)
    image = resolveApiAssetUrl(c.heroImageUrl) || image
    const ctx = {
      path: route.path,
      origin: SITE_ORIGIN,
      params: route.params ?? { slug: c.slug },
    }
    jsonLd = courseDetailGraph(ctx, opts.courseDetail)
  } else if (resolved) {
    const ctx = {
      path: route.path,
      origin: SITE_ORIGIN,
      params: route.params ?? resolved.params,
    }
    title = resolveTitle(resolved.descriptor, ctx)
    description = resolveDescription(resolved.descriptor, ctx)
    if (route.path.startsWith('/courses') && opts?.coursesIndex?.courses?.length) {
      jsonLd = coursesIndexGraph(ctx, opts.coursesIndex.courses)
    } else if (resolved.descriptor.jsonLd) {
      jsonLd = resolved.descriptor.jsonLd(ctx)
    }
    image = resolved.descriptor.ogImage ?? image
  }

  const locale = resolved?.descriptor.locale ?? 'en'
  const languageTypes = new Set(['WebPage', 'Article', 'TechArticle', 'FAQPage', 'HowTo', 'Course'])
  const nodes = Array.isArray(jsonLd) ? jsonLd : jsonLd ? [jsonLd] : []
  jsonLd = nodes.map(node => {
    const type = node?.['@type']
    const types = Array.isArray(type) ? type : [type]
    return types.some(value => languageTypes.has(String(value))) ? { ...node, inLanguage: locale } : node
  })

  return {
    title,
    description,
    canonical: route.canonical || canonicalUrl(route.path),
    robots: route.robots || 'index,follow',
    image: image || DEFAULT_OG_IMAGE,
    jsonLd,
    locale,
    dir: getLocale(locale).dir,
    alternates: hreflangAlternatesForPath(route.path),
  }
}

export async function renderPath(pathname: string, ssrData: SsrData = {}): Promise<RenderResult> {
  const path =
    pathname !== '/' && pathname.endsWith('/') ? pathname.replace(/\/+$/, '') || '/' : pathname

  const ssrPage = await loadRouteElement(path)
  const interactive = isInteractiveRoute(path)

  const bodyHtml = renderToString(
    <SsrDataProvider data={{ ...ssrData, path }}>
      <App url={path} ssrPage={ssrPage} />
    </SsrDataProvider>,
  )

  const resolved = resolveRouteDescriptor(path)
  const courseDetail =
    path.startsWith('/courses/') && path !== '/courses' ? ssrData.courseDetail : undefined
  const coursesIndex = path === '/courses' || path.startsWith('/courses/subject/') || path.startsWith('/courses/level/')
    ? ssrData.coursesIndex
    : undefined

  const head = buildHeadForRoute(
    {
      path,
      title: 'Lextures',
      description: '',
      robots: resolved?.descriptor.robots ?? (resolved ? 'index,follow' : 'noindex,follow'),
      canonical: canonicalUrl(path),
      params: resolved?.params,
    },
    {
      courseDetail: courseDetail ?? null,
      coursesIndex: coursesIndex ?? null,
    },
  )
  const breadcrumb = breadcrumbJsonLd(path)
  if (breadcrumb) head.jsonLd = [...(Array.isArray(head.jsonLd) ? head.jsonLd : head.jsonLd ? [head.jsonLd] : []), breadcrumb]

  return {
    bodyHtml,
    headTags: buildPrerenderHeadTags(head),
    head,
    path,
    interactive,
  }
}

export { SITE_ORIGIN, enumerateConcreteRoutes, resolveRouteDescriptor, isInteractiveRoute, validateLocaleManifest }
