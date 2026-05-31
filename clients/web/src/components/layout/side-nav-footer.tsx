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
import { RELEASE_VERSION } from '../../lib/release-version'
import { useShellNav } from './use-shell-nav'
import { SideNavTooltip } from './side-nav-tooltip'
import { MARKETING_SITE_URLS } from '../../lib/marketing-site'

function versionLabel(version: string) {
  const trimmed = version.trim()
  if (!trimmed) return 'v0'
  return trimmed.startsWith('v') ? trimmed : `v${trimmed}`
}

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
      className={`shrink-0 border-t border-slate-200/80 px-3 py-2.5 text-[11px] leading-snug text-slate-700 dark:border-neutral-800 dark:text-neutral-400 ${
        sideNavCollapsed ? 'flex justify-center' : ''
      }`}
    >
      <SideNavTooltip content={sideNavCollapsed ? 'Expand sidebar' : 'Collapse sidebar'}>
        <button
          type="button"
          onClick={toggleSideNav}
          className={`mb-2 flex w-full items-center gap-3 rounded-lg px-2 py-1.5 text-sm font-medium text-slate-700 transition-colors hover:bg-white/80 hover:text-slate-900 dark:text-neutral-400 dark:hover:bg-neutral-800/90 dark:hover:text-neutral-50 ${
            sideNavCollapsed ? 'justify-center' : ''
          }`}
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
          <div className="flex items-center justify-between text-slate-600 dark:text-neutral-400">
            <span>© {year} Lextures</span>
            <span
              className="tabular-nums text-[10px]"
              title="App version"
            >
              {versionLabel(RELEASE_VERSION)}
            </span>
          </div>

          <div ref={dropdownRef} className="relative mt-1">
            <button
              id={buttonId}
              type="button"
              aria-haspopup="menu"
              aria-expanded={open}
              aria-controls={menuId}
              onClick={() => setOpen((prev) => !prev)}
              className="flex w-full items-center justify-between rounded-lg border border-slate-200/60 bg-white/50 px-2 py-1.5 font-medium shadow-[0_1px_2px_rgba(0,0,0,0.02)] backdrop-blur-sm motion-safe:transition-all duration-200 hover:border-slate-300 hover:bg-white hover:text-slate-900 dark:border-neutral-800/60 dark:bg-neutral-900/40 dark:hover:border-neutral-700 dark:hover:bg-neutral-900 dark:hover:text-neutral-100"
            >
              <span className="flex items-center gap-1.5">
                <Scale className="h-3.5 w-3.5 text-slate-400 dark:text-neutral-500" aria-hidden="true" />
                <span>Legal Agreements</span>
              </span>
              <ChevronUp
                className={`h-3.5 w-3.5 text-slate-400 transition-transform duration-200 dark:text-neutral-500 ${
                  open ? 'rotate-180' : ''
                }`}
                aria-hidden="true"
              />
            </button>

            {open && (
              <div
                id={menuId}
                role="menu"
                aria-labelledby={buttonId}
                className="absolute bottom-full start-0 z-50 mb-2 w-full min-w-[220px] origin-bottom motion-safe:animate-in motion-safe:fade-in motion-safe:slide-in-from-bottom-1 duration-150 overflow-hidden rounded-xl border border-slate-200/80 bg-white/95 p-1 shadow-lg shadow-slate-950/5 backdrop-blur-md dark:border-neutral-800/90 dark:bg-neutral-900/95 dark:shadow-black/30"
              >
                <div className="px-2.5 py-1.5 text-[10px] font-semibold uppercase tracking-wider text-slate-400 dark:text-neutral-500">
                  Legal Documents
                </div>
                <div className="h-[1px] bg-slate-100 dark:bg-neutral-800/80 mx-1 mb-1" />
                <a
                  href={MARKETING_SITE_URLS.terms}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-slate-700 transition hover:bg-slate-50 hover:text-slate-950 dark:text-neutral-300 dark:hover:bg-neutral-800/50 dark:hover:text-neutral-50"
                  onClick={() => setOpen(false)}
                >
                  <FileText className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden="true" />
                  <span className="truncate">Terms of use</span>
                </a>
                <a
                  href={MARKETING_SITE_URLS.privacy}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-slate-700 transition hover:bg-slate-50 hover:text-slate-950 dark:text-neutral-300 dark:hover:bg-neutral-800/50 dark:hover:text-neutral-50"
                  onClick={() => setOpen(false)}
                >
                  <Shield className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden="true" />
                  <span className="truncate">Privacy policy</span>
                </a>
                <a
                  href={MARKETING_SITE_URLS.accessibility}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-slate-700 transition hover:bg-slate-50 hover:text-slate-950 dark:text-neutral-300 dark:hover:bg-neutral-800/50 dark:hover:text-neutral-50"
                  onClick={() => setOpen(false)}
                >
                  <Globe className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden="true" />
                  <span className="truncate">Accessibility</span>
                </a>
                <div className="h-[1px] bg-slate-100 dark:bg-neutral-800/80 mx-1 my-1" />
                <a
                  href={MARKETING_SITE_URLS.californiaPrivacyRights}
                  target="_blank"
                  rel="noopener noreferrer"
                  role="menuitem"
                  className="flex items-center gap-2 rounded-lg px-2.5 py-2 text-slate-700 transition hover:bg-slate-50 hover:text-slate-950 dark:text-neutral-300 dark:hover:bg-neutral-800/50 dark:hover:text-neutral-50"
                  onClick={() => setOpen(false)}
                >
                  <Scale className="h-4 w-4 shrink-0 text-slate-400 dark:text-neutral-500" aria-hidden="true" />
                  <span className="leading-tight text-[11px] truncate whitespace-normal">
                    Do Not Sell or Share My Info
                  </span>
                </a>
              </div>
            )}
          </div>
        </div>
      )}
    </footer>
  )
}
