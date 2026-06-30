import type { ReactNode } from 'react'
import { lazy, Suspense } from 'react'

const MaintenanceStatusBanner = lazy(() =>
  import('../StatusBanner').then((m) => ({ default: m.StatusBanner })),
)

type PublicAuthShellProps = {
  children: ReactNode
  orgSlug?: string | null
}

/**
 * Layout wrapper for public sign-in flows: warm neutral backdrop, no decorative “hero” gradient.
 */
export function PublicAuthShell({ children, orgSlug = null }: PublicAuthShellProps) {
  return (
    <div className="lex-auth-scene min-h-dvh text-stone-900">
      <Suspense fallback={null}>
        <MaintenanceStatusBanner orgSlug={orgSlug} />
      </Suspense>
      <main className="flex min-h-dvh flex-col items-center justify-center px-4 py-12 sm:px-6 sm:py-16">
        <div className="relative z-10 w-full max-w-md">{children}</div>
      </main>
    </div>
  )
}
