import { useState, useRef, useEffect, useId } from 'react'
import {
  PanelLeftClose,
  PanelLeftOpen,
  ChevronUp,
  Scale,
  Shield,
  FileText,
  Globe,
} from 'lucide-react'
import { useShellNav } from './use-shell-nav'
import { SideNavTooltip } from './side-nav-tooltip'
import { MARKETING_SITE_URLS } from '../../lib/marketing-site'

export function SideNavFooter() {
  const { sideNavCollapsed, toggleSideNav } = useShellNav()
  const [open, setOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)
  const buttonId = useId()
  const menuId = useId()
  const year = new Date().getFullYear()

  // Handle clicking outside to close
  useEffect(() => {
    if (!open) return
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setOpen(false)
      }
    }
    function handleKeyDown(event: KeyboardEvent) {
      if (event.key === 'Escape') {
        setOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('mousedown', handleClickOutside)
      document.removeEventListener('keydown', handleKeyDown)
    }
  }, [open])

  return (
    <footer
      className={`shrink-0 border-t border-border-default px-3 py-2.5 text-[11px] leading-snug text-fg-muted ${ sideNavCollapsed ? 'flex justify-center' : '' }`}
    >
      <SideNavTooltip content={sideNavCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
        <button
          type="button"
          onClick={toggleSideNav}
          className={`mb-2 flex w-full items-center gap-3 rounded-lg px-2 py-1.5 text-sm font-medium text-fg-muted transition-colors hover:bg-surface-raised hover:text-fg-default ${ sideNavCollapsed ? 'justify-center' : '' }`}
          title={!sideNavCollapsed ? 'Collapse sidebar' : undefined}
        >
          {sideNavCollapsed ? (
            <PanelLeftOpen className="h-5 w-5 shrink-0" />
          ) : (
            <>
              <PanelLeftClose className="h-5 w-5 shrink-0" />
              <span>Collapse</span>
            </>
          )}
        </button>
      </SideNavTooltip>

      {!sideNavCollapsed && (
        <div className="relative flex flex-col gap-1.5 pt-0.5 motion-safe:animate-in motion-safe:fade-in duration-200">
          <div ref={dropdownRef} className="relative">
            <button
              id={buttonId}
              type="button"
              aria-haspopup="menu"
              aria-expanded={open}
              aria-controls={menuId}
              onClick={() => setOpen((prev) => !prev)}
              className="flex w-full items-center justify-between rounded-lg border border-border-default bg-surface-raised px-2 py-1.5 font-medium text-fg-default shadow-sm motion-safe:transition-[border-color,background-color,color,box-shadow] duration-200 hover:border-border-strong hover:bg-surface-overlay"
            >
              <span className="flex items-center gap-1.5">
                <Scale className="h-3.5 w-3.5 text-fg-muted" aria-hidden="true" />
                <span>Legal Agreements</span>
              </span>
              <ChevronUp
                className={`h-3.5 w-3.5 text-fg-muted transition-transform duration-200 ${ open ? 'rotate-180' : '' }`}
                aria-hidden="true"
              />
            </button>

            {open && (
              <div
                id={menuId}
                role="menu"
                aria-labelledby={buttonId}
                className="absolute bottom-full start-0 z-50 mb-2 w-full min-w-[220px] origin-bottom motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-bottom-1 duration-150 overflow-hidden rounded-xl border border-border-default bg-surface-raised p-1 shadow-lg"
              >
                <div className="px-2.5 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-fg-subtle">
                  Legal Documents
                </div>
                <div className="h-[1px] bg-border-default mx-1 mb-1" />
                <a
                  href={MARKETING_SITE_URLS.terms}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken hover:text-fg-default"
                  onClick={() => setOpen(false)}
                >
                  <FileText className="h-4 w-4 shrink-0 text-fg-subtle" aria-hidden="true" />
                  <span className="truncate">Terms of use</span>
                </a>
                <a
                  href={MARKETING_SITE_URLS.privacy}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken hover:text-fg-default"
                  onClick={() => setOpen(false)}
                >
                  <Shield className="h-4 w-4 shrink-0 text-fg-subtle" aria-hidden="true" />
                  <span className="truncate">Privacy policy</span>
                </a>
                <a
                  href={MARKETING_SITE_URLS.accessibility}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken hover:text-fg-default"
                  onClick={() => setOpen(false)}
                >
                  <Globe className="h-4 w-4 shrink-0 text-fg-subtle" aria-hidden="true" />
                  <span className="truncate">Accessibility</span>
                </a>
                <div className="h-[1px] bg-border-default mx-1 my-1" />
                <a
                  href={MARKETING_SITE_URLS.californiaPrivacyRights}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken hover:text-fg-default"
                  onClick={() => setOpen(false)}
                >
                  <Scale className="h-4 w-4 shrink-0 text-fg-subtle" aria-hidden="true" />
                  <span className="leading-tight text-[11px] truncate whitespace-normal">
                    Do Not Sell or Share My Info
                  </span>
                </a>
                <div className="h-[1px] bg-border-default mx-1 my-1" />
                <div className="px-2.5 py-1.5 text-[10px] text-fg-subtle">
                  © {year} Lextures
                </div>
              </div>
            )}
          </div>
        </div>
      )}
    </footer>
  )
}
