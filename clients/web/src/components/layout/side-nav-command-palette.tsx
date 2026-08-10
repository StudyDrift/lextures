import { Search } from 'lucide-react'
import { useCommandPalette } from '../command-palette/use-command-palette'
import { shortcutHint } from './top-bar-utils'
import { useShellNav } from './use-shell-nav'
import { SideNavTooltip } from './side-nav-tooltip'

/** Full-width capsule trigger below the sidebar header (desktop + mobile drawer). */
export function SideNavCommandPaletteTrigger() {
  const { open } = useCommandPalette()
  const { sideNavCollapsed } = useShellNav()

  return (
    <div className={`shrink-0 pb-3 pt-0.5 ${sideNavCollapsed ? 'px-3' : 'px-3'}`}>
      <SideNavTooltip content="Search">
        <button
          type="button"
          aria-label="Search courses, people, pages, and actions"
          data-command-palette-anchor="sidebar"
          data-onboarding="command-palette"
          onClick={() => open()}
          className={`flex items-center rounded-full bg-[#E8E9EB] text-start text-sm text-fg-muted outline-none transition-[background-color,color,border-color] hover:bg-[#E0E2E5] focus-visible:ring-2 focus-visible:ring-slate-400/35 dark:bg-surface-overlay dark:text-fg-muted dark:hover:bg-neutral-700 dark:focus-visible:ring-neutral-500/40 ${ sideNavCollapsed ? 'h-10 w-10 justify-center p-0 mx-auto' : 'w-full gap-2.5 py-2 ps-3 pe-2' }`}
          title={undefined}
        >
        <Search
          className={`h-4 w-4 shrink-0 text-fg-subtle ${ sideNavCollapsed ? 'h-5 w-5' : '' }`}
          strokeWidth={1.75}
          aria-hidden
        />
        {!sideNavCollapsed && (
          <>
            <span className="min-w-0 flex-1 truncate font-medium text-fg-muted dark:text-fg-muted">
              Search
            </span>
            <kbd className="pointer-events-none flex h-7 min-w-[1.75rem] shrink-0 items-center justify-center rounded-lg border border-black/[0.06] bg-surface-raised px-2 font-mono text-[11px] font-medium text-fg-muted shadow-sm dark:border-white/10 dark:bg-surface-raised dark:text-fg-muted">
              {shortcutHint()}
            </kbd>
          </>
        )}
      </button>
    </SideNavTooltip>
    </div>
  )
}

/** Icon-only trigger when the sidebar is hidden (narrow viewports). */
export function TopBarMobileCommandPaletteButton() {
  const { open } = useCommandPalette()
  return (
    <button
      type="button"
      aria-label="Search courses, people, pages, and actions"
      data-command-palette-anchor="topbar"
      onClick={() => open()}
      className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl text-fg-muted transition-[background-color,color,border-color] hover:bg-surface-sunken focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-400/35 md:hidden dark:text-fg-muted dark:hover:bg-surface-overlay dark:focus-visible:ring-neutral-500/40"
    >
      <Search className="h-5 w-5" strokeWidth={1.75} aria-hidden />
    </button>
  )
}
