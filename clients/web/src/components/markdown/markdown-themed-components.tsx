import type { Components } from 'react-markdown'
import type { ResolvedMarkdownTheme } from '../../lib/markdown-theme'

export type ThemedMarkdownOptions = {
  imgClassName?: string
  linkComponent?: Components['a']
}

/** ReactMarkdown components using the classic LMS markdown theme (Tailwind utility classes). */
export function createThemedMarkdownComponents(
  theme: ResolvedMarkdownTheme,
  opts?: ThemedMarkdownOptions,
): Components {
  const o = theme.styleOverrides
  const c = theme.classes
  const imgClass =
    opts?.imgClassName
    ?? 'max-h-[min(28rem,80vh)] w-auto max-w-full rounded-lg border border-slate-200 dark:border-neutral-700'

  return {
    h1: ({ children }) => (
      <h1 className={c.h1} style={o.h1}>
        {children}
      </h1>
    ),
    h2: ({ children }) => (
      <h2 className={c.h2} style={o.h2}>
        {children}
      </h2>
    ),
    h3: ({ children }) => (
      <h3 className={c.h3} style={o.h3}>
        {children}
      </h3>
    ),
    h4: ({ children }) => (
      <h4 className={c.h3} style={o.h3}>
        {children}
      </h4>
    ),
    h5: ({ children }) => (
      <h5 className={c.h3} style={o.h3}>
        {children}
      </h5>
    ),
    h6: ({ children }) => (
      <h6 className={c.h3} style={o.h3}>
        {children}
      </h6>
    ),
    p: ({ children }) => (
      <p className={c.p} style={o.p}>
        {children}
      </p>
    ),
    ul: ({ children }) => (
      <ul className={c.ul} style={o.ul}>
        {children}
      </ul>
    ),
    ol: ({ children }) => (
      <ol className={c.ol} style={o.ol}>
        {children}
      </ol>
    ),
    li: ({ children }) => (
      <li className={c.li} style={o.li}>
        {children}
      </li>
    ),
    a:
      opts?.linkComponent
      ?? (({ children, href }) => (
        <a href={href} className={c.a} style={o.a} target="_blank" rel="noreferrer noopener">
          {children}
        </a>
      )),
    blockquote: ({ children }) => (
      <blockquote className={c.blockquote} style={o.blockquote}>
        {children}
      </blockquote>
    ),
    code: ({ className, children }) => {
      const inline = !className
      if (inline) {
        return (
          <code className={c.codeInline} style={o.codeInline}>
            {children}
          </code>
        )
      }
      return <code className={className}>{children}</code>
    },
    pre: ({ children }) => (
      <pre className={c.pre} style={o.pre}>
        {children}
      </pre>
    ),
    table: ({ children }) => (
      <div className={c.tableWrap}>
        <table className={c.table} style={o.table}>
          {children}
        </table>
      </div>
    ),
    thead: ({ children }) => (
      <thead className={c.thead} style={o.thead}>
        {children}
      </thead>
    ),
    th: ({ children }) => (
      <th className={c.th} style={o.th}>
        {children}
      </th>
    ),
    td: ({ children }) => (
      <td className={c.td} style={o.td}>
        {children}
      </td>
    ),
    hr: () => <hr className={c.hr} style={o.hr} />,
    img: ({ src, alt }) => (
      <img src={src ?? undefined} alt={alt ?? ''} className={imgClass} loading="lazy" />
    ),
  }
}
