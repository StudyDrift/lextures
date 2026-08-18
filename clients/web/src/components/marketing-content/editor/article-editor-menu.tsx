import { useRef, useState, type ReactNode } from 'react'
import { ChevronDown } from 'lucide-react'
import { Button, Menu, type MenuItem } from '../../ui'

type Props = {
  label: string
  items: MenuItem[]
  variant?: 'secondary' | 'ghost'
  icon?: ReactNode
  placement?: 'bottom-start' | 'bottom-end'
}

export function ArticleEditorMenu({ label, items, variant = 'ghost', icon, placement = 'bottom-start' }: Props) {
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLButtonElement>(null)
  return <><Button ref={ref} size="sm" variant={variant} className="min-h-6" aria-haspopup="menu" aria-expanded={open} onClick={() => setOpen((value) => !value)}>{icon}{label}<ChevronDown aria-hidden className="h-3.5 w-3.5 opacity-70" /></Button><Menu open={open} onOpenChange={setOpen} anchorRef={ref} items={items} placement={placement} /></>
}
