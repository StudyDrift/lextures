import i18n from 'i18next'
import HttpBackend from 'i18next-http-backend'
import { IcuFormatPlugin } from './icu-format-plugin'
import { initReactI18next } from 'react-i18next'
import { applyDocumentLocale } from './apply-document-locale'
import { detectInitialLocale, readStoredLocaleTag } from './locale-storage'
import { recordMissingTranslationKey } from './missing-key'
import { DEFAULT_LOCALE, SUPPORTED_LOCALES } from './supported-locales'
import enAuth from '../../public/locales/en/auth.json'
import enCommon from '../../public/locales/en/common.json'
import enCompliance from '../../public/locales/en/compliance.json'
import enParent from '../../public/locales/en/parent.json'
import esAuth from '../../public/locales/es/auth.json'
import esCommon from '../../public/locales/es/common.json'
import esCompliance from '../../public/locales/es/compliance.json'
import esParent from '../../public/locales/es/parent.json'
import frAuth from '../../public/locales/fr/auth.json'
import frCommon from '../../public/locales/fr/common.json'
import frCompliance from '../../public/locales/fr/compliance.json'
import frParent from '../../public/locales/fr/parent.json'

export const I18N_NAMESPACES = ['common', 'auth', 'compliance', 'parent'] as const
const bundledResources = {
  en: { common: enCommon, auth: enAuth, compliance: enCompliance, parent: enParent },
  es: { common: esCommon, auth: esAuth, compliance: esCompliance, parent: esParent },
  fr: { common: frCommon, auth: frAuth, compliance: frCompliance, parent: frParent },
} as const

const useHttpBackend = !import.meta.env.VITEST

const initialLng = detectInitialLocale()

const instance = i18n.use(new IcuFormatPlugin()).use(initReactI18next)
if (useHttpBackend) {
  instance.use(HttpBackend)
}

void instance.init({
  lng: initialLng,
  fallbackLng: DEFAULT_LOCALE,
  supportedLngs: [...SUPPORTED_LOCALES],
  ns: [...I18N_NAMESPACES],
  defaultNS: 'common',
  keySeparator: false,
  nsSeparator: false,
  ...(useHttpBackend
    ? {
        backend: {
          loadPath: '/locales/{{lng}}/{{ns}}.json',
        },
      }
    : {
        resources: bundledResources,
      }),
  interpolation: {
    escapeValue: false,
  },
  react: {
    useSuspense: false,
  },
  saveMissing: false,
  missingKeyHandler(lngs, ns, key) {
    const locale = lngs[0] ?? DEFAULT_LOCALE
    if (locale === DEFAULT_LOCALE) return
    recordMissingTranslationKey({ locale, namespace: ns, key })
  },
})

function readRtlEnabledFlag(): boolean {
  if (typeof window === 'undefined') return false
  try {
    return window.localStorage.getItem('lextures.rtlEnabled') === '1'
  } catch {
    return false
  }
}

const initialTag = readStoredLocaleTag() ?? initialLng
applyDocumentLocale(initialTag, readRtlEnabledFlag())

i18n.on('languageChanged', (lng) => {
  const tag = readStoredLocaleTag() ?? lng
  applyDocumentLocale(tag, readRtlEnabledFlag())
})

export { i18n }
