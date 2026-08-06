import { ChevronDown } from 'lucide-react'
import { useState, type ReactNode } from 'react'

type SidebarSectionProps = {
  title: string
  defaultOpen?: boolean
  children: ReactNode
}

/** Collapsible panel for settings sidebars (Gutenberg-style). */
export function SidebarSection({ title, defaultOpen = true, children }: SidebarSectionProps) {
  const [open, setOpen] = useState(defaultOpen)
  return (
    <div className="border-b border-border-default pb-3 last:border-0 dark:border-border-default">
      <button
        type="button"
        className="flex w-full items-center justify-between gap-2 py-2 text-start text-[13px] font-semibold text-fg-default"
        onClick={() => setOpen((o) => !o)}
        aria-expanded={open}
      >
        {title}
        <ChevronDown
          className={`h-4 w-4 shrink-0 text-fg-muted transition-transform dark:text-fg-muted ${open ? 'rotate-180' : ''}`}
          aria-hidden
        />
      </button>
      {open && <div className="mt-1 space-y-3 text-sm">{children}</div>}
    </div>
  )
}
