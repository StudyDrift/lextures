import type { JsonLdNode } from '../document-head'
import { absoluteUrl, breadcrumbId } from './ids'

export type BreadcrumbItem = {
  name: string
  path: string
}

/** Human labels for path segments when building crumbs from a URL. */
const SEGMENT_LABELS: Record<string, string> = {
  about: 'About',
  authors: 'Authors',
  blog: 'Blog',
  docs: 'Documentation',
  courses: 'Courses',
  pricing: 'Pricing',
  calculator: 'Calculator',
  privacy: 'Privacy Policy',
  terms: 'Terms of Service',
  security: 'Security',
  accessibility: 'Accessibility',
  vpat: 'VPAT',
  history: 'History',
  'get-started': 'Get started',
  parents: 'Parents',
  'higher-ed': 'Higher education',
  'k-12': 'K–12',
  homeschool: 'Homeschool',
  'request-information': 'Request information',
  'privacy-rights': 'Privacy rights',
  california: 'California',
}

/**
 * Build breadcrumb items for a path (FR-10). Homepage has no breadcrumb list.
 * Visible UI breadcrumbs land with SEO.5; schema mirrors the trail structure.
 */
export function breadcrumbItemsForPath(
  path: string,
  opts?: { leafName?: string },
): BreadcrumbItem[] {
  if (!path || path === '/') return []
  const clean = path.replace(/\/+$/, '')
  const parts = clean.split('/').filter(Boolean)
  const items: BreadcrumbItem[] = [{ name: 'Home', path: '/' }]
  let acc = ''
  for (let i = 0; i < parts.length; i++) {
    const part = parts[i]
    acc += `/${part}`
    const isLeaf = i === parts.length - 1
    const name =
      isLeaf && opts?.leafName
        ? opts.leafName
        : SEGMENT_LABELS[part] ||
          part
            .split('-')
            .map(w => w.charAt(0).toUpperCase() + w.slice(1))
            .join(' ')
    items.push({ name, path: acc })
  }
  return items
}

export function buildBreadcrumbList(
  path: string,
  opts?: { leafName?: string; siteOrigin?: string },
): JsonLdNode | null {
  if (!path || path === '/') return null
  const items = breadcrumbItemsForPath(path, { leafName: opts?.leafName })
  if (items.length < 2) return null
  return {
    '@type': 'BreadcrumbList',
    '@id': breadcrumbId(path, opts?.siteOrigin),
    itemListElement: items.map((item, i) => ({
      '@type': 'ListItem',
      position: i + 1,
      name: item.name,
      item: absoluteUrl(item.path, opts?.siteOrigin),
    })),
  }
}
