import { useMemo } from 'react'
import { usePlatformFeatures } from '../context/platform-features-context'
import { documentLocaleFromCode } from './apply-document-locale'
import { readStoredLocaleTag } from './locale-storage'

/** Document text direction for the active locale and platform RTL flag (plan 11.2). */
export function useDocumentDirection(): 'ltr' | 'rtl' {
  const { rtlEnabled } = usePlatformFeatures()
  return useMemo(() => {
    const tag = readStoredLocaleTag() ?? 'en'
    return documentLocaleFromCode(tag, rtlEnabled).dir
  }, [rtlEnabled])
}
