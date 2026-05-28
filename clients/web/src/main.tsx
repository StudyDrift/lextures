import { registerServiceWorker } from './lib/push-notifications'

void registerServiceWorker()

import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import { BrowserRouter } from 'react-router-dom'
import './index.css'
import App from './app'
import { LmsToaster } from './components/lms-toaster'
import { AriaAnnouncer } from './components/aria-announcer'
import { LocaleFormatProvider } from './context/locale-format-context'
import { OrgBrandingProvider } from './context/org-branding-context'
import { PermissionsProvider } from './context/permissions-provider'

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <BrowserRouter>
      <LocaleFormatProvider>
      <OrgBrandingProvider>
        <PermissionsProvider>
          <AriaAnnouncer />
          <App />
          <LmsToaster />
        </PermissionsProvider>
      </OrgBrandingProvider>
      </LocaleFormatProvider>
    </BrowserRouter>
  </StrictMode>,
)
