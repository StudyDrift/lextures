import type { ReactNode } from 'react'
import { I18nextProvider } from 'react-i18next'
import { i18n } from '../i18n'

type Props = {
  children: ReactNode
}

export function I18nProvider({ children }: Props) {
  return <I18nextProvider i18n={i18n}>{children}</I18nextProvider>
}
