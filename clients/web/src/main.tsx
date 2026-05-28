import { registerServiceWorker } from './lib/push-notifications'
import './i18n'

void registerServiceWorker()

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './app'
import { LmsToaster } from './components/lms-toaster'
import { AriaAnnouncer } from './components/aria-announcer'
import { OrgBrandingProvider } from './context/org-branding-context'
import { PermissionsProvider } from './context/permissions-provider'
import { I18nProvider } from './context/i18n-provider'
import { UserTimezoneProvider } from './context/user-timezone-provider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <I18nProvider>
      <OrgBrandingProvider>
        <UserTimezoneProvider>
        <PermissionsProvider>
          <AriaAnnouncer />
          <App />
          <LmsToaster />
        </PermissionsProvider>
        </UserTimezoneProvider>
      </OrgBrandingProvider>
      </I18nProvider>
    </BrowserRouter>
  </StrictMode>,
)
