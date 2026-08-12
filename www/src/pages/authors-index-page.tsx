import { MarketingPageShell } from '../components/marketing-page-shell'
import { getActiveAuthors } from '../lib/authors'
import { postsByAuthor } from '../utils/blog'

export function AuthorsIndexPage() {
  const authors = getActiveAuthors()

  return (
    <MarketingPageShell>
      <main id="main-content">
        <section className="border-b border-slate-200 bg-white py-16 sm:py-20">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-teal-700">
              Authors
            </p>
            <h1 className="font-display mt-4 text-4xl font-semibold tracking-tight text-slate-900 sm:text-5xl">
              People who write for Lextures
            </h1>
            <p className="mt-5 text-lg leading-relaxed text-slate-600">
              Named humans with credentials — not a faceless “team” byline. Every article links back
              here.
            </p>
          </div>
        </section>

        <section className="py-14">
          <div className="mx-auto max-w-3xl px-4 sm:px-6 lg:px-8">
            <ul className="divide-y divide-slate-200">
              {authors.map(a => {
                const posts = postsByAuthor(a.slug)
                return (
                  <li key={a.slug} className="py-8">
                    <a
                      href={`/authors/${a.slug}`}
                      className="font-display text-2xl font-semibold text-slate-900 no-underline hover:underline"
                    >
                      {a.name}
                    </a>
                    <p className="mt-1 text-sm text-slate-500">{a.jobTitle}</p>
                    <p className="mt-3 text-[15px] leading-relaxed text-slate-600">{a.bio}</p>
                    {posts.length > 0 ? (
                      <p className="mt-2 text-sm text-slate-500">
                        {posts.length} article{posts.length === 1 ? '' : 's'}
                      </p>
                    ) : null}
                  </li>
                )
              })}
            </ul>
          </div>
        </section>
      </main>
    </MarketingPageShell>
  )
}
