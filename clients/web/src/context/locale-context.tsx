/* eslint-disable react-refresh/only-export-components -- context module exports provider + hooks */
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from 'react'
import { authorizedFetch } from '../lib/api'
import { applyDocumentLocale, readStoredLocale } from '../i18n/document-locale'
import { normalizeLocaleCode } from '../i18n/supported-locales'

export type LocaleContextValue = {
  locale: string
  dir: 'ltr' | 'rtl'
  rtlEnabled: boolean
  loading: boolean
  setLocale: (locale: string) => Promise<void>
  refresh: () => Promise<void>
}

const LocaleContext = createContext<LocaleContextValue | null>(null)

type LocaleProviderProps = {
  children: ReactNode
  rtlEnabled: boolean
}

export function LocaleProvider({ children, rtlEnabled }: LocaleProviderProps) {
  const [locale, setLocaleState] = useState(() => readStoredLocale())
  const [loading, setLoading] = useState(true)

  const apply = useCallback(
    (code: string) => {
      const normalized = normalizeLocaleCode(code)
      const state = applyDocumentLocale(normalized, rtlEnabled)
      setLocaleState(state.locale)
      return state
    },
    [rtlEnabled],
  )

  const refresh = useCallback(async () => {
    setLoading(true)
    try {
      const res = await authorizedFetch('/api/v1/settings/account')
      const raw: unknown = await res.json().catch(() => ({}))
      if (res.ok) {
        const data = raw as { locale?: string }
        apply(data.locale ?? readStoredLocale())
        return
      }
    } catch {
      apply(readStoredLocale())
    } finally {
      setLoading(false)
    }
  }, [apply])

  useEffect(() => {
    apply(readStoredLocale())
    void refresh()
  }, [apply, refresh])

  useEffect(() => {
    apply(locale)
  }, [apply, locale, rtlEnabled])

  const setLocale = useCallback(
    async (next: string) => {
      const normalized = normalizeLocaleCode(next)
      const res = await authorizedFetch('/api/v1/settings/account', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ locale: normalized }),
      })
      if (!res.ok) {
        const raw: unknown = await res.json().catch(() => ({}))
        const err = raw as { error?: { message?: string } }
        throw new Error(err.error?.message ?? 'Failed to update locale.')
      }
      const state = apply(normalized)
      window.dispatchEvent(new CustomEvent('lextures-locale-updated', { detail: state }))
    },
    [apply],
  )

  const dir = useMemo(
    () => applyDocumentLocale(locale, rtlEnabled).dir,
    [locale, rtlEnabled],
  )

  const value = useMemo(
    () => ({
      locale,
      dir,
      rtlEnabled,
      loading,
      setLocale,
      refresh,
    }),
    [locale, dir, rtlEnabled, loading, setLocale, refresh],
  )

  return <LocaleContext.Provider value={value}>{children}</LocaleContext.Provider>
}

export function useLocale(): LocaleContextValue {
  const ctx = useContext(LocaleContext)
  if (!ctx) {
    throw new Error('useLocale must be used within LocaleProvider')
  }
  return ctx
}
