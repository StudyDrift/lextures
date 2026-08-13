import { useEffect, useMemo, useState } from 'react'
import { useSsrData } from '../lib/ssr-context'
import { DEFAULT_LOCALE, getLocale, localeFromPath, type LocaleCode } from '../lib/locales'
import { localeTargetsForPath, resolveRouteDescriptor } from '../lib/route-manifest'
import { trackEvent } from '../lib/analytics'

const COOKIE = 'lextures_locale'

function currentPath(ssrPath?: string): string {
  return ssrPath || (typeof window === 'undefined' ? '/' : window.location.pathname)
}

function contentLocaleTargets(article: { availableLocales?: Array<{ locale: string; path: string }> } | null | undefined) {
  const alts = article?.availableLocales
  if (!alts || alts.length < 2) return null
  return alts.map(alt => {
    const locale = getLocale(alt.locale)
    return { locale: locale.code as LocaleCode, name: locale.name, href: alt.path, exact: true }
  })
}

export function LocaleSwitcher({ compact = false }: { compact?: boolean }) {
  const { path: ssrPath, article } = useSsrData()
  const pathname = currentPath(ssrPath)
  const active = localeFromPath(pathname)
  const targets = contentLocaleTargets(article) ?? localeTargetsForPath(pathname)
  if (targets.length < 2) return null

  return (
    <label className={compact ? 'text-[13px]' : 'text-[14px]'}>
      <span className="sr-only">Language and region</span>
      <select
        aria-label="Language and region"
        value={active.code}
        onChange={event => {
          const target = targets.find(option => option.locale === event.currentTarget.value)
          if (!target || typeof window === 'undefined') return
          document.cookie = `${COOKIE}=${encodeURIComponent(target.locale)}; Path=/; Max-Age=31536000; SameSite=Lax; Secure`
          trackEvent('locale_switch', { from_locale: active.code, to_locale: target.locale, exact_page: target.exact })
          window.location.assign(target.href)
        }}
        className="rounded border px-2 py-1.5"
        style={{ background: 'var(--panel)', borderColor: 'var(--line)', color: 'inherit' }}
      >
        {targets.map(target => (
          <option key={target.locale} value={target.locale} lang={target.locale}>
            {target.name}{target.exact ? '' : ' — homepage'}
          </option>
        ))}
      </select>
    </label>
  )
}

export function LocaleSuggestion() {
  const { path: ssrPath, article } = useSsrData()
  const pathname = currentPath(ssrPath)
  const targets = useMemo(() => contentLocaleTargets(article) ?? localeTargetsForPath(pathname), [article, pathname])
  const [suggested, setSuggested] = useState<(typeof targets)[number] | null>(null)

  useEffect(() => {
    if (targets.length < 2 || document.cookie.includes(`${COOKIE}=`)) return
    const preferred = navigator.languages
      .map(language => getLocale(language))
      .find(locale => locale.code !== DEFAULT_LOCALE && targets.some(target => target.locale === locale.code))
    if (preferred) setSuggested(targets.find(target => target.locale === preferred.code) ?? null)
  }, [targets])

  if (!suggested) return null
  return (
    <aside aria-label="Language suggestion" className="border-b px-5 py-3 text-center text-sm" style={{ borderColor: 'var(--line)' }}>
      View this site in <a href={suggested.href} lang={suggested.locale}>{suggested.name}</a>?
      {' '}
      <button type="button" onClick={() => {
        document.cookie = `${COOKIE}=dismissed; Path=/; Max-Age=2592000; SameSite=Lax; Secure`
        setSuggested(null)
      }} className="underline">No thanks</button>
    </aside>
  )
}

export function TranslationStaleNotice() {
  const { path: ssrPath } = useSsrData()
  const pathname = currentPath(ssrPath)
  const route = resolveRouteDescriptor(pathname)?.descriptor
  if (!route?.translationOf || route.translationStatus !== 'stale' || !route.sourceUpdatedAt) return null
  const staleFor = Date.now() - new Date(route.sourceUpdatedAt).getTime()
  if (!Number.isFinite(staleFor) || staleFor < 90 * 24 * 60 * 60 * 1000) return null
  return (
    <aside role="status" className="border-b px-5 py-3 text-center text-sm" style={{ borderColor: 'var(--line)' }}>
      The English version was updated. <a href={route.translationOf}>View the current English page.</a>
    </aside>
  )
}
