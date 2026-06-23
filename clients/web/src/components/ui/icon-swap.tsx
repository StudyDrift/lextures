import type { LucideIcon } from 'lucide-react'
import type { ReactNode } from 'react'

type IconSwapProps = {
  active: boolean
  activeIcon: LucideIcon
  inactiveIcon: LucideIcon
  className?: string
  iconClassName?: string
}

const swapTransition =
  'transition-[opacity,transform,filter] duration-300 ease-[cubic-bezier(0.2,0,0,1)]'
const activeState = 'scale-100 opacity-100 blur-0'
const inactiveState = 'scale-[0.25] opacity-0 blur-[4px]'

/** Cross-fades two icons without unmounting — CSS-only enter/exit (plan 22.1, rule 7). */
export function IconSwap({
  active,
  activeIcon: ActiveIcon,
  inactiveIcon: InactiveIcon,
  className = '',
  iconClassName = 'h-4 w-4',
}: IconSwapProps) {
  return (
    <span className={`relative inline-flex items-center justify-center ${className}`}>
      <span
        aria-hidden
        className={`absolute inset-0 flex items-center justify-center ${swapTransition} ${
          active ? activeState : inactiveState
        }`}
      >
        <ActiveIcon className={iconClassName} />
      </span>
      <span
        aria-hidden
        className={`flex items-center justify-center ${swapTransition} ${
          active ? inactiveState : activeState
        }`}
      >
        <InactiveIcon className={iconClassName} />
      </span>
    </span>
  )
}

type IconSwapSlotProps = {
  active: boolean
  activeSlot: ReactNode
  inactiveSlot: ReactNode
  className?: string
}

/** Slot-based variant when icons are not Lucide components. */
export function IconSwapSlots({ active, activeSlot, inactiveSlot, className = '' }: IconSwapSlotProps) {
  return (
    <span className={`relative inline-flex items-center justify-center ${className}`}>
      <span
        aria-hidden
        className={`absolute inset-0 flex items-center justify-center ${swapTransition} ${
          active ? activeState : inactiveState
        }`}
      >
        {activeSlot}
      </span>
      <span
        aria-hidden
        className={`flex items-center justify-center ${swapTransition} ${
          active ? inactiveState : activeState
        }`}
      >
        {inactiveSlot}
      </span>
    </span>
  )
}