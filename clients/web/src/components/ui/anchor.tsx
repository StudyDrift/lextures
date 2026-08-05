/**
 * Focus-anchor attachment (CC.8).
 *
 * Prefer `useAnchorRef` when a host element already exists and you only need a
 * ref. Use `<Anchor>` when you need a wrapper (region kind, or third-party
 * nodes you cannot ref).
 *
 * When `asChild` is true the single child is cloned with `data-focus-anchor`
 * (no extra DOM node). Otherwise a `div` wrapper is used (region anchors).
 */

import {
  Children,
  cloneElement,
  isValidElement,
  type HTMLAttributes,
  type ReactElement,
  type ReactNode,
} from 'react'

type AnchorProps = {
  id: string
  /** For entity-kind anchors — pairs with `?focusEntity=`. */
  entityId?: string
  /**
   * When true, merge attributes onto the single child (no wrapper).
   * Child must accept a ref and HTML attributes.
   */
  asChild?: boolean
  children?: ReactNode
  className?: string
} & Omit<HTMLAttributes<HTMLElement>, 'id'>

export function Anchor({
  id,
  entityId,
  asChild,
  children,
  className,
  ...rest
}: AnchorProps) {
  const dataAttrs = {
    'data-focus-anchor': id,
    ...(entityId ? { 'data-focus-entity': entityId } : {}),
  }

  if (asChild) {
    const child = Children.only(children)
    if (!isValidElement(child)) {
      return null
    }
    const el = child as ReactElement<Record<string, unknown>>
    return cloneElement(el, {
      ...dataAttrs,
      className: [el.props.className, className].filter(Boolean).join(' ') || undefined,
      ...rest,
    })
  }

  return (
    <div
      className={className}
      // Region anchors need a focus target; control anchors usually wrap a control.
      tabIndex={rest.tabIndex ?? -1}
      {...dataAttrs}
      {...rest}
    >
      {children}
    </div>
  )
}
