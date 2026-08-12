import type { ReactNode } from 'react'
export function ComparisonTable({ summary, children }: { summary: string; children: ReactNode }) { return <section className="content-comparison"><p className="content-table-summary">{summary}</p><div className="content-table-scroll">{children}</div></section> }
