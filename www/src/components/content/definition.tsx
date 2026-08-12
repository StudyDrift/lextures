import type { ReactNode } from 'react'
export function Definition({ term, children }: { term: string; children: ReactNode }) { return <dfn className="content-definition" data-definition-term={term}><strong>{term}</strong>{children}</dfn> }
