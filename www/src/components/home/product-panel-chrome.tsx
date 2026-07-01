import type { ReactNode } from 'react'

type ProductPanelChromeProps = {
  filename: string
  term?: string
  children: ReactNode
  className?: string
}

export function ProductPanelChrome({
  filename,
  term = 'Fall 2026',
  children,
  className = '',
}: ProductPanelChromeProps) {
  return (
    <div
      className={`overflow-hidden border bg-[var(--panel)] ${className}`}
      style={{
        borderColor: 'var(--line-card)',
        borderRadius: 'var(--radius-panel)',
        boxShadow: 'var(--shadow-panel)',
      }}
    >
      <div
        className="flex items-center gap-3 border-b px-4 py-2.5"
        style={{ backgroundColor: 'var(--panel-sunken)', borderColor: '#E8E2D4' }}
      >
        <div className="flex items-center gap-1.5" aria-hidden>
          <span className="h-[11px] w-[11px] rounded-full" style={{ backgroundColor: '#DCD5C5' }} />
          <span className="h-[11px] w-[11px] rounded-full" style={{ backgroundColor: '#DCD5C5' }} />
          <span className="h-[11px] w-[11px] rounded-full" style={{ backgroundColor: '#DCD5C5' }} />
        </div>
        <span
          className="font-mono text-[11px]"
          style={{ color: 'var(--muted)' }}
        >
          {filename}
        </span>
        <span
          className="ml-auto rounded px-2 py-0.5 font-mono text-[10px]"
          style={{ backgroundColor: '#ECE6D8', color: 'var(--muted)' }}
        >
          {term}
        </span>
      </div>
      {children}
    </div>
  )
}
