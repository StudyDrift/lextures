import { MarketingPageShell } from '../components/marketing-page-shell'
import { getAuthor, isAuthorLinkable } from '../lib/authors'
import { formatDate, postsByAuthor } from '../utils/blog'
import { articlesByAuthor } from '../utils/docs'

export function AuthorPage({ slug }: { slug: string }) {
  const author = getAuthor(slug)

  if (!author || !isAuthorLinkable(slug)) {
    return (
      <MarketingPageShell>
        <main id="main-content" className="mx-auto max-w-3xl px-4 py-24 sm:px-6">
          <h1 className="font-display text-3xl font-semibold text-slate-900">Author not found</h1>
          <p className="mt-3 text-slate-600">
            This author page is unavailable.
          </p>
          <a href="/authors" className="btn-secondary mt-6 inline-flex">
            All authors
          </a>
        </main>
      </MarketingPageShell>
    )
  }

  const posts = postsByAuthor(slug)
  const docs = articlesByAuthor(slug)

  return (
    <MarketingPageShell>
      <main id="main-content">
        <section className="border-b border-slate-200 bg-white py-16 sm:py-20">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <a
              href="/authors"
              className="text-sm font-medium text-slate-500 no-underline hover:text-slate-900"
            >
              ← Authors
            </a>
            <div className="mt-6 flex items-start gap-4">
              <span
                className="flex h-16 w-16 shrink-0 items-center justify-center rounded-full text-lg font-semibold"
                style={{ backgroundColor: 'rgba(106,197,176,0.2)', color: '#2f6f63' }}
                aria-hidden
              >
                {author.name
                  .split(/\s+/)
                  .slice(0, 2)
                  .map(w => w[0])
                  .join('')}
              </span>
              <div>
                <h1 className="font-display text-4xl font-semibold tracking-tight text-slate-900">
                  {author.name}
                </h1>
                <p className="mt-1 text-slate-500">{author.jobTitle}</p>
              </div>
            </div>
            <p className="mt-6 text-lg leading-relaxed text-slate-600">{author.bio}</p>
            {author.knowsAbout.length > 0 ? (
              <p className="mt-4 text-sm text-slate-500">
                Topics: {author.knowsAbout.join(', ')}
              </p>
            ) : null}
            {author.sameAs.length > 0 ? (
              <ul className="mt-4 flex flex-wrap gap-3 text-sm">
                {author.sameAs.map(url => (
                  <li key={url}>
                    <a
                      href={url}
                      className="text-teal-800 underline underline-offset-2"
                      target="_blank"
                      rel="noopener noreferrer"
                    >
                      {url.replace(/^https?:\/\//, '').replace(/\/$/, '')}
                    </a>
                  </li>
                ))}
              </ul>
            ) : null}
          </div>
        </section>

        {(posts.length > 0 || docs.length > 0) && (
          <section className="py-14">
            <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
              {posts.length > 0 ? (
                <>
                  <h2 className="font-display text-2xl font-semibold text-slate-900">Articles</h2>
                  <ul className="mt-6 space-y-5">
                    {posts.map(p => (
                      <li key={p.slug}>
                        <a
                          href={`/blog/${p.slug}`}
                          className="font-medium text-slate-900 no-underline hover:underline"
                        >
                          {p.title}
                        </a>
                        <p className="text-sm text-slate-500">
                          <time dateTime={p.date}>{formatDate(p.date)}</time>
                        </p>
                      </li>
                    ))}
                  </ul>
                </>
              ) : null}
              {docs.length > 0 ? (
                <>
                  <h2
                    className={`font-display text-2xl font-semibold text-slate-900 ${posts.length ? 'mt-12' : ''}`}
                  >
                    Documentation
                  </h2>
                  <ul className="mt-6 space-y-5">
                    {docs.map(d => (
                      <li key={d.slug}>
                        <a
                          href={`/docs/${d.slug}`}
                          className="font-medium text-slate-900 no-underline hover:underline"
                        >
                          {d.title}
                        </a>
                      </li>
                    ))}
                  </ul>
                </>
              ) : null}
            </div>
          </section>
        )}
      </main>
    </MarketingPageShell>
  )
}
