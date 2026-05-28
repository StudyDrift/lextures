import { i18n } from '../i18n'
import { applyDocumentLocale } from '../i18n/apply-document-locale'
import { writeStoredLocaleTag } from '../i18n/locale-storage'
import { resolveResourceLanguage } from '../i18n/supported-locales'

function readRtlEnabledFlag(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return window.localStorage.getItem('lextures.rtlEnabled') === '1'
  } catch {
    return false
  }
}

/** Applies profile locale after login or account load (plan FR-4 priority #1). */
export async function syncUserLocale(tag: string | null | undefined): Promise<void> {
  const trimmed = tag?.trim()
  if (!trimmed) return
  writeStoredLocaleTag(trimmed)
  applyDocumentLocale(trimmed, readRtlEnabledFlag())
  await i18n.changeLanguage(resolveResourceLanguage(trimmed))
}
