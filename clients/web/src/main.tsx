import { registerServiceWorker } from './lib/push-notifications'
import './i18n'
import { applyUiTheme, readStoredUiTheme } from './lib/ui-theme'

applyUiTheme(readStoredUiTheme())

void registerServiceWorker()

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './app'
import { LmsToaster } from './components/lms-toaster'
import { AriaAnnouncer } from './components/aria-announcer'
import { I18nProvider } from './context/i18n-provider'
import { LocaleFormatProvider } from './context/locale-format-context'
import { OrgBrandingProvider } from './context/org-branding-context'
import { PermissionsProvider } from './context/permissions-provider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <I18nProvider>
        <LocaleFormatProvider>
          <OrgBrandingProvider>
            <PermissionsProvider>
              <AriaAnnouncer />
              <App />
              <LmsToaster />
            </PermissionsProvider>
          </OrgBrandingProvider>
        </LocaleFormatProvider>
      </I18nProvider>
    </BrowserRouter>
  </StrictMode>,
)
