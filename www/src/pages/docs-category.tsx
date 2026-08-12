import { Header } from '../components/header'
import { SiteFooter } from '../components/site-footer'
import { getHelpCategory } from '../docs/_categories'
import { allArticles, articlePath } from '../utils/docs'

export function DocsCategory({ category: categoryId }: { category: string }) {
  const category = getHelpCategory(categoryId)
  const articles = allArticles.filter(article => article.category === categoryId)
  if (!category) return <main><h1>Help category not found</h1></main>
  return <div className="min-h-screen bg-white text-slate-900"><Header /><main>
    <header className="border-b border-slate-200 py-14"><div className="mx-auto max-w-5xl px-4 sm:px-6">
      <nav aria-label="Breadcrumb" className="text-sm text-slate-500"><a href="/docs">Help center</a> / <span aria-current="page">{category.title}</span></nav>
      <h1 className="mt-5 text-4xl font-semibold">{category.title}</h1>
      <p className="mt-4 max-w-3xl text-lg leading-8 text-slate-600">{category.description} Start with the article closest to your goal, check the roles shown on the page, and follow its verification step before expanding a change. The guides apply across Lextures segments, but an organization’s enabled features and permissions determine which controls appear.</p>
      <p className="mt-4 max-w-3xl leading-7 text-slate-600">Use this category as the canonical starting point for related support questions. Each article states when it was last checked, links to adjacent tasks, and points back to the relevant product area. If a screen differs from the guide, record your deployed version and ask an administrator to confirm configuration before proceeding.</p>
    </div></header>
    <section className="py-12"><div className="mx-auto max-w-5xl px-4 sm:px-6"><h2 className="text-2xl font-semibold">All {category.title.toLowerCase()} articles</h2><div className="mt-6 grid gap-4 sm:grid-cols-2">
      {articles.map(article => <article key={article.slug} className="rounded-xl border border-slate-200 p-5"><h3 className="text-lg font-semibold"><a href={articlePath(article)}>{article.title}</a></h3><p className="mt-2 text-sm leading-6 text-slate-600">{article.description}</p></article>)}
    </div><p className="mt-10"><a href={category.platformPath} className="font-semibold text-indigo-700">Explore the related Lextures product area</a></p></div></section>
  </main><SiteFooter /></div>
}
