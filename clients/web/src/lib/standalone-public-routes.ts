const STANDALONE_PUBLIC_ROUTE_PREFIXES = [
  '/login',
  '/signup',
  '/forgot-password',
  '/reset-password',
  '/saml-callback',
  '/sso-error',
  '/ai-disclosure',
  '/trust',
  '/p',
  '/paths',
  '/verify',
  '/explore',
] as const

export function isStandalonePublicRoute(pathname: string): boolean {
  return STANDALONE_PUBLIC_ROUTE_PREFIXES.some(
    (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
  )
}

export function applyDocumentScrollMode(pathname: string): void {
  document.documentElement.style.overflow = isStandalonePublicRoute(pathname) ? 'auto' : 'hidden'
}
