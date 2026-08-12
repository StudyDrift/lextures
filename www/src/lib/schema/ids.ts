/**
 * Canonical schema.org @id constants (SEO.3 FR-2).
 * This is the only place absolute entity @ids are spelled.
 */
import { SITE_ORIGIN } from '../site-origin'

const origin = () => SITE_ORIGIN.replace(/\/$/, '')

export function organizationId(siteOrigin = origin()): string {
  return `${siteOrigin}/#organization`
}

export function websiteId(siteOrigin = origin()): string {
  return `${siteOrigin}/#website`
}

export function softwareApplicationId(siteOrigin = origin()): string {
  return `${siteOrigin}/#software`
}

export function founderPersonId(siteOrigin = origin()): string {
  return `${siteOrigin}/about#founder`
}

export function logoImageId(siteOrigin = origin()): string {
  return `${siteOrigin}/#logo`
}

export function authorPersonId(slug: string, siteOrigin = origin()): string {
  return `${siteOrigin}/authors/${slug}#person`
}

export function articleId(path: string, siteOrigin = origin()): string {
  const clean = path.startsWith('/') ? path : `/${path}`
  return `${siteOrigin}${clean.replace(/\/+$/, '')}#article`
}

export function webpageId(path: string, siteOrigin = origin()): string {
  const clean = path === '/' ? '/' : (path.startsWith('/') ? path : `/${path}`).replace(/\/+$/, '')
  return `${siteOrigin}${clean === '/' ? '' : clean}#webpage`
}

export function breadcrumbId(path: string, siteOrigin = origin()): string {
  const clean = path === '/' ? '/' : (path.startsWith('/') ? path : `/${path}`).replace(/\/+$/, '')
  return `${siteOrigin}${clean === '/' ? '' : clean}#breadcrumb`
}

export function faqId(path: string, siteOrigin = origin()): string {
  const clean = path.startsWith('/') ? path : `/${path}`
  return `${siteOrigin}${clean.replace(/\/+$/, '')}#faq`
}

export function courseId(slug: string, siteOrigin = origin()): string {
  return `${siteOrigin}/courses/${slug}#course`
}

export function productPricingId(siteOrigin = origin()): string {
  return `${siteOrigin}/pricing#product`
}

export function vpatCreativeWorkId(siteOrigin = origin()): string {
  return `${siteOrigin}/accessibility/vpat#vpat`
}

export function digitalDocumentId(path: string, siteOrigin = origin()): string {
  const clean = path.startsWith('/') ? path : `/${path}`
  return `${siteOrigin}${clean.replace(/\/+$/, '')}#document`
}

export function absoluteUrl(path: string, siteOrigin = origin()): string {
  if (!path || path === '/') return `${siteOrigin}/`
  const clean = path.startsWith('/') ? path : `/${path}`
  return `${siteOrigin}${clean.replace(/\/+$/, '')}`
}
