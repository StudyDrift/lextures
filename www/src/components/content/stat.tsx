import type { ReactNode } from 'react'
export function Stat({ source, children }: { source: string; children: ReactNode }) { return <figure className="content-stat"><blockquote>{children}</blockquote><figcaption>{source}</figcaption></figure> }
