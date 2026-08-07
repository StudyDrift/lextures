import {
  createContext,
  useContext,
  useId,
  useState,
  type KeyboardEvent,
  type ReactNode,
} from 'react'
import { cx, focusRingClass } from './utils'

type TabsContextValue = {
  value: string
  setValue: (v: string) => void
  baseId: string
  orientation: 'horizontal' | 'vertical'
}

const TabsContext = createContext<TabsContextValue | null>(null)

export type TabsProps = {
  value?: string
  defaultValue?: string
  onValueChange?: (value: string) => void
  orientation?: 'horizontal' | 'vertical'
  children: ReactNode
  className?: string
}

export function Tabs({
  value: valueProp,
  defaultValue = '',
  onValueChange,
  orientation = 'horizontal',
  children,
  className = '',
}: TabsProps) {
  const baseId = useId()
  const [uncontrolled, setUncontrolled] = useState(defaultValue)
  const value = valueProp ?? uncontrolled
  const setValue = (v: string) => {
    if (valueProp === undefined) setUncontrolled(v)
    onValueChange?.(v)
  }

  return (
    <TabsContext.Provider value={{ value, setValue, baseId, orientation }}>
      <div className={cx(orientation === 'vertical' && 'flex gap-4', className)}>{children}</div>
    </TabsContext.Provider>
  )
}

function useTabs() {
  const ctx = useContext(TabsContext)
  if (!ctx) throw new Error('Tabs compound components must be used within <Tabs>')
  return ctx
}

/** True when the tablist (or document) is RTL — APG inverts horizontal arrows. */
function isRtl(el: HTMLElement): boolean {
  const dir = el.closest('[dir]')?.getAttribute('dir') ?? document.documentElement.dir
  return dir === 'rtl'
}

export type TabListProps = {
  children: ReactNode
  className?: string
  'aria-label'?: string
}

export function TabList({ children, className = '', 'aria-label': ariaLabel }: TabListProps) {
  const { orientation } = useTabs()

  function onKeyDown(e: KeyboardEvent<HTMLDivElement>) {
    const tabs = Array.from(
      e.currentTarget.querySelectorAll<HTMLElement>('[role="tab"]:not([disabled])'),
    )
    if (tabs.length === 0) return
    const current = document.activeElement as HTMLElement
    const idx = tabs.indexOf(current)
    if (idx < 0) return
    const horizontal = orientation === 'horizontal'
    const rtl = horizontal && isRtl(e.currentTarget)
    let next = idx

    const goNext =
      (horizontal && (rtl ? e.key === 'ArrowLeft' : e.key === 'ArrowRight')) ||
      (!horizontal && e.key === 'ArrowDown')
    const goPrev =
      (horizontal && (rtl ? e.key === 'ArrowRight' : e.key === 'ArrowLeft')) ||
      (!horizontal && e.key === 'ArrowUp')

    if (goNext) {
      e.preventDefault()
      next = (idx + 1) % tabs.length
    } else if (goPrev) {
      e.preventDefault()
      next = (idx - 1 + tabs.length) % tabs.length
    } else if (e.key === 'Home') {
      e.preventDefault()
      next = 0
    } else if (e.key === 'End') {
      e.preventDefault()
      next = tabs.length - 1
    } else {
      return
    }
    tabs[next]?.focus()
    tabs[next]?.click()
  }

  return (
    <div
      role="tablist"
      aria-orientation={orientation}
      aria-label={ariaLabel}
      className={cx(
        'flex gap-1 border-b border-border-default',
        orientation === 'vertical' && 'flex-col border-b-0 border-e',
        className,
      )}
      onKeyDown={onKeyDown}
    >
      {children}
    </div>
  )
}

export type TabProps = {
  value: string
  children: ReactNode
  disabled?: boolean
  className?: string
}

export function Tab({ value, children, disabled, className = '' }: TabProps) {
  const { value: selected, setValue, baseId } = useTabs()
  const selectedTab = selected === value
  return (
    <button
      type="button"
      role="tab"
      id={`${baseId}-tab-${value}`}
      aria-selected={selectedTab}
      aria-controls={`${baseId}-panel-${value}`}
      tabIndex={selectedTab ? 0 : -1}
      disabled={disabled}
      className={cx(
        'min-h-9 min-w-9 px-3 py-2 text-sm font-semibold text-fg-muted',
        focusRingClass,
        selectedTab && 'border-b-2 border-accent-solid text-fg-default',
        !selectedTab && 'hover:text-fg-default',
        disabled && 'opacity-50',
        className,
      )}
      onClick={() => setValue(value)}
    >
      {children}
    </button>
  )
}

export type TabPanelProps = {
  value: string
  children: ReactNode
  className?: string
}

export function TabPanel({ value, children, className = '' }: TabPanelProps) {
  const { value: selected, baseId } = useTabs()
  if (selected !== value) return null
  return (
    <div
      role="tabpanel"
      id={`${baseId}-panel-${value}`}
      aria-labelledby={`${baseId}-tab-${value}`}
      tabIndex={0}
      className={cx('py-3 outline-none', focusRingClass, className)}
    >
      {children}
    </div>
  )
}
