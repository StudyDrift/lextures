import { NavLink as RouterNavLink, type NavLinkProps } from 'react-router-dom'
import { cx, focusRingClass } from './utils'

export type UiNavLinkProps = NavLinkProps & {
  /** Visual weight for in-page / side nav. */
  variant?: 'default' | 'subtle'
}

/**
 * Styled React Router NavLink with focus ring and active styles from tokens.
 * Domain shells (side-nav) may still use local styles; this is the generic primitive.
 */
export function UiNavLink({
  className,
  variant = 'default',
  ...props
}: UiNavLinkProps) {
  return (
    <RouterNavLink
      {...props}
      className={(nav) => {
        const extra = typeof className === 'function' ? className(nav) : className
        return cx(
          'inline-flex min-h-9 items-center gap-2 rounded-xl px-3 py-2 text-sm font-semibold',
          focusRingClass,
          variant === 'default' &&
            (nav.isActive
              ? 'bg-accent-surface text-accent-fg'
              : 'text-fg-muted hover:bg-surface-sunken hover:text-fg-default'),
          variant === 'subtle' &&
            (nav.isActive
              ? 'text-accent-fg underline'
              : 'text-fg-muted hover:text-fg-default'),
          extra,
        )
      }}
    />
  )
}

/** Alias matching FR-2 name. Prefer importing `UiNavLink` when colliding with react-router. */
export { UiNavLink as NavLink }
