import { ArrowLeft, ArrowRight, BookOpen, Search } from 'lucide-react'
import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { BLOG_PAGE_SIZE, blogPostSearchText } from '../lib/blog-index-filters'
import { authorDisplayName } from '../lib/authors'
import { allPosts, formatDate } from '../utils/blog'
import { EDITORIAL_PILLARS, editorialPillar } from '../lib/editorial-pillars'

export function BlogIndex() {
  return (
    <div className="relative min-h-screen overflow-x-hidden bg-white text-slate-900" data-blog-index data-page-size={BLOG_PAGE_SIZE}>
      <Header />

      <main>
        <section className="border-b border-slate-200 bg-white py-16 sm:py-20">
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="flex items-center gap-3">
              <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-50 text-indigo-600 ring-1 ring-indigo-200">
                <BookOpen className="h-5 w-5" aria-hidden />
              </div>
              <p className="text-[0.7rem] font-semibold uppercase tracking-[0.22em] text-indigo-500">
                Lextures Blog
              </p>
            </div>
            <h1 className="mt-5 text-4xl font-semibold tracking-tight text-slate-900 sm:text-5xl">
              Writing
            </h1>
            <p className="mt-4 max-w-xl text-lg leading-relaxed text-slate-600">
              Thoughts on adaptive learning, educational technology, and building software for institutions that run at scale.
            </p>

            <form data-blog-filters role="search" className="mt-10 grid max-w-3xl gap-3 sm:grid-cols-3">
              <div className="relative">
                <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3">
                  <Search className="h-4 w-4 text-slate-400" aria-hidden />
                </div>
                <label className="sr-only" htmlFor="blog-search">Search articles</label>
                <input
                  id="blog-search"
                  data-blog-search
                  type="search"
                  name="q"
                  placeholder="Search articles..."
                  autoComplete="off"
                  className="block w-full rounded-lg border border-slate-200 bg-slate-50 py-2.5 pl-10 pr-3 text-sm placeholder-stone-400 outline-none transition-colors focus:border-indigo-500 focus:ring-1 focus:ring-accent"
                />
              </div>
              <label className="sr-only" htmlFor="pillar-filter">Filter by guide</label>
              <select id="pillar-filter" name="pillar" data-blog-pillar defaultValue="" className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm">
                <option value="">All guides</option>
                {EDITORIAL_PILLARS.map(pillar => <option key={pillar.id} value={pillar.id}>{pillar.title}</option>)}
              </select>
              <label className="sr-only" htmlFor="author-filter">Filter by author</label>
              <select id="author-filter" name="author" data-blog-author defaultValue="" className="rounded-lg border border-slate-200 bg-slate-50 px-3 py-2.5 text-sm">
                <option value="">All authors</option>
                {[...new Set(allPosts.map(post => post.author))].map(author => <option key={author} value={author}>{authorDisplayName(author)}</option>)}
              </select>
            </form>
            <p className="sr-only" aria-live="polite" data-blog-status></p>
          </div>
        </section>

        <section className="py-16 sm:py-20">
          <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
            <div className="py-20 text-center" data-blog-empty hidden={allPosts.length > 0}>
              <p className="text-lg text-slate-500">
                {allPosts.length === 0
                  ? 'No published posts are available yet.'
                  : 'No posts found matching your search.'}
              </p>
              {allPosts.length > 0 && (
                <button
                  type="button"
                  data-blog-clear
                  className="mt-4 text-sm font-semibold text-accent hover:underline"
                >
                  Clear search
                </button>
              )}
            </div>
            <div className="divide-y divide-stone-200/80" data-blog-list hidden={allPosts.length === 0}>
              {allPosts.map((post) => (
                <article
                  key={post.slug}
                  data-blog-post
                  data-search={blogPostSearchText([
                    post.title,
                    post.description,
                    post.slug,
                    post.author,
                    authorDisplayName(post.author),
                    editorialPillar(post.pillar)?.title,
                    post.pillar ? `pillar:${post.pillar}` : '',
                    post.author ? `author:${post.author}` : '',
                  ])}
                  className="group py-10 first:pt-0"
                >
                  <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between sm:gap-8">
                    <div className="flex-1">
                      <time
                        dateTime={post.date}
                        className="text-xs font-medium uppercase tracking-widest text-slate-400"
                      >
                        {formatDate(post.date)}
                      </time>
                      <h2 className="mt-2 text-xl font-semibold leading-snug text-slate-900 sm:text-2xl">
                        <a
                          href={`/blog/${post.slug}`}
                          className="no-underline transition-colors hover:text-accent"
                        >
                          {post.title}
                        </a>
                      </h2>
                      <p className="mt-3 max-w-2xl text-base leading-relaxed text-slate-600">
                        {post.description}
                      </p>
                      <p className="mt-2 text-sm text-slate-400">
                        By {authorDisplayName(post.author)} · {editorialPillar(post.pillar)?.title}
                      </p>
                    </div>
                    <a
                      href={`/blog/${post.slug}`}
                      className="btn-primary shrink-0 gap-2 self-start"
                      aria-label={`Read ${post.title}`}
                    >
                      Read
                      <ArrowRight className="h-4 w-4" aria-hidden />
                    </a>
                  </div>
                </article>
              ))}
            </div>
            <nav className="mt-16 flex items-center justify-between border-t border-slate-200 pt-8" aria-label="Pagination" data-blog-pagination hidden>
              <p className="text-sm text-slate-500" data-blog-page-label>Page 1</p>
              <div className="flex gap-2">
                <button type="button" data-blog-prev className="btn-secondary px-4 py-2 disabled:opacity-50">
                  <ArrowLeft className="h-4 w-4" aria-hidden />
                  Previous
                </button>
                <button type="button" data-blog-next className="btn-secondary px-4 py-2 disabled:opacity-50">
                  Next
                  <ArrowRight className="h-4 w-4" aria-hidden />
                </button>
              </div>
            </nav>
          </div>
        </section>
      </main>

      <SiteFooter />
    </div>
  )
}
