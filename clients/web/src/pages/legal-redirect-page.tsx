import { useEffect } from 'react'
import { MARKETING_SITE_URLS } from '../lib/marketing-site'

type LegalRedirectPageProps = {
  document: 'terms' | 'privacy'
}

/**
 * Public `/terms` and `/privacy` on the app origin send visitors to the canonical
 * documents on the marketing site instead of bouncing signed-out users to `/login`.
 */
export default function LegalRedirectPage({ document }: LegalRedirectPageProps) {
  const href = MARKETING_SITE_URLS[document]
  useEffect(() => {
    window.location.replace(href)
  }, [href])
  return (
    <main className="flex min-h-dvh items-center justify-center px-4 text-sm text-stone-600 dark:text-fg-muted">
      <p>
        Redirecting to{' '}
        <a href={href} className="underline">
          {href}
        </a>
        …
      </p>
    </main>
  )
}
