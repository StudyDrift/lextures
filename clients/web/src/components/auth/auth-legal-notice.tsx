import { Trans } from 'react-i18next'
import { MARKETING_SITE_URLS } from '../../lib/marketing-site'

type AuthLegalNoticeProps = {
  /** `signup` shows account-creation consent text; `login` shows the sign-in notice. */
  variant: 'signup' | 'login'
}

const linkClass =
  'font-medium text-stone-700 underline underline-offset-2 hover:text-stone-900 dark:text-fg-default dark:hover:text-fg-default'

/**
 * Terms of Service and Privacy Policy consent text for the public sign-in and sign-up pages.
 * Links open the canonical documents on the marketing site.
 */
export function AuthLegalNotice({ variant }: AuthLegalNoticeProps) {
  /* Trans injects the translated link text as the anchor children. */
  /* oxlint-disable jsx-a11y/anchor-has-content */
  return (
    <p
      className="mt-6 text-center text-xs leading-relaxed text-stone-600 dark:text-fg-muted"
      data-testid="auth-legal-notice"
    >
      <Trans
        i18nKey={variant === 'signup' ? 'auth.legal.signupConsent' : 'auth.legal.loginNotice'}
        ns="auth"
        components={{
          termsLink: (
            <a href={MARKETING_SITE_URLS.terms} target="_blank" rel="noopener noreferrer" className={linkClass} />
          ),
          privacyLink: (
            <a href={MARKETING_SITE_URLS.privacy} target="_blank" rel="noopener noreferrer" className={linkClass} />
          ),
        }}
      />
    </p>
  )
}
