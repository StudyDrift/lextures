import { APP_ORIGIN, SITE_LINKS } from '../lib/site-links'

const linkClass = 'font-medium text-accent no-underline underline-offset-2 hover:underline'
const mutedClass = 'text-stone-600 no-underline transition-colors hover:text-stone-900'

export function LegalNav() {
  return (
    <nav aria-label="Legal" className="mb-6 flex flex-wrap gap-3 text-sm">
      <a href={SITE_LINKS.privacy} className={linkClass}>
        Privacy
      </a>
      <a href={SITE_LINKS.californiaPrivacyRights} className={linkClass}>
        California Privacy
      </a>
      <a href={SITE_LINKS.terms} className={linkClass}>
        Terms
      </a>
      <a href={SITE_LINKS.security} className={linkClass}>
        Security
      </a>
      <a href={SITE_LINKS.accessibility} className={linkClass}>
        Accessibility
      </a>
      <a href={`${APP_ORIGIN}/trust`} className={mutedClass}>
        Trust Center
      </a>
      <a href={APP_ORIGIN} className={mutedClass}>
        Sign in
      </a>
    </nav>
  )
}
