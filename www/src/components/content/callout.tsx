import type { ReactNode } from 'react'
export function Callout({ type = 'note', children }: { type?: 'note' | 'warning' | 'tip'; children: ReactNode }) { return <aside className={`content-callout content-callout-${type}`}><strong>{type[0].toUpperCase() + type.slice(1)}</strong>{children}</aside> }
