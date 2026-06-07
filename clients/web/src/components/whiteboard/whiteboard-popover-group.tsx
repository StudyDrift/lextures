import { useRef, useState, type ReactNode } from 'react'

export function WhiteboardPopoverGroup({ trigger, children }: { trigger: ReactNode; children: ReactNode }) {
  const [open, setOpen] = useState(false)
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null)

  const show = () => {
    if (timerRef.current) clearTimeout(timerRef.current)
    setOpen(true)
  }
  const hide = () => {
    timerRef.current = setTimeout(() => setOpen(false), 80)
  }

  return (
    <div className="relative" onMouseEnter={show} onMouseLeave={hide}>
      <div onClick={() => setOpen((o) => !o)}>{trigger}</div>
      {open && (
        <div
          className="absolute left-full top-0 z-50 ml-2 rounded-xl border border-slate-200 bg-white p-2 shadow-lg dark:border-neutral-800 dark:bg-neutral-950"
          onMouseEnter={show}
          onMouseLeave={hide}
        >
          {children}
        </div>
      )}
    </div>
  )
}
