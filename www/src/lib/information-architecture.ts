import { ROUTE_MANIFEST, resolveRouteDescriptor, resolveTitle, type ConcreteRoute } from './route-manifest'
import { SITE_ORIGIN } from './site-origin'

export const WWW_MESSAGES = {
  'www.nav.platform': 'Platform', 'www.nav.solutions': 'Solutions',
  'www.nav.resources': 'Resources', 'www.nav.pricing': 'Pricing', 'www.nav.docs': 'Docs',
  'www.breadcrumb.home': 'Home', 'www.related.heading': 'Related resources',
} as const

export function routeLabel(pathname: string): string {
  if (pathname === '/') return WWW_MESSAGES['www.breadcrumb.home']
  const resolved = resolveRouteDescriptor(pathname)
  if (!resolved) return pathname.split('/').filter(Boolean).at(-1)?.replace(/-/g, ' ') ?? pathname
  return resolveTitle(resolved.descriptor, { path: pathname, origin: SITE_ORIGIN, params: resolved.params })
    .replace(/\s+[—|-]\s+Lextures$/, '')
}

export function breadcrumbPaths(pathname: string): string[] {
  if (pathname === '/') return []
  const trail: string[] = []
  const seen = new Set<string>()
  let current: string | undefined = pathname
  while (current && current !== '/') {
    if (seen.has(current)) throw new Error(`IA parent cycle at ${current}`)
    seen.add(current)
    trail.unshift(current)
    const resolved = resolveRouteDescriptor(current)
    current = resolved?.descriptor.parent ?? '/'
  }
  return ['/', ...trail]
}

export function breadcrumbJsonLd(pathname: string) {
  const paths = breadcrumbPaths(pathname)
  if (!paths.length) return null
  return {
    '@id': `${SITE_ORIGIN}${pathname}#breadcrumb`,
    '@type': 'BreadcrumbList',
    itemListElement: paths.map((path, index) => ({
      '@type': 'ListItem', position: index + 1, name: routeLabel(path),
      item: `${SITE_ORIGIN}${path === '/' ? '/' : path}`,
    })),
  }
}

export function childRoutes(parent: string) {
  return ROUTE_MANIFEST.filter(route => route.parent === parent && !route.path.includes(':'))
}

export function relatedRoutes(pathname: string, minimum = 3, maximum = 6): Array<{ path: string; label: string }> {
  const current = resolveRouteDescriptor(pathname)?.descriptor
  if (!current) return []
  const explicit = current.relatedTo ?? []
  const candidates = ROUTE_MANIFEST.filter(route =>
    !route.path.includes(':') && route.path !== pathname && route.sitemap &&
    (explicit.includes(route.path) || (current.cluster && route.cluster === current.cluster) || route.path === current.parent),
  )
  const fallback = ['/platform', '/resources', '/pricing', '/docs', '/get-started']
    .map(path => ROUTE_MANIFEST.find(route => route.path === path))
    .filter((route): route is NonNullable<typeof route> => Boolean(route) && route!.path !== pathname)
  const ordered = [...candidates.sort((a, b) => {
    const ai = explicit.indexOf(a.path); const bi = explicit.indexOf(b.path)
    if (ai >= 0 || bi >= 0) return (ai < 0 ? 999 : ai) - (bi < 0 ? 999 : bi)
    return a.path.localeCompare(b.path)
  }), ...fallback]
  return [...new Map(ordered.map(route => [route.path, { path: route.path, label: routeLabel(route.path) }])).values()]
    .slice(0, Math.max(minimum, maximum))
}

export function concretePathSet(routes: ConcreteRoute[]): Set<string> {
  return new Set(routes.filter(route => route.descriptor.sitemap).map(route => route.path))
}
