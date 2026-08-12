import { useId, useState, type ReactNode } from 'react'

export function KeyFinding({ figureId, children }: { figureId: string; children: ReactNode }) {
  return <figure className="my-8 border-l-4 py-3 pl-6" style={{ borderColor: 'var(--coral)' }}><blockquote className="font-display text-2xl font-semibold">{children}</blockquote><figcaption className="mt-2 text-sm">Published dataset figure: <code>{figureId}</code></figcaption></figure>
}

export function CiteThis({ citation, url }: { citation: string; url: string }) {
  return <aside className="my-8 rounded-[16px] border p-5" style={{ borderColor: 'var(--line-card)', background: 'var(--panel)' }} aria-labelledby="cite-this-heading"><h2 id="cite-this-heading" className="font-display text-xl font-semibold">Cite this report</h2><p className="mt-3">{citation}</p><p className="mt-2"><a href={url}>{url}</a></p></aside>
}

export function DatasetDownload({ csv, json, dictionary }: { csv: string; json: string; dictionary: string }) {
  return <aside aria-labelledby="dataset-download-heading"><h2 id="dataset-download-heading" className="font-display text-xl font-semibold">Download the aggregate dataset</h2><ul className="mt-3 flex flex-wrap gap-5"><li><a href={csv} download>CSV</a></li><li><a href={json} download>JSON</a></li><li><a href={dictionary}>Data dictionary</a></li></ul><p className="mt-2 text-sm">Licensed CC BY 4.0.</p></aside>
}

export function ResearchChart({ title, alt, chart, table }: { title: string; alt: string; chart: ReactNode; table: ReactNode }) {
  const [showTable, setShowTable] = useState(false)
  const id = useId()
  return <figure className="my-10"><figcaption id={`${id}-caption`} className="font-display text-xl font-semibold">{title}</figcaption><p id={`${id}-description`} className="mt-2 text-sm">{alt}</p><div className="mt-4 overflow-x-auto" role="img" aria-labelledby={`${id}-caption`} aria-describedby={`${id}-description`}>{chart}</div><button type="button" className="mt-4 rounded-full border px-4 py-2 font-semibold" aria-expanded={showTable} aria-controls={`${id}-table`} onClick={() => setShowTable(v => !v)}>{showTable ? 'Hide data table' : 'Show data table'}</button><div id={`${id}-table`} className={`mt-4 overflow-x-auto ${showTable ? '' : 'sr-only'}`}>{table}</div></figure>
}

export function Methodology({ children }: { children: ReactNode }) {
  return <section aria-labelledby="methodology-heading"><h2 id="methodology-heading" className="font-display text-3xl font-semibold">Methodology</h2><div className="mt-5 space-y-4 leading-relaxed">{children}</div></section>
}

