import { useSsrData } from '../lib/ssr-context'
import { breadcrumbPaths, routeLabel } from '../lib/information-architecture'

export function Breadcrumbs() {
  const ssr = useSsrData()
  const pathname = ssr.path ?? (typeof window !== 'undefined' ? window.location.pathname.replace(/\/$/, '') || '/' : '/')
  const paths = breadcrumbPaths(pathname)
  if (!paths.length) return null
  return (
    <nav aria-label="Breadcrumb" className="border-b" style={{ borderColor: 'var(--line)' }}>
      <ol className="mx-auto flex max-w-[1200px] items-center gap-2 overflow-hidden px-5 py-2.5 text-[13px] md:px-10 xl:px-14">
        {paths.map((path, index) => {
          const current = index === paths.length - 1
          return <li key={path} className="flex min-w-0 items-center gap-2">
            {index > 0 && <span aria-hidden>/</span>}
            {current
              ? <span aria-current="page" className="truncate" style={{ color: 'var(--ink-nav)' }}>{routeLabel(path)}</span>
              : <a href={path} className="whitespace-nowrap no-underline" style={{ color: 'var(--text-soft)' }}>{routeLabel(path)}</a>}
          </li>
        })}
      </ol>
    </nav>
  )
}
