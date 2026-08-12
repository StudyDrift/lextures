import { HelpCircle, Search } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { HELP_CATEGORIES } from '../docs/_categories'

type SearchItem = { title: string; description: string; path: string; headings: string[]; category: string }

export function DocsIndex() {
  const [query, setQuery] = useState('')
  const [items, setItems] = useState<SearchItem[] | null>(null)
  const [loading, setLoading] = useState(false)

  async function loadIndex() {
    if (items || loading) return
    setLoading(true)
    try {
      const response = await fetch('/docs-search-index.json')
      setItems(response.ok ? await response.json() : [])
    } finally {
      setLoading(false)
    }
  }

  const results = useMemo(() => {
    const needle = query.trim().toLowerCase()
    if (!needle || !items) return []
    return items.filter(item => [item.title, item.description, item.category, ...item.headings].join(' ').toLowerCase().includes(needle)).slice(0, 20)
  }, [items, query])

  return <div className="relative min-h-screen bg-white text-slate-900">
    <Header />
    <main>
      <section className="border-b border-slate-200 py-16 sm:py-20">
        <div className="mx-auto max-w-6xl px-4 sm:px-6 lg:px-8">
          <div className="flex items-center gap-3 text-indigo-600"><HelpCircle className="h-6 w-6" aria-hidden /><span className="text-sm font-semibold uppercase tracking-widest">Lextures help center</span></div>
          <h1 className="mt-5 text-4xl font-semibold tracking-tight sm:text-5xl">How can we help?</h1>
          <p className="mt-4 max-w-3xl text-lg leading-relaxed text-slate-600">Find practical answers for setting up, teaching, learning, administering, integrating, and self-hosting Lextures. Search all articles or browse a category to see its complete set of guides.</p>
          <div role="search" className="relative mt-8 max-w-3xl">
            <label htmlFor="docs-search" className="sr-only">Search help articles</label>
            <Search className="pointer-events-none absolute left-4 top-3.5 h-5 w-5 text-slate-400" aria-hidden />
            <input id="docs-search" type="search" role="searchbox" aria-controls="docs-search-results" value={query} onFocus={loadIndex} onChange={event => { setQuery(event.target.value); void loadIndex() }} placeholder="Search titles, descriptions, and headings" className="w-full rounded-xl border border-slate-300 py-3 pl-12 pr-4" />
          </div>
          <p className="sr-only" aria-live="polite">{query ? loading ? 'Loading help search' : `${results.length} results found` : ''}</p>
          {query && <ul id="docs-search-results" className="mt-4 max-w-3xl divide-y rounded-xl border border-slate-200 bg-white">
            {!loading && results.length === 0 && <li className="p-5 text-slate-600">No matching articles. Try a feature name or browse the categories below.</li>}
            {results.map(item => <li key={item.path} className="p-5"><a className="font-semibold text-indigo-700" href={item.path}>{item.title}</a><p className="mt-1 text-sm text-slate-600">{item.description}</p></li>)}
          </ul>}
        </div>
      </section>
      <section className="py-16"><div className="mx-auto grid max-w-6xl gap-5 px-4 sm:grid-cols-2 sm:px-6 lg:grid-cols-3 lg:px-8">
        {HELP_CATEGORIES.map(category => <a key={category.id} href={`/docs/${category.id}`} className="rounded-xl border border-slate-200 p-6 no-underline transition hover:border-indigo-300 hover:shadow-sm"><h2 className="text-xl font-semibold">{category.title}</h2><p className="mt-3 text-sm leading-6 text-slate-600">{category.description}</p></a>)}
      </div></section>
    </main>
    <SiteFooter />
  </div>
}
