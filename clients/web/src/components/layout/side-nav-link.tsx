import { type ReactNode } from 'react'
import { NavLink, type NavLinkProps } from 'react-router-dom'
import {
  sideNavActiveClass,
  sideNavLinkClass,
  sideNavSubActiveClass,
  sideNavSubLinkClass,
} from './side-nav-styles'
import { useShellNav } from './use-shell-nav'
import { SideNavTooltip } from './side-nav-tooltip'

interface SideNavLinkProps extends NavLinkProps {
  icon?: ReactNode
  children: ReactNode
  badge?: ReactNode
  nested?: boolean
}

export function SideNavLink({
  icon,
  children,
  badge,
  nested = false,
  className,
  ...props
}: SideNavLinkProps) {
  const { sideNavCollapsed } = useShellNav()

  const label = typeof children === 'string' ? children : ''

  return (
    <SideNavTooltip content={label}>
      <NavLink
        {...props}
        className={(navProps) => {
          const baseClass = typeof className === 'function' ? className(navProps) : className
          const activeClass = navProps.isActive
            ? nested
              ? sideNavSubActiveClass
              : sideNavActiveClass
            : ''
          const collapseClass = sideNavCollapsed ? 'justify-center' : ''
          const linkClass = nested ? sideNavSubLinkClass : sideNavLinkClass
          return `${linkClass} ${activeClass} ${collapseClass} ${baseClass || ''}`
        }}
      >
        {!nested && (
          <span className="flex h-5 w-5 shrink-0 items-center justify-center text-current opacity-90">
            {icon}
          </span>
        )}
        {!sideNavCollapsed && (
          <span className="flex min-w-0 flex-1 items-center justify-between gap-2">
            <span className="truncate">{children}</span>
            {badge}
          </span>
        )}
        {sideNavCollapsed && badge && <span className="absolute end-2 top-2">{badge}</span>}
      </NavLink>
    </SideNavTooltip>
  )
}
